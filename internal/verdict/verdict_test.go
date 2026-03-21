package verdict

import (
	"testing"
	"time"

	"github.com/Sudhan30/freshprobe/internal/policy"
	"github.com/Sudhan30/freshprobe/internal/probe"
)

func TestComputeVerdict_Determinism(t *testing.T) {
	result := &probe.ProbeResult{
		URL:       "https://example.com",
		Timestamp: time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC),
		DurationMs: 100,
		Cache: &probe.CacheAnalysis{
			FreshnessScore: 0.9,
			DataAgeSeconds: 10,
		},
		Liveness: &probe.LivenessResult{
			StatusCode: 200,
			P50Ms:      50,
			P95Ms:      100,
			P99Ms:      120,
			IsHealthy:  true,
			Degraded:   false,
		},
	}

	v1 := ComputeVerdict(result, nil)
	v2 := ComputeVerdict(result, nil)

	if v1.Verdict != v2.Verdict {
		t.Errorf("verdict not deterministic: %s vs %s", v1.Verdict, v2.Verdict)
	}
	if v1.Confidence != v2.Confidence {
		t.Errorf("confidence not deterministic: %f vs %f", v1.Confidence, v2.Confidence)
	}
	if v1.ProbeMetadata.ProbeID != v2.ProbeMetadata.ProbeID {
		t.Errorf("probe ID not deterministic: %s vs %s", v1.ProbeMetadata.ProbeID, v2.ProbeMetadata.ProbeID)
	}
}

func TestComputeVerdict_FreshData(t *testing.T) {
	result := &probe.ProbeResult{
		URL:       "https://example.com",
		Timestamp: time.Now(),
		Cache: &probe.CacheAnalysis{
			FreshnessScore: 0.95,
			DataAgeSeconds: 5,
		},
		Liveness: &probe.LivenessResult{
			StatusCode: 200,
			IsHealthy:  true,
			Degraded:   false,
			P50Ms:      30,
			P95Ms:      50,
			P99Ms:      60,
		},
	}

	v := ComputeVerdict(result, nil)

	if v.Verdict != VerdictFresh {
		t.Errorf("expected FRESH, got %s", v.Verdict)
	}
}

func TestComputeVerdict_StaleData(t *testing.T) {
	result := &probe.ProbeResult{
		URL:       "https://example.com",
		Timestamp: time.Now(),
		Cache: &probe.CacheAnalysis{
			FreshnessScore: 0.1,
			DataAgeSeconds: 7200,
		},
		Liveness: &probe.LivenessResult{
			StatusCode: 200,
			IsHealthy:  true,
			Degraded:   true,
			P50Ms:      100,
			P95Ms:      500,
			P99Ms:      1000,
		},
		Fingerprint: &probe.FingerprintResult{
			IsChanging:  false,
			UniqueCount: 1,
		},
	}

	v := ComputeVerdict(result, nil)

	if v.Verdict != VerdictStale {
		t.Errorf("expected STALE, got %s", v.Verdict)
	}
}

func TestComputeVerdict_WithError(t *testing.T) {
	result := &probe.ProbeResult{
		URL:       "https://example.com",
		Timestamp: time.Now(),
		Error:     "connection refused",
	}

	v := ComputeVerdict(result, nil)

	if v.Verdict != VerdictUnknown {
		t.Errorf("expected UNKNOWN for error, got %s", v.Verdict)
	}
	if v.Confidence != 0.0 {
		t.Errorf("expected 0 confidence for error, got %f", v.Confidence)
	}
}

func TestComputeVerdict_NISTMapping(t *testing.T) {
	result := &probe.ProbeResult{
		URL:       "https://example.com",
		Timestamp: time.Now(),
		Cache:     &probe.CacheAnalysis{FreshnessScore: 0.5},
		Liveness:  &probe.LivenessResult{IsHealthy: true, StatusCode: 200},
	}

	v := ComputeVerdict(result, nil)

	if v.NISTMapping == nil {
		t.Fatal("expected NIST mapping")
	}
	if v.NISTMapping.AIRMFFunction != "MEASURE" {
		t.Errorf("expected MEASURE, got %s", v.NISTMapping.AIRMFFunction)
	}
}

func TestComputeVerdict_WithPolicy(t *testing.T) {
	result := &probe.ProbeResult{
		URL:       "https://api.example.com/data",
		Timestamp: time.Now(),
		Cache: &probe.CacheAnalysis{
			FreshnessScore: 0.3,
			DataAgeSeconds: 120,
		},
		Liveness: &probe.LivenessResult{
			StatusCode: 200,
			IsHealthy:  true,
			P50Ms:      50,
			P95Ms:      600,
			P99Ms:      800,
		},
	}

	rule := &policy.PolicyRule{
		Name:              "test-policy",
		MaxStaleness:      policy.Duration{Duration: 60 * time.Second},
		MinFreshnessScore: 0.8,
		MaxLatencyP95Ms:   500,
	}

	v := ComputeVerdict(result, rule)

	if v.PolicyResult == nil {
		t.Fatal("expected policy result")
	}
	if v.PolicyResult.Passed {
		t.Error("expected policy to fail")
	}
	if len(v.PolicyResult.Violations) < 2 {
		t.Errorf("expected at least 2 violations, got %d", len(v.PolicyResult.Violations))
	}
}
