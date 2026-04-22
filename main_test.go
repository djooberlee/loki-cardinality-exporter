package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestScrapeDiscoversLabelsAndCounts verifies the generic scrape flow: fetch /labels,
// then /label/<name>/values for each, and report unique-value counts.
func TestScrapeDiscoversLabelsAndCounts(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/loki/api/v1/labels", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"data":   []string{"job", "host", "__internal__"},
		})
	})
	mux.HandleFunc("/loki/api/v1/label/job/values", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"data":   []string{"loki", "promtail", "docker"},
		})
	})
	mux.HandleFunc("/loki/api/v1/label/host/values", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"data":   []string{"h1", "h2"},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	got, err := scrape(ctx, Datasource{Name: "test", URL: srv.URL}, time.Hour)
	if err != nil {
		t.Fatalf("scrape returned error: %v", err)
	}
	if got["job"] != 3 {
		t.Errorf("want job=3, got %d", got["job"])
	}
	if got["host"] != 2 {
		t.Errorf("want host=2, got %d", got["host"])
	}
	if _, leaked := got["__internal__"]; leaked {
		t.Errorf("internal label __internal__ should be skipped, got it")
	}
}

func TestScrapeBasicAuth(t *testing.T) {
	var gotUser, gotPass string
	var ok bool
	mux := http.NewServeMux()
	mux.HandleFunc("/loki/api/v1/labels", func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPass, ok = r.BasicAuth()
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": []string{}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	_, err := scrape(context.Background(),
		Datasource{Name: "t", URL: srv.URL, BasicAuth: &BasicAuth{Username: "u", Password: "p"}},
		time.Hour)
	if err != nil {
		t.Fatalf("scrape error: %v", err)
	}
	if !ok || gotUser != "u" || gotPass != "p" {
		t.Errorf("basic auth not applied: ok=%v user=%q pass=%q", ok, gotUser, gotPass)
	}
}

func TestScrapeCustomHeaders(t *testing.T) {
	var got string
	mux := http.NewServeMux()
	mux.HandleFunc("/loki/api/v1/labels", func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Scope-OrgID")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": []string{}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	_, err := scrape(context.Background(),
		Datasource{Name: "t", URL: srv.URL, Headers: map[string]string{"X-Scope-OrgID": "tenant-a"}},
		time.Hour)
	if err != nil {
		t.Fatalf("scrape error: %v", err)
	}
	if got != "tenant-a" {
		t.Errorf("header not forwarded, got %q", got)
	}
}

func TestEscapeLabelValue(t *testing.T) {
	cases := map[string]string{
		`plain`:           `plain`,
		`with "quote"`:    `with \"quote\"`,
		`with \backslash`: `with \\backslash`,
		"with\nnewline":   `with\nnewline`,
	}
	for in, want := range cases {
		if got := escapeLabelValue(in); got != want {
			t.Errorf("escapeLabelValue(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestScrapeBadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	_, err := scrape(context.Background(), Datasource{Name: "t", URL: srv.URL}, time.Hour)
	if err == nil || !strings.Contains(err.Error(), "http 500") {
		t.Errorf("want http 500 error, got %v", err)
	}
}
