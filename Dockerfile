# syntax=docker/dockerfile:1.6
FROM --platform=$BUILDPLATFORM golang:1.22-alpine AS builder

ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown
ARG TARGETOS TARGETARCH

WORKDIR /src
COPY go.mod ./
COPY *.go ./

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath \
        -ldflags "-s -w -X main.version=$VERSION -X main.commit=$COMMIT -X main.date=$DATE" \
        -o /out/loki-cardinality-exporter .

# Final image: distroless static, no shell, no package manager
FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.title="loki-cardinality-exporter"
LABEL org.opencontainers.image.description="Prometheus exporter for per-label cardinality in Loki"
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.source="https://github.com/djooberlee/loki-cardinality-exporter"

COPY --from=builder /out/loki-cardinality-exporter /usr/local/bin/loki-cardinality-exporter

EXPOSE 9105
USER nonroot:nonroot

ENTRYPOINT ["/usr/local/bin/loki-cardinality-exporter"]
CMD ["-config", "/etc/loki-cardinality-exporter/config.json"]
