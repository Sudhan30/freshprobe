# Contributing to freshprobe

Contributions welcome! Here's how to get started.

## Development setup

```bash
git clone https://github.com/Sudhan30/freshprobe.git
cd freshprobe
make build
make test
```

Requirements: Go 1.25+

## Making changes

1. Fork the repo and create a feature branch: `git checkout -b feature/your-feature`
2. Write your code and tests
3. Run `make test` and `make lint`
4. Commit with a descriptive message
5. Open a PR against `main`

## High-value contribution areas

**Policy packs**: Domain-specific freshness policies (healthcare APIs, weather, financial feeds)

**Probe signals**: WebSocket liveness, gRPC health checks, GraphQL introspection freshness

**Integrations**: OpenTelemetry spans, webhook notifications, Slack alerts

**Platform**: Homebrew formula, APT/RPM packages, ARM64 Docker images

## Code style

- Run `gofmt` and `go vet` before committing
- Write table-driven tests
- Keep functions focused and files small
- No external dependencies unless absolutely necessary

## Reporting issues

Open an issue with:
- What you expected
- What happened instead
- Steps to reproduce
- `freshprobe --version` output
