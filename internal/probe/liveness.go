package probe

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"
)

// LivenessResult holds latency and status data from endpoint probing.
type LivenessResult struct {
	StatusCode    int     `json:"status_code"`
	LatencyMs     []int64 `json:"latencies_ms"`
	P50Ms         int64   `json:"p50_ms"`
	P95Ms         int64   `json:"p95_ms"`
	P99Ms         int64   `json:"p99_ms"`
	IsHealthy     bool    `json:"is_healthy"`
	Degraded      bool    `json:"degraded"`
	ErrorRate     float64 `json:"error_rate"`
	BodySizeBytes int64   `json:"body_size_bytes"`
}

// MeasureLiveness probes the URL count times and measures latency percentiles.
// Returns the liveness result and the HTTP headers from the last successful response.
func MeasureLiveness(ctx context.Context, client *http.Client, rawURL, method string, headers map[string]string, count int) (*LivenessResult, http.Header, error) {
	if count < 1 {
		count = 1
	}

	latencies := make([]int64, 0, count)
	var lastHeaders http.Header
	var lastStatus int
	var lastBodySize int64
	errors := 0

	for i := 0; i < count; i++ {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		default:
		}

		req, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("create request: %w", err)
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		start := time.Now()
		resp, err := client.Do(req)
		elapsed := time.Since(start).Milliseconds()

		if err != nil {
			errors++
			latencies = append(latencies, elapsed)
			continue
		}

		bodySize, _ := io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		latencies = append(latencies, elapsed)
		lastStatus = resp.StatusCode
		lastHeaders = resp.Header
		lastBodySize = bodySize
	}

	if len(latencies) == 0 {
		return nil, nil, fmt.Errorf("all %d probes failed", count)
	}

	sorted := make([]int64, len(latencies))
	copy(sorted, latencies)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	result := &LivenessResult{
		StatusCode:    lastStatus,
		LatencyMs:     latencies,
		P50Ms:         percentile(sorted, 0.50),
		P95Ms:         percentile(sorted, 0.95),
		P99Ms:         percentile(sorted, 0.99),
		ErrorRate:     float64(errors) / float64(count),
		BodySizeBytes: lastBodySize,
	}

	// Health assessment
	result.IsHealthy = lastStatus >= 200 && lastStatus < 500 && result.ErrorRate < 0.5
	// Degraded: P99 > 3x P50, or error rate > 10%
	result.Degraded = (result.P50Ms > 0 && result.P99Ms > 3*result.P50Ms) || result.ErrorRate > 0.1

	return result, lastHeaders, nil
}

func percentile(sorted []int64, p float64) int64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(p * float64(len(sorted)-1))
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
