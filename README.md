<p align="center">
  <h1 align="center">freshprobe</h1>
  <p align="center">
    <strong>Data freshness verification for AI agents.</strong>
    <br />
    Stop your agents from acting on stale data.
  </p>
  <p align="center">
    <a href="https://github.com/Sudhan30/freshprobe/actions/workflows/ci.yml"><img src="https://github.com/Sudhan30/freshprobe/actions/workflows/ci.yml/badge.svg" alt="CI" /></a>
    <a href="https://goreportcard.com/report/github.com/Sudhan30/freshprobe"><img src="https://goreportcard.com/badge/github.com/Sudhan30/freshprobe" alt="Go Report Card" /></a>
    <a href="https://github.com/Sudhan30/freshprobe/blob/main/LICENSE"><img src="https://img.shields.io/github/license/Sudhan30/freshprobe" alt="License" /></a>
    <a href="https://github.com/Sudhan30/freshprobe/releases"><img src="https://img.shields.io/github/v/release/Sudhan30/freshprobe" alt="Release" /></a>
    <img src="https://img.shields.io/badge/go-1.25-blue" alt="Go Version" />
  </p>
</p>

---

> *"I asked my agent to check flight prices. It gave me options. I booked one. The fare had changed 3 hours ago."*

AI agents routinely act on stale data without knowing it. A financial agent queries cached quotes from 47 minutes ago. A support bot tells a customer their order doesn't exist because the CRM hasn't synced. An RAG pipeline confidently answers with yesterday's docs.

**freshprobe** sits between your agent and the external world. Before the agent acts, it asks: *is this data fresh enough?* The answer is always a deterministic JSON verdict: **FRESH**, **STALE**, or **UNKNOWN**.

```
$ freshprobe check https://api.example.com/v2/quotes

{
  "verdict": "STALE",
  "confidence": 0.94,
  "endpoint": "https://api.example.com/v2/quotes",
  "freshness": {
    "data_age_seconds": 2847,
    "freshness_score": 0.12,
    "cache_control": "max-age=3600"
  },
  "liveness": {
    "status": "DEGRADED",
    "latency_p50_ms": 342,
    "latency_p95_ms": 1847,
    "body_size_bytes": 4096,
    "error_rate": 0.03
  },
  "redirects": {
    "total_hops": 1,
    "final_url": "https://api-v2.example.com/quotes",
    "has_redirect": true
  },
  "nist_mapping": {
    "ai_rmf_function": "MEASURE",
    "control": "MS-2.6-001"
  }
}
```

Single Go binary. No dependencies. Runs as CLI, MCP server, or HTTP microservice.

## Why this matters

| Problem | Cost |
|---------|------|
| E-commerce agent used 6-month-old product data | [$5M+ revenue loss](https://www.informatica.com/) |
| Enterprise RAG with overlapping refresh infrastructure | $340K/year wasted |
| AI project failures from data quality issues | [60%+ of failures (Gartner)](https://www.gartner.com/) |

Unlike crashes that trigger alerts, stale data produces **confident, well-formatted, completely wrong responses**. Chain a few of those in a multi-agent pipeline and every component reports green while the output is catastrophically wrong.

## Install

**Go install (recommended):**

```bash
go install github.com/Sudhan30/freshprobe/cmd/freshprobe@latest
```

**Docker:**

```bash
docker run --rm ghcr.io/sudhan30/freshprobe:latest check https://example.com
```

**From source:**

```bash
git clone https://github.com/Sudhan30/freshprobe.git && cd freshprobe && make build
./bin/freshprobe --version
```

**GitHub Releases:** Download pre-built binaries for Linux, macOS, and Windows from [Releases](https://github.com/Sudhan30/freshprobe/releases).

## Quick start

```bash
# Basic freshness check
freshprobe check https://api.example.com/data

# Human-readable output
freshprobe check https://api.example.com/data --output text

# Content fingerprinting: detect if data actually changes
freshprobe check https://api.example.com/data --repeat 3 --interval 2s

# Check against a freshness policy
freshprobe check https://api.example.com/data --policy-dir ./policies --policy financial-data

# Batch check multiple endpoints
freshprobe batch https://api1.example.com https://api2.example.com https://cdn.example.com

# Continuous monitoring (Ctrl+C to stop)
freshprobe watch https://api.example.com/data --interval 30s --output text

# Only alert on verdict changes (FRESH -> STALE)
freshprobe watch https://api.example.com/data --interval 1m --on-change --output text

# View probe history for an endpoint
freshprobe history https://api.example.com/data --limit 20 --output text
```

## Six verification signals

| Signal | What it checks |
|--------|---------------|
| **HTTP cache headers** | Parses `Last-Modified`, `Cache-Control`, `Age`, `ETag`, `Date`, `Expires`. Computes 0.0 to 1.0 freshness score |
| **Endpoint liveness** | Measures response latency (P50/P95/P99), status codes, body size, degradation patterns |
| **Content fingerprinting** | SHA-256 hashes response bodies across repeated probes to detect stale caches |
| **TLS certificate health** | Certificate validity, days remaining, OCSP stapling status |
| **DNS resolution timing** | DNS lookup latency as infrastructure health signal |
| **Redirect chain analysis** | Tracks 301/302/307/308 hops, detects stale CDN configs |

## Three deployment modes

### CLI

```bash
freshprobe check <url> [flags]
freshprobe batch <urls...> [flags]
freshprobe watch <url> --interval 30s [flags]
freshprobe history <url> --limit 20
```

### MCP server (for AI agents)

Add to your AI tool config:

<details>
<summary><strong>Claude Desktop / Claude Code</strong></summary>

```json
{
  "freshprobe": {
    "type": "stdio",
    "command": "freshprobe",
    "args": ["serve", "--mode", "mcp", "--policy-dir", "/path/to/policies", "--stateless"]
  }
}
```
</details>

<details>
<summary><strong>Cursor</strong></summary>

In `.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "freshprobe": {
      "command": "freshprobe",
      "args": ["serve", "--mode", "mcp", "--stateless"]
    }
  }
}
```
</details>

<details>
<summary><strong>VS Code (Copilot)</strong></summary>

In `.vscode/mcp.json`:

```json
{
  "servers": {
    "freshprobe": {
      "type": "stdio",
      "command": "freshprobe",
      "args": ["serve", "--mode", "mcp", "--stateless"]
    }
  }
}
```
</details>

This exposes three tools to AI agents:

| Tool | Description |
|------|-------------|
| `freshprobe_check` | Probe a single endpoint. Returns JSON verdict |
| `freshprobe_batch` | Probe multiple endpoints concurrently |
| `freshprobe_policy` | Check an endpoint against a named freshness policy |

### HTTP server

```bash
freshprobe serve --mode http --addr :8080
```

```
POST /api/v1/check    {"url": "https://..."}
POST /api/v1/batch    {"urls": ["https://...", "https://..."]}
POST /api/v1/policy   {"url": "https://...", "policy_name": "api-realtime"}
GET  /healthz
GET  /metrics          # Prometheus-compatible metrics
```

## Policies (freshness-as-code)

Define freshness thresholds per domain in YAML:

```yaml
version: "1"
policies:
  financial-data:
    name: "Financial Data"
    domains: ["*.market.*", "*.trading.*"]
    max_staleness: "30s"
    min_freshness_score: 0.9
    max_latency_p95_ms: 200
    require_tls: true
    min_tls_days_left: 30
    require_changing: true

  api-standard:
    name: "Standard API"
    domains: ["api.*"]
    max_staleness: "5m"
    min_freshness_score: 0.6
    max_latency_p95_ms: 2000
    require_tls: true
```

When a probe violates a policy:

```json
{
  "policy_result": {
    "policy_name": "Financial Data",
    "passed": false,
    "violations": [
      {"check": "max_staleness", "expected": "<= 30s", "actual": "2m15s"},
      {"check": "max_latency_p95", "expected": "<= 200 ms", "actual": "847 ms"}
    ]
  }
}
```

Four built-in policies included: `api-realtime`, `api-standard`, `static-content`, `financial-data`.

## Continuous monitoring

```bash
# Watch an endpoint, print every probe
freshprobe watch https://api.example.com/quotes --interval 30s --output text

# Only print when verdict changes (FRESH -> STALE transitions)
freshprobe watch https://api.example.com/quotes --interval 1m --on-change --output text

# Run 10 probes and exit
freshprobe watch https://api.example.com/quotes --count 10 --interval 5s
```

Example output:
```
Watching https://api.example.com/quotes every 30s
[14:22:01] FRESH conf=0.90 score=0.87 p95=142ms
[14:22:31] FRESH conf=0.90 score=0.85 p95=156ms
[14:23:01] STALE conf=0.85 score=0.22 p95=1847ms [FRESH -> STALE]
```

## Prometheus metrics

The HTTP server exposes `/metrics` with Prometheus-compatible text format:

```
freshprobe_probes_total 142
freshprobe_verdict_total{verdict="FRESH"} 98
freshprobe_verdict_total{verdict="STALE"} 31
freshprobe_verdict_total{verdict="UNKNOWN"} 13
freshprobe_latency_p95_seconds 0.234000
freshprobe_freshness_score 0.7200
```

## How it compares

| Feature | freshprobe | Uptime Kuma | Gatus | freshcontext-mcp |
|---------|-----------|-------------|-------|-----------------|
| **Purpose** | Data freshness for AI agents | Uptime monitoring | Health dashboards | Web extraction timestamps |
| **Knows data is stale** | Yes (cache headers + fingerprinting) | No (only checks HTTP status) | No (only checks response assertions) | Partial (timestamps, no verification) |
| **MCP server** | Yes (3 tools) | No | No | Yes |
| **Policy engine** | Yes (YAML, per-domain) | No | Yes (YAML conditions) | No |
| **Continuous monitoring** | Yes (`watch` command) | Yes (dashboard) | Yes (dashboard) | No |
| **Prometheus metrics** | Yes | No (push-based) | Yes | No |
| **Deployment** | Single binary | Docker + DB | Single binary | npm package |

## Architecture

```
                    +------------------+
                    |   freshprobe     |
                    |   single binary  |
                    +--------+---------+
                             |
              +--------------+--------------+
              |              |              |
         +----+----+   +----+----+   +-----+-----+
         |   CLI   |   |   MCP   |   |   HTTP    |
         | (cobra) |   | (stdio) |   | (net/http)|
         +---------+   +---------+   +-----------+
              |              |              |
              +--------------+--------------+
                             |
                    +--------+---------+
                    |   Probe Engine   |
                    |                  |
                    | HTTP headers     |
                    | Latency P50/95/99|
                    | Content SHA-256  |
                    | TLS/OCSP        |
                    | DNS timing       |
                    | Redirect chains  |
                    +--------+---------+
                             |
              +--------------+--------------+
              |              |              |
         +----+----+   +----+----+   +-----+-----+
         | Verdict |   | Policy  |   |   Store   |
         | Engine  |   | Engine  |   | SQLite /  |
         |         |   | (YAML)  |   | Stateless |
         +---------+   +---------+   +-----------+
```

## Kubernetes deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: freshprobe
spec:
  replicas: 1
  selector:
    matchLabels: { app: freshprobe }
  template:
    metadata:
      labels: { app: freshprobe }
    spec:
      containers:
        - name: freshprobe
          image: ghcr.io/sudhan30/freshprobe:latest
          args: ["serve", "--mode", "http", "--addr", ":8080",
                 "--policy-dir", "/etc/freshprobe/policies", "--stateless"]
          ports:
            - containerPort: 8080
          resources:
            requests: { cpu: 50m, memory: 64Mi }
            limits: { cpu: 200m, memory: 128Mi }
          readinessProbe:
            httpGet: { path: /healthz, port: 8080 }
          livenessProbe:
            httpGet: { path: /healthz, port: 8080 }
```

## Claude Code plugin

```
/plugin install github:Sudhan30/freshprobe
```

After installing, ask Claude:
- *"Is the trading API returning fresh data?"*
- *"Check all our endpoints before running the batch job"*
- *"Does this API meet our real-time SLA?"*

## Development

```bash
make build       # Build binary
make test        # Run tests with race detector
make lint        # go vet
make cross       # Cross-compile (linux, macOS, Windows)
make docker      # Docker build
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). High-value areas:
- Policy packs for specific domains (healthcare, weather, finance)
- WebSocket/gRPC/GraphQL probe signals
- OpenTelemetry integration
- Homebrew formula

## License

MIT. See [LICENSE](LICENSE).

---

<p align="center">
  If freshprobe helps your agents make better decisions, <a href="https://github.com/Sudhan30/freshprobe">give it a star</a>.
</p>
