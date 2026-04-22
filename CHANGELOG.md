# Changelog

All notable changes to this project will be documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial public release scaffolding — LICENSE (Apache 2.0), Dockerfile, Makefile,
  GitHub Actions CI + release (goreleaser), systemd unit, example config.
- `loki_label_cardinality{datasource,label}` metric with discovery of all Loki labels per source.
- Direct Loki URL mode (with optional basic-auth) and Grafana datasource-proxy mode.
- `-version` CLI flag and `loki_label_cardinality_build_info` metric.

[Unreleased]: https://github.com/djooberlee/loki-cardinality-exporter/commits/master
