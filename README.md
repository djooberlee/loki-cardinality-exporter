# loki-cardinality-exporter

[![CI](https://github.com/djooberlee/loki-cardinality-exporter/actions/workflows/ci.yml/badge.svg)](https://github.com/djooberlee/loki-cardinality-exporter/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/djooberlee/loki-cardinality-exporter?sort=semver)](https://github.com/djooberlee/loki-cardinality-exporter/releases)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://www.apache.org/licenses/LICENSE-2.0)
[![Go Version](https://img.shields.io/github/go-mod/go-version/djooberlee/loki-cardinality-exporter)](https://go.dev/)

Prometheus exporter that reports **per-label cardinality** for one or more Loki instances.

For each configured Loki source it queries `/loki/api/v1/labels` to discover labels, then queries
`/loki/api/v1/label/<name>/values` to count unique values per label. Result is exposed as
`loki_label_cardinality{datasource,label}` gauge metrics that Grafana/Prometheus alerts can consume.

No label names are assumed — the exporter discovers whatever each Loki instance has.

## Features

- Single static binary, no runtime dependencies (Go stdlib only)
- Supports multiple Loki sources in one process
- Supports **direct Loki HTTP** (with optional basic-auth or custom headers) and **Grafana datasource proxy** modes side-by-side
- `systemd` unit, `Dockerfile` provided
- Pre-built binaries for Linux / macOS / Windows on [Releases](https://github.com/djooberlee/loki-cardinality-exporter/releases)

## Exposed metrics

| Metric | Type | Labels | Description |
|---|---|---|---|
| `loki_label_cardinality` | gauge | `datasource`, `label` | Unique value count per label in last scrape |
| `loki_label_cardinality_last_scrape_timestamp` | gauge | `datasource` | Unix ts of last successful scrape |
| `loki_label_cardinality_scrape_duration_seconds` | gauge | `datasource` | Duration of last scrape |
| `loki_label_cardinality_scrape_errors_total` | counter | `datasource` | Cumulative failed-scrape count |
| `loki_label_cardinality_build_info` | gauge | `version`, `commit`, `date`, `goversion` | Build metadata |

## Quick start

### Docker

```bash
docker run --rm -p 9105:9105 \
  -v $(pwd)/config.json:/etc/loki-cardinality-exporter/config.json:ro \
  ghcr.io/djooberlee/loki-cardinality-exporter:latest
```

### Binary (from Releases)

```bash
curl -sSfL https://github.com/djooberlee/loki-cardinality-exporter/releases/latest/download/loki-cardinality-exporter_Linux_x86_64.tar.gz \
  | tar xz loki-cardinality-exporter
sudo mv loki-cardinality-exporter /usr/local/bin/
```

### Build from source

```bash
git clone https://github.com/djooberlee/loki-cardinality-exporter.git
cd loki-cardinality-exporter
make build
./loki-cardinality-exporter -version
```

## Configuration

JSON file; default path `/etc/loki-cardinality-exporter/config.json`.
See [`config.example.json`](config.example.json) for a full example.

| Key | Type | Default | Description |
|---|---|---|---|
| `listen` | string | `:9105` | HTTP listen address |
| `scrape_interval` | duration | `5m` | How often to scrape Loki |
| `lookback` | duration | `1h` | `start`/`end` window used on `/labels` and `/label/*/values` calls |
| `datasources[]` | array | — | Loki sources to scrape |
| `datasources[].name` | string | — | Becomes `datasource=` label on metrics |
| `datasources[].url` | string | — | Direct Loki URL, or a Grafana datasource-proxy URL |
| `datasources[].headers` | map | `{}` | Raw HTTP headers (e.g. `X-Scope-OrgID`, `Authorization: Bearer …`) |
| `datasources[].basic_auth` | `{username,password}` | — | Convenience — exporter builds `Authorization: Basic …` header |
| `datasources[].insecure_tls` | bool | `false` | Skip TLS verification for this datasource |

### Direct Loki URL

```json
{
  "name": "loki1",
  "url": "https://loki.example.com",
  "basic_auth": { "username": "loki", "password": "secret" }
}
```

Exporter calls `https://loki.example.com/loki/api/v1/labels` etc.

### Via Grafana datasource proxy

Useful when a Loki instance is only reachable as a Grafana datasource (auth handled by Grafana):

```json
{
  "name": "loki1",
  "url": "https://grafana.example.com/api/datasources/proxy/uid/abc123",
  "headers": { "Authorization": "Bearer glsa_..." }
}
```

## Prometheus scrape

```yaml
scrape_configs:
  - job_name: loki-cardinality
    scrape_interval: 60s
    static_configs:
      - targets: ['localhost:9105']
```

## Grafana usage

### Table panel — top labels by cardinality

```promql
topk(15, loki_label_cardinality{datasource="$ds"})
```

Then `Transform → Labels to fields` to display `label` and value as columns.

### Alert rule — cardinality explosion

```yaml
- alert: LokiLabelCardinalityExplosion
  expr: loki_label_cardinality > 50000
  for: 30m
  labels:
    severity: warning
  annotations:
    summary: "Label {{ $labels.label }} on {{ $labels.datasource }} has {{ $value }} unique values"
```

## Systemd deploy

```bash
sudo install -m 755 loki-cardinality-exporter /usr/local/bin/
sudo useradd --system --no-create-home --shell /usr/sbin/nologin loki-exporter
sudo install -m 0750 -o loki-exporter -g loki-exporter -d /etc/loki-cardinality-exporter
sudo install -m 0640 -o loki-exporter -g loki-exporter config.json /etc/loki-cardinality-exporter/
sudo install -m 0644 loki-cardinality-exporter.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now loki-cardinality-exporter
sudo systemctl status loki-cardinality-exporter
```

## Development

```bash
make help        # list targets
make build       # build the binary
make test        # go test ./...
make vet         # go vet ./...
make fmt         # gofmt -s -w .
make lint        # golangci-lint run  (requires golangci-lint)
make run         # go run with local config.json
make docker      # docker build
```

## Releases

Tagged releases are built by [goreleaser](https://goreleaser.com/) via GitHub Actions. Push a `vX.Y.Z` tag to cut:

- Linux / macOS / Windows archives (`amd64`, `arm64`)
- Docker images (`ghcr.io/djooberlee/loki-cardinality-exporter:vX.Y.Z` + `latest`)
- `checksums.txt` with sha256

## Contributing

Pull requests welcome — see [CONTRIBUTING.md](CONTRIBUTING.md).

## Security

If you discover a security issue, see [SECURITY.md](SECURITY.md) — please don't open a public issue.

## License

Apache 2.0 — see [LICENSE](LICENSE).
