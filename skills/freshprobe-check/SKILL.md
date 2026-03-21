---
name: freshprobe-check
description: |
  Use this skill when the user wants to check if an API endpoint or URL is returning fresh data,
  verify endpoint health/liveness, measure latency, check TLS certificate status, or validate
  data freshness against a policy. Also trigger when the user mentions staleness, cache issues,
  or endpoint reliability concerns.

  Examples of when to trigger:
  - "Is this API returning fresh data?"
  - "Check if the trading endpoint is healthy"
  - "How stale is the data from this URL?"
  - "Verify the TLS certificate on this endpoint"
  - "Does this endpoint meet our SLA?"
version: 1.0.0
allowed-tools: [Bash, Read]
---

# Freshprobe: Data Freshness Verification

Use the `freshprobe` CLI to check endpoint freshness and liveness.

## Available Commands

### Single endpoint check
```bash
freshprobe check <url> [--repeat N] [--interval 2s] [--policy <name>] [--policy-dir <path>] [--output text|json] [--stateless]
```

### Batch check
```bash
freshprobe batch <url1> <url2> ... [--concurrency 5] [--stateless]
```

### With policy evaluation
```bash
freshprobe check <url> --policy-dir /path/to/policies --policy <policy-name> --stateless
```

## Built-in Policies

Located at the plugin's policies directory:
- `api-realtime`: max 60s staleness, 0.8 freshness, 500ms P95
- `api-standard`: max 5m staleness, 0.6 freshness, 2000ms P95
- `static-content`: max 24h staleness, 0.3 freshness
- `financial-data`: max 30s staleness, 0.9 freshness, 200ms P95

## Workflow

1. Run `freshprobe check` with the user's URL
2. Use `--output text` for readable summaries, `--output json` for structured data
3. If the user asks about policy compliance, use `--policy` with the appropriate policy name
4. For content change detection, use `--repeat 3 --interval 2s`
5. Present the verdict (FRESH/STALE/UNKNOWN) clearly with key metrics

## Interpreting Results

- **FRESH**: Data is current and endpoint is healthy
- **STALE**: Data age exceeds acceptable thresholds or endpoint is degraded
- **UNKNOWN**: Insufficient signals to determine freshness (missing headers, errors)
- **Confidence**: 0.0 to 1.0, higher means more signals confirmed the verdict
- **Freshness Score**: 0.0 (stale) to 1.0 (fresh), derived from HTTP cache headers
