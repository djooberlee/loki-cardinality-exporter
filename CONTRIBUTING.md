# Contributing

Thanks for your interest in contributing!

## Reporting bugs

Open a [GitHub issue](https://github.com/djooberlee/loki-cardinality-exporter/issues/new/choose)
using the bug-report template. Please include:

- Exporter version (`loki-cardinality-exporter -version`)
- Loki version(s) being scraped
- Relevant config snippet (redact secrets)
- Logs / metrics output illustrating the issue

## Proposing changes

1. Open an issue first for non-trivial changes so the design can be discussed.
2. Fork the repository and create a topic branch.
3. Run the checks locally:
   ```bash
   make fmt vet test
   make lint        # optional, requires golangci-lint
   make build
   ```
4. Add / update tests when you change behavior.
5. Update `CHANGELOG.md` under `[Unreleased]`.
6. Open a PR referencing the issue.

## Commit messages

Short imperative summary line, followed by a blank line and, optionally, a body
explaining the *why*. Conventional-commits prefixes (`feat:`, `fix:`, `docs:`)
are welcome but not required.

## Code style

- `gofmt -s` enforced by CI.
- No third-party runtime dependencies — the binary stays stdlib-only. Test-only
  dependencies are fine.
- Keep the exporter hands-off: it must not make assumptions about which labels
  exist in a given Loki. Always discover via the API.

## Licensing

By submitting a contribution, you agree your work is licensed under Apache 2.0
(see [LICENSE](LICENSE)).
