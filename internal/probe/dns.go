package probe

import (
	"context"
	"net"
	"time"
)

// DNSResult holds DNS resolution timing data.
type DNSResult struct {
	Host      string   `json:"host"`
	IPs       []string `json:"ips"`
	LatencyMs int64    `json:"latency_ms"`
}

// MeasureDNS resolves a hostname and measures the time taken.
func MeasureDNS(ctx context.Context, host string) (*DNSResult, error) {
	resolver := &net.Resolver{}

	start := time.Now()
	addrs, err := resolver.LookupHost(ctx, host)
	elapsed := time.Since(start).Milliseconds()

	if err != nil {
		return nil, err
	}

	return &DNSResult{
		Host:      host,
		IPs:       addrs,
		LatencyMs: elapsed,
	}, nil
}
