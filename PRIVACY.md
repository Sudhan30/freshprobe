# Privacy Policy

**freshprobe** is an open source data freshness verification tool. This document describes what data freshprobe collects, stores, and transmits.

## What freshprobe does

freshprobe sends HTTP requests to URLs that **you** specify. It analyzes the responses (headers, latency, TLS certificates, content hashes) and produces freshness verdicts. It does not crawl, scrape, or discover URLs on its own.

## Data collection

### What freshprobe collects

| Data | Source | Purpose |
|------|--------|---------|
| HTTP response headers | Endpoints you probe | Cache freshness analysis |
| Response latency | Endpoints you probe | Liveness and degradation detection |
| SHA-256 content hashes | Endpoints you probe | Change detection (fingerprinting) |
| TLS certificate metadata | Endpoints you probe | Certificate validity checks |
| DNS resolution timing | Endpoints you probe | Infrastructure health signal |

### What freshprobe does NOT collect

- No personal information (names, emails, IP addresses, cookies)
- No authentication credentials (tokens, passwords, API keys)
- No request or response bodies are stored (only SHA-256 hashes for fingerprinting)
- No telemetry, analytics, or usage tracking
- No data is sent to Anthropic, the maintainers, or any third party
- No cookies are set or read
- No browser fingerprinting

## Data storage

### Stateless mode (default for MCP and recommended)

When run with `--stateless`, freshprobe stores **nothing**. All probe results exist only in memory for the duration of the request and are discarded immediately after the verdict is returned.

### Persistent mode (SQLite)

When run without `--stateless`, freshprobe stores probe results in a local SQLite database at `~/.freshprobe/data.db` (configurable via `--db`). This database contains:

- Probe verdicts (URL, verdict, confidence, timestamp)
- Content fingerprint hashes (URL, SHA-256 hash, timestamp)

This data never leaves your machine. You can delete it at any time:

```bash
rm -rf ~/.freshprobe/
```

## Network activity

freshprobe makes outbound network requests **only** to endpoints you explicitly specify. It connects to:

- The target URL(s) you provide (HTTP/HTTPS requests)
- DNS resolvers configured on your system (for DNS timing measurements)
- TLS endpoints on port 443 (for certificate chain verification)
- OCSP responders referenced in TLS certificates (for revocation checks)

freshprobe never phones home. There are no update checks, no analytics endpoints, no crash reporting, no telemetry of any kind.

## MCP server mode

When running as an MCP server (stdio transport), freshprobe communicates exclusively with the local AI agent process via stdin/stdout. No network connections are made except to the endpoints the agent requests to probe.

## HTTP server mode

When running as an HTTP microservice, freshprobe listens on the address you specify (`--addr`). It does not expose any endpoints beyond:

- `POST /api/v1/check`
- `POST /api/v1/batch`
- `POST /api/v1/policy`
- `GET /healthz`

No authentication, session management, or user tracking is performed by the HTTP server. If you deploy freshprobe as a shared service, you are responsible for access control (network policies, reverse proxy authentication, etc.).

## Docker and Kubernetes

The Docker image (`sudhan03/freshprobe`) is built from the public Dockerfile in this repository. It is based on `alpine:3.21` with `ca-certificates` added. No additional software, agents, or telemetry is included in the image.

## Third party dependencies

freshprobe's Go dependencies are listed in `go.mod`. None of them collect telemetry or transmit data. Key dependencies:

- `github.com/modelcontextprotocol/go-sdk` (MCP protocol, stdio only)
- `github.com/spf13/cobra` (CLI framework)
- `gopkg.in/yaml.v3` (YAML parsing)
- `modernc.org/sqlite` (pure Go SQLite, local storage only)
- `golang.org/x/crypto` (OCSP response parsing)

## Policy files

Policy YAML files are read from your local filesystem. They are never uploaded, shared, or transmitted anywhere.

## Changes to this policy

Changes to this privacy policy will be committed to this repository with a clear commit message. The git history serves as a changelog.

## Contact

If you have questions about freshprobe's privacy practices, open an issue at https://github.com/Sudhan30/freshprobe/issues.
