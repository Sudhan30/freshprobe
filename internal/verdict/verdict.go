package verdict

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/Sudhan30/freshprobe/internal/policy"
	"github.com/Sudhan30/freshprobe/internal/probe"
	"github.com/Sudhan30/freshprobe/internal/version"
)

// Verdict represents the freshness determination.
type Verdict string

const (
	VerdictFresh   Verdict = "FRESH"
	VerdictStale   Verdict = "STALE"
	VerdictUnknown Verdict = "UNKNOWN"
)

// ProbeVerdict is the deterministic output of a freshprobe run.
type ProbeVerdict struct {
	Verdict        Verdict           `json:"verdict"`
	Confidence     float64           `json:"confidence"`
	Endpoint       string            `json:"endpoint"`
	Freshness      *FreshnessSummary `json:"freshness"`
	Liveness       *LivenessSummary  `json:"liveness,omitempty"`
	TLS            *TLSSummary       `json:"tls,omitempty"`
	DNS            *DNSSummary       `json:"dns,omitempty"`
	Fingerprint    *FPSummary        `json:"fingerprint,omitempty"`
	NISTMapping    *NISTMapping      `json:"nist_mapping"`
	PolicyResult   *PolicyEvaluation `json:"policy_result,omitempty"`
	ProbeMetadata  ProbeMetadata     `json:"probe_metadata"`
}

// FreshnessSummary captures cache freshness details.
type FreshnessSummary struct {
	DataAgeSeconds             int     `json:"data_age_seconds"`
	MaxAcceptableSeconds       int     `json:"max_acceptable_seconds,omitempty"`
	CacheControl               string  `json:"cache_control,omitempty"`
	FreshnessScore             float64 `json:"freshness_score"`
	ContentChangedSinceLastProbe *bool `json:"content_changed_since_last_probe,omitempty"`
}

// LivenessSummary captures endpoint health.
type LivenessSummary struct {
	Status     string  `json:"status"`
	StatusCode int     `json:"status_code"`
	P50Ms      int64   `json:"latency_p50_ms"`
	P95Ms      int64   `json:"latency_p95_ms"`
	P99Ms      int64   `json:"latency_p99_ms"`
	ErrorRate  float64 `json:"error_rate"`
}

// TLSSummary captures TLS certificate state.
type TLSSummary struct {
	Valid         bool   `json:"valid"`
	DaysRemaining int    `json:"days_remaining"`
	OCSPStatus    string `json:"ocsp_status,omitempty"`
}

// DNSSummary captures DNS resolution performance.
type DNSSummary struct {
	LatencyMs int64 `json:"latency_ms"`
	IPCount   int   `json:"ip_count"`
}

// FPSummary captures content fingerprint state.
type FPSummary struct {
	IsChanging  bool `json:"is_changing"`
	UniqueCount int  `json:"unique_count"`
}

// NISTMapping references applicable NIST controls.
type NISTMapping struct {
	AIRMFFunction string `json:"ai_rmf_function"`
	Control       string `json:"control"`
}

// ProbeMetadata contains probe execution metadata.
type ProbeMetadata struct {
	ProbeID    string    `json:"probe_id"`
	Timestamp  time.Time `json:"probe_timestamp"`
	DurationMs int64     `json:"duration_ms"`
	Engine     string    `json:"engine"`
}

// ComputeVerdict converts a raw ProbeResult into a deterministic verdict.
// Same inputs always produce the same output.
func ComputeVerdict(result *probe.ProbeResult, rule *policy.PolicyRule) *ProbeVerdict {
	v := &ProbeVerdict{
		Endpoint: result.URL,
		NISTMapping: &NISTMapping{
			AIRMFFunction: "MEASURE",
			Control:       "MS-2.6-001",
		},
		ProbeMetadata: ProbeMetadata{
			ProbeID:    computeProbeID(result),
			Timestamp:  result.Timestamp,
			DurationMs: result.DurationMs,
			Engine:     "freshprobe/" + version.Version,
		},
	}

	// Handle error case
	if result.Error != "" {
		v.Verdict = VerdictUnknown
		v.Confidence = 0.0
		v.Freshness = &FreshnessSummary{FreshnessScore: 0}
		return v
	}

	// Build freshness summary
	v.Freshness = buildFreshnessSummary(result, rule)

	// Build liveness summary
	if result.Liveness != nil {
		status := "HEALTHY"
		if !result.Liveness.IsHealthy {
			status = "UNHEALTHY"
		} else if result.Liveness.Degraded {
			status = "DEGRADED"
		}
		v.Liveness = &LivenessSummary{
			Status:     status,
			StatusCode: result.Liveness.StatusCode,
			P50Ms:      result.Liveness.P50Ms,
			P95Ms:      result.Liveness.P95Ms,
			P99Ms:      result.Liveness.P99Ms,
			ErrorRate:  result.Liveness.ErrorRate,
		}
	}

	// Build TLS summary
	if result.TLS != nil {
		v.TLS = &TLSSummary{
			Valid:         result.TLS.IsValid,
			DaysRemaining: result.TLS.DaysRemaining,
			OCSPStatus:    result.TLS.OCSPStatus,
		}
	}

	// Build DNS summary
	if result.DNS != nil {
		v.DNS = &DNSSummary{
			LatencyMs: result.DNS.LatencyMs,
			IPCount:   len(result.DNS.IPs),
		}
	}

	// Build fingerprint summary
	if result.Fingerprint != nil {
		v.Fingerprint = &FPSummary{
			IsChanging:  result.Fingerprint.IsChanging,
			UniqueCount: result.Fingerprint.UniqueCount,
		}
		changing := result.Fingerprint.IsChanging
		v.Freshness.ContentChangedSinceLastProbe = &changing
	}

	// Compute verdict and confidence
	v.Verdict, v.Confidence = computeVerdictAndConfidence(result, rule)

	// Evaluate policy if provided
	if rule != nil {
		v.PolicyResult = EvaluatePolicy(result, rule)
	}

	return v
}

func buildFreshnessSummary(result *probe.ProbeResult, rule *policy.PolicyRule) *FreshnessSummary {
	fs := &FreshnessSummary{}

	if result.Cache != nil {
		fs.DataAgeSeconds = result.Cache.DataAgeSeconds
		fs.CacheControl = result.Cache.CacheControl
		fs.FreshnessScore = result.Cache.FreshnessScore
	}

	if rule != nil && rule.MaxStaleness.Duration > 0 {
		fs.MaxAcceptableSeconds = int(rule.MaxStaleness.Seconds())
	}

	return fs
}

func computeVerdictAndConfidence(result *probe.ProbeResult, rule *policy.PolicyRule) (Verdict, float64) {
	signals := 0
	freshSignals := 0
	totalConfidence := 0.0

	// Signal 1: Cache freshness score
	if result.Cache != nil {
		signals++
		totalConfidence += 0.9 // high confidence from headers
		if result.Cache.FreshnessScore >= 0.5 {
			freshSignals++
		}
	}

	// Signal 2: Content fingerprint
	if result.Fingerprint != nil {
		signals++
		totalConfidence += 0.8
		if result.Fingerprint.IsChanging {
			freshSignals++
		}
	}

	// Signal 3: Liveness
	if result.Liveness != nil {
		signals++
		totalConfidence += 0.7
		if result.Liveness.IsHealthy && !result.Liveness.Degraded {
			freshSignals++
		}
		// Unhealthy endpoints are never fresh
	}

	// Signal 4: Policy threshold check
	if rule != nil && result.Cache != nil {
		if rule.MaxStaleness.Duration > 0 {
			signals++
			totalConfidence += 0.95
			if time.Duration(result.Cache.DataAgeSeconds)*time.Second <= rule.MaxStaleness.Duration {
				freshSignals++
			}
		}
	}

	if signals == 0 {
		return VerdictUnknown, 0.0
	}

	confidence := totalConfidence / float64(signals) / 1.0 // normalize
	if confidence > 1.0 {
		confidence = 1.0
	}

	freshRatio := float64(freshSignals) / float64(signals)

	if freshRatio >= 0.7 {
		return VerdictFresh, confidence
	}
	if freshRatio <= 0.3 {
		return VerdictStale, confidence
	}
	return VerdictUnknown, confidence * 0.6 // lower confidence for ambiguous
}

func computeProbeID(result *probe.ProbeResult) string {
	data := fmt.Sprintf("%s|%s|%d", result.URL, result.Timestamp.UTC().Format(time.RFC3339Nano), result.DurationMs)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("fp_%x", hash[:8])
}
