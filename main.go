// loki-cardinality-exporter — Prometheus exporter for Loki label cardinality.
//
// Scrapes each configured Loki datasource's /loki/api/v1/labels and
// /loki/api/v1/label/<name>/values endpoints, counts unique values per
// label name, and exposes the result as loki_label_cardinality{datasource,label}
// gauge metrics.
//
// Designed as a self-contained binary (stdlib only) so systemd + single
// binary deploy is enough.
package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// Populated at build time via -ldflags "-X main.version=... -X main.commit=... -X main.date=..."
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// ─── Config ────────────────────────────────────────────────────────────

type Config struct {
	Listen         string       `json:"listen"`
	ScrapeInterval string       `json:"scrape_interval"`
	Lookback       string       `json:"lookback"`
	Datasources    []Datasource `json:"datasources"`
}

type Datasource struct {
	Name        string            `json:"name"`
	URL         string            `json:"url"`                  // direct Loki URL OR Grafana proxy URL
	Headers     map[string]string `json:"headers,omitempty"`    // raw headers (e.g. X-Scope-OrgID, Bearer …)
	BasicAuth   *BasicAuth        `json:"basic_auth,omitempty"` // convenience — exporter builds the header
	InsecureTLS bool              `json:"insecure_tls,omitempty"`
}

type BasicAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ─── Metric state ──────────────────────────────────────────────────────

type State struct {
	mu            sync.Mutex
	cardinality   map[string]map[string]int // datasource → label → unique-value-count
	lastScrape    map[string]time.Time
	lastDuration  map[string]time.Duration
	scrapeErrors  map[string]int64
	lastErrorText map[string]string
}

func newState() *State {
	return &State{
		cardinality:   map[string]map[string]int{},
		lastScrape:    map[string]time.Time{},
		lastDuration:  map[string]time.Duration{},
		scrapeErrors:  map[string]int64{},
		lastErrorText: map[string]string{},
	}
}

// ─── Scraping ──────────────────────────────────────────────────────────

// doGET performs an authenticated GET; returns body bytes or error with HTTP status/reason.
func doGET(ctx context.Context, ds Datasource, path string, query url.Values) ([]byte, error) {
	reqURL := strings.TrimRight(ds.URL, "/") + path
	if query != nil {
		reqURL += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range ds.Headers {
		req.Header.Set(k, v)
	}
	if ds.BasicAuth != nil {
		req.SetBasicAuth(ds.BasicAuth.Username, ds.BasicAuth.Password)
	}
	client := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: ds.InsecureTLS},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		limit := len(body)
		if limit > 250 {
			limit = 250
		}
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, body[:limit])
	}
	return body, nil
}

// scrape discovers all label names via /loki/api/v1/labels, then counts unique values
// per label via /loki/api/v1/label/<name>/values. Fully generic — no matcher needed,
// no label names assumed.
func scrape(ctx context.Context, ds Datasource, lookback time.Duration) (map[string]int, error) {
	nowNs := time.Now().UnixNano()
	startNs := nowNs - lookback.Nanoseconds()
	q := url.Values{
		"start": []string{fmt.Sprintf("%d", startNs)},
		"end":   []string{fmt.Sprintf("%d", nowNs)},
	}

	// 1. fetch all label names
	body, err := doGET(ctx, ds, "/loki/api/v1/labels", q)
	if err != nil {
		return nil, fmt.Errorf("labels: %w", err)
	}
	var lr struct {
		Status string   `json:"status"`
		Data   []string `json:"data"`
	}
	if err := json.Unmarshal(body, &lr); err != nil {
		return nil, fmt.Errorf("decode labels json: %w", err)
	}

	// 2. count values per label
	out := make(map[string]int, len(lr.Data))
	for _, lb := range lr.Data {
		if strings.HasPrefix(lb, "__") {
			continue
		}
		vbody, verr := doGET(ctx, ds, "/loki/api/v1/label/"+url.PathEscape(lb)+"/values", q)
		if verr != nil {
			// skip per-label errors; keep going
			continue
		}
		var vr struct {
			Status string   `json:"status"`
			Data   []string `json:"data"`
		}
		if jerr := json.Unmarshal(vbody, &vr); jerr == nil {
			out[lb] = len(vr.Data)
		}
	}
	return out, nil
}

func scrapeAll(cfg *Config, state *State, lookback time.Duration) {
	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	for _, ds := range cfg.Datasources {
		wg.Add(1)
		go func(ds Datasource) {
			defer wg.Done()
			t0 := time.Now()
			cards, err := scrape(ctx, ds, lookback)
			dur := time.Since(t0)

			state.mu.Lock()
			defer state.mu.Unlock()
			state.lastScrape[ds.Name] = time.Now()
			state.lastDuration[ds.Name] = dur
			if err != nil {
				state.scrapeErrors[ds.Name]++
				state.lastErrorText[ds.Name] = err.Error()
				log.Printf("[%s] scrape error (%.2fs): %v", ds.Name, dur.Seconds(), err)
				return
			}
			state.cardinality[ds.Name] = cards
			state.lastErrorText[ds.Name] = ""
			log.Printf("[%s] scraped ok: %d labels, dur=%.2fs",
				ds.Name, len(cards), dur.Seconds())
		}(ds)
	}
	wg.Wait()
}

// ─── HTTP handlers ─────────────────────────────────────────────────────

func escapeLabelValue(s string) string {
	// Prometheus text exposition: escape \ and " and \n
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}

func metricsHandler(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		defer state.mu.Unlock()

		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		fmt.Fprintln(w, "# HELP loki_label_cardinality Unique value count per label, per Loki datasource.")
		fmt.Fprintln(w, "# TYPE loki_label_cardinality gauge")
		dsNames := make([]string, 0, len(state.cardinality))
		for ds := range state.cardinality {
			dsNames = append(dsNames, ds)
		}
		sort.Strings(dsNames)
		for _, ds := range dsNames {
			labels := state.cardinality[ds]
			keys := make([]string, 0, len(labels))
			for k := range labels {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, lb := range keys {
				fmt.Fprintf(w, "loki_label_cardinality{datasource=%q,label=%q} %d\n",
					escapeLabelValue(ds), escapeLabelValue(lb), labels[lb])
			}
		}

		fmt.Fprintln(w, "# HELP loki_label_cardinality_last_scrape_timestamp Unix ts of last scrape per datasource.")
		fmt.Fprintln(w, "# TYPE loki_label_cardinality_last_scrape_timestamp gauge")
		for _, ds := range dsNames {
			if t, ok := state.lastScrape[ds]; ok {
				fmt.Fprintf(w, "loki_label_cardinality_last_scrape_timestamp{datasource=%q} %d\n",
					escapeLabelValue(ds), t.Unix())
			}
		}

		fmt.Fprintln(w, "# HELP loki_label_cardinality_scrape_duration_seconds Duration of last scrape per datasource.")
		fmt.Fprintln(w, "# TYPE loki_label_cardinality_scrape_duration_seconds gauge")
		for _, ds := range dsNames {
			if d, ok := state.lastDuration[ds]; ok {
				fmt.Fprintf(w, "loki_label_cardinality_scrape_duration_seconds{datasource=%q} %f\n",
					escapeLabelValue(ds), d.Seconds())
			}
		}

		fmt.Fprintln(w, "# HELP loki_label_cardinality_scrape_errors_total Cumulative count of failed scrapes per datasource.")
		fmt.Fprintln(w, "# TYPE loki_label_cardinality_scrape_errors_total counter")
		for _, ds := range dsNames {
			fmt.Fprintf(w, "loki_label_cardinality_scrape_errors_total{datasource=%q} %d\n",
				escapeLabelValue(ds), state.scrapeErrors[ds])
		}

		fmt.Fprintln(w, "# HELP loki_label_cardinality_build_info Build metadata.")
		fmt.Fprintln(w, "# TYPE loki_label_cardinality_build_info gauge")
		fmt.Fprintf(w, "loki_label_cardinality_build_info{version=%q,commit=%q,date=%q,goversion=%q} 1\n",
			escapeLabelValue(version), escapeLabelValue(commit), escapeLabelValue(date), escapeLabelValue(runtime.Version()))
	}
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "ok")
}

// ─── main ──────────────────────────────────────────────────────────────

func parseDurOr(s string, def time.Duration) time.Duration {
	if s == "" {
		return def
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		log.Fatalf("invalid duration %q: %v", s, err)
	}
	return d
}

func main() {
	cfgPath := flag.String("config", "/etc/loki-cardinality-exporter/config.json", "path to config JSON")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("loki-cardinality-exporter %s (commit %s, built %s, %s/%s)\n",
			version, commit, date, runtime.GOOS, runtime.GOARCH)
		return
	}

	raw, err := os.ReadFile(*cfgPath)
	if err != nil {
		log.Fatalf("read config %q: %v", *cfgPath, err)
	}
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		log.Fatalf("parse config: %v", err)
	}
	if cfg.Listen == "" {
		cfg.Listen = ":9105"
	}
	interval := parseDurOr(cfg.ScrapeInterval, 5*time.Minute)
	lookback := parseDurOr(cfg.Lookback, 1*time.Hour)

	if len(cfg.Datasources) == 0 {
		log.Fatal("no datasources in config")
	}

	state := newState()
	log.Printf("loki-cardinality-exporter starting: listen=%s scrape=%s lookback=%s datasources=%d",
		cfg.Listen, interval, lookback, len(cfg.Datasources))

	// initial scrape then ticker
	go func() {
		scrapeAll(&cfg, state, lookback)
		t := time.NewTicker(interval)
		defer t.Stop()
		for range t.C {
			scrapeAll(&cfg, state, lookback)
		}
	}()

	http.HandleFunc("/metrics", metricsHandler(state))
	http.HandleFunc("/healthz", healthzHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "loki-cardinality-exporter\nendpoints: /metrics /healthz")
	})

	srv := &http.Server{
		Addr:         cfg.Listen,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}
