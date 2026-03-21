# freshprobe

**Data freshness and endpoint liveness verification for AI agents.**

freshprobe gives autonomous AI agents a deterministic answer to: *"Is the data I'm about to use current and reliable?"*

It probes endpoints, analyzes HTTP cache headers, measures latency percentiles, fingerprints content for change detection, checks TLS certificate health, and returns structured JSON verdicts that agents can act on programmatically.

Built as a single Go binary. Runs as a CLI tool, an MCP server for AI agent integration, or an HTTP microservice.

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
    "error_rate": 0.03
  },
  "nist_mapping": {
    "ai_rmf_function": "MEASURE",
    "control": "MS-2.6-001"
  }
}
```

## Why

AI agents routinely act on stale data without knowing it. An agent querying a financial API may receive cached data from 47 minutes ago while believing it's real time. No tool currently provides agents with a structured, deterministic assessment of whether external data sources are fresh enough for the action being contemplated.

freshprobe sits between the agent and the external world. Before the agent acts, it asks freshprobe: is this endpoint alive, is this data fresh, does this meet my policy thresholds? The answer is always a deterministic JSON verdict with a clear FRESH / STALE / UNKNOWN signal.

## Features

**Five verification signals in one probe:**

| Signal | What it checks |
|--------|---------------|
| **HTTP cache headers** | Parses `Last-Modified`, `Cache-Control`, `Age`, `ETag`, `Date`, `Expires`. Computes a 0.0 to 1.0 freshness score |
| **Endpoint liveness** | Measures response latency (P50/P95/P99), checks status codes, detects degradation patterns |
| **Content fingerprinting** | SHA-256 hashes response bodies across repeated probes to detect if content is actually updating |
| **TLS certificate freshness** | Checks certificate validity, days remaining, OCSP stapling status |
| **DNS resolution timing** | Measures DNS lookup latency as an infrastructure health signal |

**Three deployment modes:**

| Mode | Command | Use case |
|------|---------|----------|
| **CLI** | `freshprobe check <url>` | Ad hoc checks, scripts, CI/CD |
| **MCP server** | `freshprobe serve --mode mcp` | AI agent integration (Claude, GPT, etc.) |
| **HTTP server** | `freshprobe serve --mode http` | Microservice, team shared endpoint |

**Policy as code:** Define freshness thresholds per domain in YAML. The agent asks "does this endpoint meet my `financial-data` policy?" and gets a pass/fail with specific violations listed.

**NIST AI RMF aligned:** Every verdict includes NIST AI Risk Management Framework mappings for enterprise compliance reporting.

## Install

**Go install (recommended):**

```bash
go install github.com/Sudhan30/freshprobe/cmd/freshprobe@latest
```

**From source:**

```bash
git clone https://github.com/Sudhan30/freshprobe.git
cd freshprobe
make build
./bin/freshprobe --version
```

**Docker:**

```bash
docker pull sudhan03/freshprobe:v0.1.0
docker run sudhan03/freshprobe:v0.1.0 check https://example.com
```

## Quick start

### CLI usage

```bash
# Basic freshness check
freshprobe check https://api.example.com/data

# Text output instead of JSON
freshprobe check https://api.example.com/data --output text

# Content fingerprinting (3 probes, 2s apart)
freshprobe check https://api.example.com/data --repeat 3 --interval 2s

# Check against a policy
freshprobe check https://api.example.com/data \
  --policy-dir ./policies --policy financial-data

# Batch check multiple endpoints
freshprobe batch https://api1.example.com https://api2.example.com https://cdn.example.com

# Stateless mode (no SQLite, no persistence)
freshprobe check https://example.com --stateless
```

### MCP server (for AI agents)

Add to your Claude Code config (`.mcp.json`):

```json
{
  "freshprobe": {
    "type": "stdio",
    "command": "freshprobe",
    "args": ["serve", "--mode", "mcp", "--policy-dir", "/path/to/policies", "--stateless"]
  }
}
```

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

Endpoints:

```
POST /api/v1/check    {"url": "https://..."}
POST /api/v1/batch    {"urls": ["https://...", "https://..."]}
POST /api/v1/policy   {"url": "https://...", "policy_name": "api-realtime"}
GET  /healthz
```

## Policies

Policies define freshness thresholds per domain or endpoint category. Create a YAML file:

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
    min_tls_days_left: 14
```

Policy fields:

| Field | Type | Description |
|-------|------|-------------|
| `max_staleness` | duration | Maximum acceptable data age (`30s`, `5m`, `24h`) |
| `min_freshness_score` | float | Minimum freshness score (0.0 to 1.0) |
| `max_latency_p95_ms` | int | Maximum acceptable P95 latency in milliseconds |
| `require_tls` | bool | Require valid TLS certificate |
| `min_tls_days_left` | int | Minimum days before TLS certificate expires |
| `require_changing` | bool | Content must change between probes (detects stale caches) |

When a probe violates a policy, the verdict includes specific violations:

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

## Output schema

Every probe returns a deterministic JSON verdict:

```json
{
  "verdict": "FRESH | STALE | UNKNOWN",
  "confidence": 0.0-1.0,
  "endpoint": "https://...",
  "freshness": {
    "data_age_seconds": 0,
    "max_acceptable_seconds": 60,
    "cache_control": "max-age=3600",
    "freshness_score": 0.85,
    "content_changed_since_last_probe": true
  },
  "liveness": {
    "status": "HEALTHY | DEGRADED | UNHEALTHY",
    "latency_p50_ms": 45,
    "latency_p95_ms": 120,
    "latency_p99_ms": 340,
    "error_rate": 0.0
  },
  "tls": {
    "valid": true,
    "days_remaining": 149,
    "ocsp_status": "good"
  },
  "dns": {
    "latency_ms": 3,
    "ip_count": 4
  },
  "nist_mapping": {
    "ai_rmf_function": "MEASURE",
    "control": "MS-2.6-001"
  },
  "policy_result": {
    "policy_name": "...",
    "passed": true,
    "violations": []
  },
  "probe_metadata": {
    "probe_id": "fp_a8c3e91d...",
    "probe_timestamp": "2026-03-21T15:49:33Z",
    "duration_ms": 553,
    "engine": "freshprobe/v0.1.0"
  }
}
```

**Determinism contract:** Same probe result + same policy always produces the same verdict. The probe ID is derived from a hash of inputs, not random. This makes verdicts reproducible and auditable.

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

**Stack:**
- Go (single binary, no runtime dependencies)
- `modernc.org/sqlite` (pure Go SQLite, no CGO, easy cross compilation)
- `github.com/modelcontextprotocol/go-sdk` (official MCP SDK, co-maintained with Google)
- `github.com/spf13/cobra` (CLI framework)
- `gopkg.in/yaml.v3` (policy file parsing)

## Claude Code plugin

freshprobe is available as a Claude Code plugin. Once installed, Claude gains the
ability to verify endpoint freshness and data quality on your behalf.

### Install the plugin

```
/plugin install github:Sudhan30/freshprobe
```

### What the plugin provides

| Component | Type | Description |
|-----------|------|-------------|
| `freshprobe` MCP server | MCP (stdio) | Exposes `freshprobe_check`, `freshprobe_batch`, `freshprobe_policy` tools |
| `freshprobe-check` | Skill | Triggers when you ask about data freshness, endpoint health, or API reliability |
| `freshprobe-verifier` | Agent | Autonomous endpoint verification with actionable recommendations |

### Example conversations

After installing the plugin, you can ask Claude things like:

- "Is the trading API at https://api.example.com/quotes returning fresh data?"
- "Check all our endpoints before running the batch job"
- "Does https://api.example.com/data meet our real-time SLA?"
- "The dashboard seems stale, can you check the data source?"

Claude will automatically use the freshprobe skill or agent to probe the endpoints
and report back with a clear verdict and recommendation.

### Prerequisites

The `freshprobe` binary must be installed and in your PATH:

```bash
go install github.com/Sudhan30/freshprobe/cmd/freshprobe@latest
```

If `freshprobe: command not found` appears, add Go's bin directory to your PATH:
```bash
export PATH=$PATH:$HOME/go/bin
```

## Kubernetes deployment

freshprobe ships as a 42MB Docker image. Example K8s deployment:

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
          image: sudhan03/freshprobe:v0.1.0
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

## Development

```bash
# Build
make build

# Run tests
make test

# Lint
make lint

# Cross compile (linux, macOS, Windows)
make cross

# Docker build
make docker
```

### Project structure

```
cmd/freshprobe/main.go          Entry point
internal/
  probe/                        Core probe engine (5 signal modules)
    engine.go                   Orchestrator
    http.go                     Cache header analysis
    liveness.go                 Latency measurement
    fingerprint.go              Content hashing
    tls.go                      Certificate checks
    dns.go                      DNS timing
  verdict/                      Deterministic verdict computation
  policy/                       YAML policy loader and types
  store/                        Storage (SQLite + stateless)
  mcpserver/                    MCP server (stdio transport)
  httpserver/                   HTTP REST server
  cli/                          Cobra CLI commands
policies/                       Default policy pack
```

## Contributing

Contributions welcome. Some areas that would be particularly valuable:

- **Policy packs**: Domain specific freshness policies (healthcare APIs, financial feeds, weather services)
- **Additional probe signals**: WebSocket liveness, gRPC health checks, GraphQL introspection freshness
- **Output integrations**: Prometheus metrics exporter, OpenTelemetry spans, webhook notifications
- **Platform support**: ARM64 Docker images, Homebrew formula, APT/RPM packages

```bash
# Fork, clone, branch
git checkout -b feature/your-feature

# Make changes, test
make test

# Submit PR
```

## License

MIT License. Use it, modify it, ship it, sell it. No restrictions.

```
MIT License

Copyright (c) 2026 Sudhan30

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```
