package probe

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// ProbeRequest configures what to probe.
type ProbeRequest struct {
	URL            string
	Method         string
	Headers        map[string]string
	Timeout        time.Duration
	RepeatCount    int
	RepeatInterval time.Duration
	SkipTLS        bool
	SkipDNS        bool
}

// ProbeResult holds all signal outputs from a probe run.
type ProbeResult struct {
	URL         string             `json:"url"`
	Timestamp   time.Time          `json:"timestamp"`
	DurationMs  int64              `json:"duration_ms"`
	Cache       *CacheAnalysis     `json:"cache,omitempty"`
	Liveness    *LivenessResult    `json:"liveness,omitempty"`
	Fingerprint *FingerprintResult `json:"fingerprint,omitempty"`
	TLS         *TLSResult         `json:"tls,omitempty"`
	DNS         *DNSResult         `json:"dns,omitempty"`
	Error       string             `json:"error,omitempty"`
}

// EngineOption configures the probe engine.
type EngineOption func(*Engine)

// WithClient sets a custom HTTP client.
func WithClient(c *http.Client) EngineOption {
	return func(e *Engine) {
		e.client = c
	}
}

// WithDefaultTimeout sets the default timeout for probes.
func WithDefaultTimeout(d time.Duration) EngineOption {
	return func(e *Engine) {
		e.defaultTimeout = d
	}
}

// Engine orchestrates all probe signals.
type Engine struct {
	client         *http.Client
	defaultTimeout time.Duration
}

// NewEngine creates a probe engine with the given options.
func NewEngine(opts ...EngineOption) *Engine {
	e := &Engine{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		defaultTimeout: 10 * time.Second,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Probe runs all configured signals against a single URL.
func (e *Engine) Probe(ctx context.Context, req ProbeRequest) (*ProbeResult, error) {
	parsed, err := url.Parse(req.URL)
	if err != nil {
		return nil, err
	}

	if req.Method == "" {
		req.Method = http.MethodGet
	}
	if req.Timeout == 0 {
		req.Timeout = e.defaultTimeout
	}
	if req.RepeatCount < 1 {
		req.RepeatCount = 1
	}

	start := time.Now()
	result := &ProbeResult{
		URL:       req.URL,
		Timestamp: start,
	}

	ctx, cancel := context.WithTimeout(ctx, req.Timeout+30*time.Second)
	defer cancel()

	// Run liveness probe (also captures headers for cache analysis)
	liveness, headers, err := MeasureLiveness(ctx, e.client, req.URL, req.Method, req.Headers, req.RepeatCount)
	if err != nil {
		result.Error = err.Error()
		result.DurationMs = time.Since(start).Milliseconds()
		return result, nil
	}
	result.Liveness = liveness

	// Analyze cache headers from the last response
	if headers != nil {
		result.Cache = AnalyzeCacheHeaders(headers)
	}

	// Content fingerprinting (only if repeat > 1)
	if req.RepeatCount > 1 {
		fp, err := Fingerprint(ctx, e.client, req.URL, req.Method, req.Headers, req.RepeatCount, req.RepeatInterval)
		if err == nil {
			result.Fingerprint = fp
		}
	}

	// TLS check
	if !req.SkipTLS && parsed.Scheme == "https" {
		tlsResult, err := CheckTLS(ctx, parsed.Host)
		if err == nil {
			result.TLS = tlsResult
		}
	}

	// DNS timing
	if !req.SkipDNS {
		dnsResult, err := MeasureDNS(ctx, parsed.Hostname())
		if err == nil {
			result.DNS = dnsResult
		}
	}

	result.DurationMs = time.Since(start).Milliseconds()
	return result, nil
}
