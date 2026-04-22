# Final, distroless image. Expects a pre-built binary named `loki-cardinality-exporter`
# in the build context. For dev builds see `make docker` (which runs `make build` first).
# For releases, goreleaser provides the pre-built binary in its Docker build context.
FROM gcr.io/distroless/static-debian12:nonroot

ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

LABEL org.opencontainers.image.title="loki-cardinality-exporter"
LABEL org.opencontainers.image.description="Prometheus exporter for per-label cardinality in Loki"
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.source="https://github.com/djooberlee/loki-cardinality-exporter"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.revision="${COMMIT}"
LABEL org.opencontainers.image.created="${DATE}"

COPY loki-cardinality-exporter /usr/local/bin/loki-cardinality-exporter

EXPOSE 9105
USER nonroot:nonroot

ENTRYPOINT ["/usr/local/bin/loki-cardinality-exporter"]
CMD ["-config", "/etc/loki-cardinality-exporter/config.json"]
