---
name: freshprobe-check
description: >
  This skill should be used when the user asks to "check if an API is fresh",
  "verify endpoint health", "is this data stale", "measure API latency",
  "check TLS certificate status", "validate data freshness", "is this endpoint alive",
  "run a freshness check", "probe this URL", or "check endpoint reliability".
  Also trigger when the user mentions cache staleness, endpoint degradation,
  data age concerns, API health checks, or pre-action data verification.
version: 1.0.0
allowed-tools: [Bash, Read, Glob, Grep]
---

# Freshprobe: Data Freshness Verification

Requires `freshprobe` binary in PATH. Install: `go install github.com/Sudhan30/freshprobe/cmd/freshprobe@latest`

## Commands

**Single check:**
```bash
freshprobe check <url> --stateless --output text
```

**Content change detection:**
```bash
freshprobe check <url> --repeat 3 --interval 2s --stateless
```

**Policy evaluation:**
```bash
freshprobe check <url> --policy-dir ${CLAUDE_PLUGIN_ROOT}/policies --policy <name> --stateless --output text
```

**Batch check:**
```bash
freshprobe batch <url1> <url2> ... --stateless --concurrency 5
```

## Flags

`--output text|json`, `--repeat N`, `--interval 2s`, `--timeout 10s`, `--skip-tls`, `--skip-dns`, `--policy <name>`, `--policy-dir <path>`, `--stateless`

## Verdicts

- **FRESH**: Data is current, endpoint healthy. Safe to proceed.
- **STALE**: Old data or degraded endpoint. Consider retry or fallback.
- **UNKNOWN**: Insufficient signals. Use `--repeat 3` for fingerprinting.
- **Confidence**: 0.0 (no data) to 1.0 (high certainty).

## Built-in Policies

| Policy | Max Staleness | Min Freshness | Max P95 |
|--------|--------------|---------------|---------|
| `api-realtime` | 60s | 0.8 | 500ms |
| `api-standard` | 5m | 0.6 | 2000ms |
| `static-content` | 24h | 0.3 | 2000ms |
| `financial-data` | 30s | 0.9 | 200ms |

## Common Workflows

**"Is this API fresh?"**
Run `freshprobe check <url> --stateless --output text`. Check the verdict line.

**"Check all endpoints before a batch job"**
Run `freshprobe batch <urls> --stateless`. Filter for non-FRESH verdicts.

**"Does this meet our SLA?"**
Run with `--policy-dir` and `--policy`. Check Policy: PASS/FAIL line.

**"Is content actually updating?"**
Run with `--repeat 3 --interval 2s`. Check `content_changed_since_last_probe`.

**"Check TLS certificate"**
Run basic check. The `tls` section shows `valid`, `days_remaining`, `ocsp_status`.

## Troubleshooting

- **"command not found"**: Run `go install github.com/Sudhan30/freshprobe/cmd/freshprobe@latest` and ensure `$HOME/go/bin` is in PATH.
- **"policy not found"**: Error lists available policies. Check `--policy-dir` path.
- **Freshness score 0.5**: Endpoint has no cache headers. Add `--repeat 3` for fingerprinting.

See `references/policies.md` for custom policy authoring and `references/verdicts.md` for detailed verdict interpretation.
