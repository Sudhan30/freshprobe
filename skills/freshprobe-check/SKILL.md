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

# Freshprobe: Data Freshness and Endpoint Liveness Verification

freshprobe is a CLI tool and MCP server that probes external endpoints and returns
deterministic FRESH/STALE/UNKNOWN verdicts. It helps AI agents and developers verify
that external data sources are current and reliable before acting on them.

## Prerequisites

The `freshprobe` binary must be installed and available in PATH.

Install via Go:
```bash
go install github.com/Sudhan30/freshprobe/cmd/freshprobe@latest
```

Or download pre-built binaries from https://github.com/Sudhan30/freshprobe/releases

Verify installation:
```bash
freshprobe --version
```

If the command is not found, ensure `$HOME/go/bin` is in your PATH:
```bash
export PATH=$PATH:$HOME/go/bin
```

## Core Commands

### Single endpoint check

The most common operation. Probes a single URL and returns a freshness verdict.

```bash
freshprobe check <url> [flags]
```

**Flags:**
| Flag | Default | Description |
|------|---------|-------------|
| `--output` | `json` | Output format: `json` or `text` |
| `--repeat` | `1` | Number of repeat probes for content fingerprinting |
| `--interval` | `2s` | Interval between repeat probes |
| `--timeout` | `10s` | Probe timeout |
| `--skip-tls` | `false` | Skip TLS certificate checks |
| `--skip-dns` | `false` | Skip DNS resolution timing |
| `--policy` | | Policy name to evaluate against |
| `--policy-dir` | | Directory containing policy YAML files |
| `--stateless` | `false` | Run without SQLite persistence |
| `--db` | `~/.freshprobe/data.db` | SQLite database path |

**Example: Quick health check**
```bash
freshprobe check https://api.example.com/v2/data --stateless --output text
```

**Example: Content change detection**
```bash
freshprobe check https://api.example.com/quotes --repeat 3 --interval 2s --stateless
```

**Example: Policy evaluation**
```bash
freshprobe check https://api.example.com/data \
  --policy-dir ${CLAUDE_PLUGIN_ROOT}/policies \
  --policy financial-data \
  --stateless --output text
```

### Batch check

Probes multiple endpoints concurrently. Useful for checking all APIs before a workflow.

```bash
freshprobe batch <url1> <url2> ... [flags]
```

**Flags:**
| Flag | Default | Description |
|------|---------|-------------|
| `--concurrency` | `5` | Maximum concurrent probes |
| `--repeat` | `1` | Number of repeat probes per URL |
| `--timeout` | `10s` | Probe timeout per URL |
| `--stateless` | `false` | Run without persistence |

**Example:**
```bash
freshprobe batch \
  https://api.example.com/data \
  https://cdn.example.com/assets \
  https://auth.example.com/health \
  --stateless --concurrency 3
```

## Understanding Verdicts

Every probe returns one of three verdicts:

### FRESH
The endpoint is healthy and serving current data. All freshness signals are positive.
Confidence is typically 0.7 to 1.0. Safe to proceed with using data from this endpoint.

### STALE
The endpoint is returning old data, is degraded, or fails freshness thresholds.
The `freshness` section of the output shows the data age and cache headers.
Consider retrying, using a fallback source, or alerting the user.

### UNKNOWN
Insufficient signals to make a determination. This happens when:
- The endpoint returns no cache headers (no Last-Modified, no Cache-Control)
- The endpoint is unreachable or returns errors
- Only partial data is available

In UNKNOWN cases, check the `confidence` field. A confidence of 0.0 means a total failure.
A confidence of 0.3 to 0.5 means some signals were available but inconclusive.

## Five Probe Signals

freshprobe analyzes five independent signals from each probe:

### 1. HTTP cache headers
Parses `Last-Modified`, `Cache-Control`, `Age`, `ETag`, `Date`, and `Expires` headers.
Computes a freshness score from 0.0 (stale) to 1.0 (fresh). If no cache headers are
present, the score defaults to 0.5 (unknown).

### 2. Endpoint liveness
Measures response latency across repeat probes. Reports P50, P95, and P99 percentiles.
Marks the endpoint as DEGRADED when P99 exceeds 3x P50, indicating inconsistent performance.
Status codes 2xx are healthy, 5xx are unhealthy.

### 3. Content fingerprinting
When `--repeat` is greater than 1, freshprobe SHA-256 hashes each response body and
compares them. If all hashes are identical, the content is static (potentially cached).
If hashes differ, the content is actively updating. This catches endpoints that return
200 OK with stale cached content.

### 4. TLS certificate freshness
Connects to the TLS endpoint and inspects the leaf certificate. Reports validity dates,
days remaining until expiration, issuer, and OCSP stapling status. Certificates with
fewer than 14 days remaining are flagged. Revoked certificates (via OCSP) mark the
endpoint as invalid.

### 5. DNS resolution timing
Measures how long DNS resolution takes for the hostname. High DNS latency (above 100ms)
can indicate infrastructure issues. Also reports the number of resolved IP addresses.

## Policy Evaluation

Policies define freshness thresholds in YAML. When a probe is evaluated against a policy,
the verdict includes a `policy_result` with pass/fail and specific violations.

### Built-in policies

The plugin ships with four policies in the `policies/` directory:

| Policy | Max Staleness | Min Freshness | Max P95 | Requires TLS | Requires Change |
|--------|--------------|---------------|---------|-------------|-----------------|
| `api-realtime` | 60s | 0.8 | 500ms | Yes | Yes |
| `api-standard` | 5m | 0.6 | 2000ms | Yes | No |
| `static-content` | 24h | 0.3 | 2000ms | Yes | No |
| `financial-data` | 30s | 0.9 | 200ms | Yes | Yes |

### Using policies

```bash
# Use a built-in policy
freshprobe check https://api.trading.com/quotes \
  --policy-dir ${CLAUDE_PLUGIN_ROOT}/policies \
  --policy financial-data \
  --stateless

# Use custom policies from a local directory
freshprobe check https://api.example.com/data \
  --policy-dir /path/to/my/policies \
  --policy my-custom-policy
```

### Writing custom policies

See `references/policies.md` for the full policy YAML schema and examples.

## NIST AI RMF Mapping

Every verdict includes a `nist_mapping` field referencing the applicable NIST AI Risk
Management Framework function and control. This is advisory metadata for compliance
reporting in regulated industries:

- **AI RMF Function**: MEASURE (data quality verification before agent action)
- **Control**: MS-2.6-001 (information system monitoring)

## Workflow for Common Tasks

### "Is this API returning fresh data?"
1. Run `freshprobe check <url> --stateless --output text`
2. Check the verdict line: FRESH means current, STALE means old data
3. If STALE, check `Data Age` and `Freshness` score for details

### "Check all our endpoints before a batch job"
1. Run `freshprobe batch <url1> <url2> ... --stateless`
2. Parse the JSON array, filter for any verdict that is not FRESH
3. Report failing endpoints to the user before proceeding

### "Does this meet our SLA/policy?"
1. Run `freshprobe check <url> --policy-dir <dir> --policy <name> --stateless --output text`
2. Check the `Policy:` line for PASS or FAIL
3. If FAIL, violations list shows exactly what thresholds were breached

### "Is the content actually changing or serving cached?"
1. Run `freshprobe check <url> --repeat 3 --interval 2s --stateless`
2. Check `content_changed_since_last_probe` in the JSON output
3. If false, the endpoint may be serving stale cached responses despite returning 200

### "Check TLS certificate health"
1. Run `freshprobe check <url> --stateless`
2. Check the `tls` section: `valid`, `days_remaining`, `ocsp_status`
3. Flag if `days_remaining` is below 30 (approaching expiration)

## Error Handling

- **Invalid URL**: Returns UNKNOWN verdict with confidence 0.0. No crash.
- **Unreachable host**: Returns UNKNOWN verdict. Check the `liveness.status` field.
- **Missing policy**: Returns a helpful error listing available policy names.
- **Timeout**: Configurable via `--timeout`. Default 10 seconds.
- **TLS errors**: If `--skip-tls` is set, TLS checks are skipped. Otherwise, invalid
  certificates are reported in the verdict without causing a probe failure.

## Troubleshooting

### "freshprobe: command not found"
Install the binary: `go install github.com/Sudhan30/freshprobe/cmd/freshprobe@latest`
Then ensure `$HOME/go/bin` is in your PATH.

### "policy not found"
List available policies: the error message shows all loaded policy names.
Check that `--policy-dir` points to a directory containing `.yaml` files.

### "all probes failed"
The endpoint is completely unreachable. Check your network connection and the URL.
Try with `--timeout 30s` if the endpoint is slow to respond.

### Freshness score always 0.5
The endpoint returns no cache headers (no Last-Modified, Cache-Control, etc.).
A score of 0.5 means "unknown" rather than "fresh" or "stale".
Use `--repeat 3` to add content fingerprinting as an additional freshness signal.
