# Security policy

## Supported versions

Only the latest tagged release is supported. Security fixes are issued as new
patch releases (e.g. `v1.2.3` → `v1.2.4`).

## Reporting a vulnerability

Please **do not** open a public GitHub issue.

1. Use GitHub's [private vulnerability reporting](https://github.com/djooberlee/loki-cardinality-exporter/security/advisories/new), or
2. Email the maintainer directly (contact info in the repo owner's GitHub profile).

Include:

- Affected version(s)
- A description of the issue
- Reproduction steps or proof-of-concept
- Suggested fix, if any

We will acknowledge receipt within 72 hours and aim to provide a fix or
mitigation within 30 days of confirming the report.

## Scope

- The exporter binary and configuration handling
- Docker image defaults
- Systemd unit security settings

Out of scope: bugs in Loki itself (report to
[grafana/loki](https://github.com/grafana/loki/security)).
