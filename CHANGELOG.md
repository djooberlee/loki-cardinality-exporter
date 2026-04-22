# Changelog

All notable changes to this project will be documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-04-22

### Added
- New metric `loki_label_cardinality_scrape_success{datasource}` — gauge 0/1 indicating
  whether the last scrape of that datasource succeeded. Easier to alert on than
  the pre-existing cumulative error counter.
- Failed-on-first-scrape datasources now appear in `/metrics` output (previously
  they were absent until at least one scrape had produced cardinality data).

### Changed
- Docker release now publishes both `:X.Y.Z` and `:vX.Y.Z` manifest tags (and `:latest`),
  so users who type the Git tag verbatim get a working image.
- Pin `goreleaser-action` to `~> v2` for reproducible releases.
- Migrate archives config to `formats: [...]` (goreleaser v2 deprecation).
- `Dockerfile` is now single-stage — expects the binary to be provided in the build
  context (done automatically by goreleaser; `make docker` handles it for dev builds).

## [0.1.0] - 2026-04-22

### Added
- Initial public release scaffolding — LICENSE (Apache 2.0), Dockerfile, Makefile,
  GitHub Actions CI + release (goreleaser), systemd unit, example config.
- `loki_label_cardinality{datasource,label}` metric with discovery of all Loki labels per source.
- Direct Loki URL mode (with optional basic-auth) and Grafana datasource-proxy mode.
- `-version` CLI flag and `loki_label_cardinality_build_info` metric.

[Unreleased]: https://github.com/djooberlee/loki-cardinality-exporter/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/djooberlee/loki-cardinality-exporter/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/djooberlee/loki-cardinality-exporter/releases/tag/v0.1.0
