package httpserver

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
)

// Metrics tracks Prometheus-compatible counters for the HTTP server.
type Metrics struct {
	probesTotal   atomic.Int64
	verdictFresh  atomic.Int64
	verdictStale  atomic.Int64
	verdictUnknown atomic.Int64

	mu            sync.Mutex
	latencySum    float64
	latencyCount  int64
	freshnessSum  float64
	freshnessCount int64
}

var globalMetrics = &Metrics{}

// RecordProbe records a completed probe for metrics.
func RecordProbe(verdict string, latencyP95Ms int64, freshnessScore float64) {
	globalMetrics.probesTotal.Add(1)

	switch verdict {
	case "FRESH":
		globalMetrics.verdictFresh.Add(1)
	case "STALE":
		globalMetrics.verdictStale.Add(1)
	default:
		globalMetrics.verdictUnknown.Add(1)
	}

	globalMetrics.mu.Lock()
	if latencyP95Ms > 0 {
		globalMetrics.latencySum += float64(latencyP95Ms) / 1000.0
		globalMetrics.latencyCount++
	}
	if freshnessScore >= 0 {
		globalMetrics.freshnessSum += freshnessScore
		globalMetrics.freshnessCount++
	}
	globalMetrics.mu.Unlock()
}

func handleMetrics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		m := globalMetrics

		fmt.Fprintf(w, "# HELP freshprobe_probes_total Total number of probes executed.\n")
		fmt.Fprintf(w, "# TYPE freshprobe_probes_total counter\n")
		fmt.Fprintf(w, "freshprobe_probes_total %d\n\n", m.probesTotal.Load())

		fmt.Fprintf(w, "# HELP freshprobe_verdict_total Number of probes by verdict.\n")
		fmt.Fprintf(w, "# TYPE freshprobe_verdict_total counter\n")
		fmt.Fprintf(w, "freshprobe_verdict_total{verdict=\"FRESH\"} %d\n", m.verdictFresh.Load())
		fmt.Fprintf(w, "freshprobe_verdict_total{verdict=\"STALE\"} %d\n", m.verdictStale.Load())
		fmt.Fprintf(w, "freshprobe_verdict_total{verdict=\"UNKNOWN\"} %d\n\n", m.verdictUnknown.Load())

		m.mu.Lock()
		avgLatency := 0.0
		if m.latencyCount > 0 {
			avgLatency = m.latencySum / float64(m.latencyCount)
		}
		avgFreshness := 0.0
		if m.freshnessCount > 0 {
			avgFreshness = m.freshnessSum / float64(m.freshnessCount)
		}
		m.mu.Unlock()

		fmt.Fprintf(w, "# HELP freshprobe_latency_p95_seconds Average P95 latency across probes.\n")
		fmt.Fprintf(w, "# TYPE freshprobe_latency_p95_seconds gauge\n")
		fmt.Fprintf(w, "freshprobe_latency_p95_seconds %.6f\n\n", avgLatency)

		fmt.Fprintf(w, "# HELP freshprobe_freshness_score Average freshness score across probes.\n")
		fmt.Fprintf(w, "# TYPE freshprobe_freshness_score gauge\n")
		fmt.Fprintf(w, "freshprobe_freshness_score %.4f\n", avgFreshness)
	}
}
