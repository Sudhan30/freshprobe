# Verdict Interpretation Guide

## Verdict decision logic

freshprobe uses multiple independent signals to compute verdicts. The logic:

1. Each signal (cache headers, fingerprint, liveness, policy threshold) is evaluated
2. Signals that indicate "fresh" are counted as fresh signals
3. The ratio of fresh signals to total signals determines the verdict:
   - 70%+ fresh signals = FRESH
   - 30% or fewer fresh signals = STALE
   - Between 30% and 70% = UNKNOWN

## Confidence scoring

Confidence ranges from 0.0 to 1.0 and reflects how many signals were available:

| Confidence | Meaning |
|-----------|---------|
| 0.0 | No signals available (error, unreachable) |
| 0.3 to 0.5 | Few signals, verdict is uncertain |
| 0.6 to 0.8 | Multiple signals agree, verdict is reliable |
| 0.8 to 1.0 | Strong signal agreement, high reliability |

## Freshness score breakdown

The freshness score (0.0 to 1.0) is computed from HTTP cache headers:

| Score | Meaning |
|-------|---------|
| 0.0 to 0.2 | Data is stale (expired cache, old Last-Modified) |
| 0.3 to 0.4 | Data is aging but may still be usable |
| 0.5 | Unknown (no cache headers present) |
| 0.6 to 0.8 | Data is reasonably fresh |
| 0.9 to 1.0 | Data is very fresh (recent Last-Modified, low Age) |

## Liveness status values

| Status | Condition |
|--------|-----------|
| HEALTHY | 2xx status code, error rate below 50%, not degraded |
| DEGRADED | Healthy but P99 latency exceeds 3x P50, or error rate above 10% |
| UNHEALTHY | 5xx status code or error rate above 50% |

## Common verdict patterns

**FRESH + high confidence**: Endpoint is healthy, data is current. Safe to proceed.

**STALE + high confidence**: Multiple signals confirm staleness. The data age, cache headers,
and/or content fingerprint all indicate old data. Consider fallback sources.

**UNKNOWN + low confidence**: Insufficient data. The endpoint may be fine but doesn't
provide enough metadata (cache headers) for freshprobe to make a determination.
Use `--repeat 3` to add fingerprinting as an additional signal.

**FRESH + policy FAIL**: The data is fresh in absolute terms, but fails a specific
policy threshold (e.g., latency too high for a real-time API). The policy violations
section shows exactly which thresholds were breached.
