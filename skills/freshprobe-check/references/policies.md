# Writing Custom Freshprobe Policies

Policies are YAML files that define freshness thresholds for categories of endpoints.
Place them in any directory and point freshprobe at it with `--policy-dir`.

## Policy file format

```yaml
version: "1"
policies:
  policy-name:
    name: "Human Readable Name"
    description: "What this policy covers"
    domains: ["api.*", "*.example.com"]
    path_patterns: ["/v2/.*"]
    max_staleness: "60s"
    min_freshness_score: 0.8
    max_latency_p95_ms: 500
    require_tls: true
    min_tls_days_left: 30
    require_changing: true
```

## Field reference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Display name shown in verdicts |
| `description` | string | No | Human-readable description |
| `domains` | string[] | No | Glob patterns for URL host matching |
| `path_patterns` | string[] | No | Regex patterns for URL path matching |
| `max_staleness` | duration | No | Maximum data age (Go duration: "30s", "5m", "24h") |
| `min_freshness_score` | float | No | Minimum cache freshness score (0.0 to 1.0) |
| `max_latency_p95_ms` | int | No | Maximum P95 latency in milliseconds |
| `require_tls` | bool | No | Require valid TLS certificate |
| `min_tls_days_left` | int | No | Minimum days before TLS cert expiration |
| `require_changing` | bool | No | Content must change between repeat probes |

## Domain matching

Domains use simple glob patterns:
- `api.*` matches `api.example.com`, `api.trading.io`
- `*.example.com` matches `data.example.com`, `cdn.example.com`
- `*` matches everything

## Duration format

Uses Go duration strings:
- `30s` = 30 seconds
- `5m` = 5 minutes
- `2h30m` = 2 hours 30 minutes
- `24h` = 24 hours

## Example: E-commerce platform

```yaml
version: "1"
policies:
  product-catalog:
    name: "Product Catalog"
    description: "Product data can be up to 5 minutes old"
    domains: ["catalog.*", "products.*"]
    max_staleness: "5m"
    min_freshness_score: 0.5
    max_latency_p95_ms: 1000
    require_tls: true

  pricing:
    name: "Pricing API"
    description: "Prices must be real-time"
    domains: ["pricing.*", "*.pricing.*"]
    max_staleness: "10s"
    min_freshness_score: 0.9
    max_latency_p95_ms: 200
    require_tls: true
    require_changing: true

  media-assets:
    name: "Media CDN"
    domains: ["cdn.*", "media.*", "images.*"]
    max_staleness: "24h"
    min_freshness_score: 0.2
    max_latency_p95_ms: 3000
    require_tls: true
    min_tls_days_left: 7
```

## Multiple files

You can split policies across multiple YAML files in the same directory.
freshprobe loads all `.yaml` and `.yml` files from the `--policy-dir`.

## Policy auto-matching

When no `--policy` flag is specified, `freshprobe check` does not auto-match.
Explicit policy names prevent accidental misclassification.
