package verdict

import (
	"fmt"
	"time"

	"github.com/Sudhan30/freshprobe/internal/policy"
	"github.com/Sudhan30/freshprobe/internal/probe"
)

// PolicyEvaluation holds the result of checking probe data against a policy.
type PolicyEvaluation struct {
	PolicyName string            `json:"policy_name"`
	Passed     bool              `json:"passed"`
	Violations []PolicyViolation `json:"violations,omitempty"`
}

// PolicyViolation describes a specific policy threshold breach.
type PolicyViolation struct {
	Check    string `json:"check"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
}

// EvaluatePolicy checks probe results against policy thresholds.
func EvaluatePolicy(result *probe.ProbeResult, rule *policy.PolicyRule) *PolicyEvaluation {
	eval := &PolicyEvaluation{
		PolicyName: rule.Name,
		Passed:     true,
	}

	// Check max staleness
	if rule.MaxStaleness.Duration > 0 && result.Cache != nil {
		dataAge := time.Duration(result.Cache.DataAgeSeconds) * time.Second
		if dataAge > rule.MaxStaleness.Duration {
			eval.Passed = false
			eval.Violations = append(eval.Violations, PolicyViolation{
				Check:    "max_staleness",
				Expected: fmt.Sprintf("<= %s", rule.MaxStaleness.Duration),
				Actual:   fmt.Sprintf("%s", dataAge),
			})
		}
	}

	// Check min freshness score
	if rule.MinFreshnessScore > 0 && result.Cache != nil {
		if result.Cache.FreshnessScore < rule.MinFreshnessScore {
			eval.Passed = false
			eval.Violations = append(eval.Violations, PolicyViolation{
				Check:    "min_freshness_score",
				Expected: fmt.Sprintf(">= %.2f", rule.MinFreshnessScore),
				Actual:   fmt.Sprintf("%.2f", result.Cache.FreshnessScore),
			})
		}
	}

	// Check max latency P95
	if rule.MaxLatencyP95Ms > 0 && result.Liveness != nil {
		if result.Liveness.P95Ms > rule.MaxLatencyP95Ms {
			eval.Passed = false
			eval.Violations = append(eval.Violations, PolicyViolation{
				Check:    "max_latency_p95",
				Expected: fmt.Sprintf("<= %d ms", rule.MaxLatencyP95Ms),
				Actual:   fmt.Sprintf("%d ms", result.Liveness.P95Ms),
			})
		}
	}

	// Check TLS requirement
	if rule.RequireTLS {
		if result.TLS == nil {
			eval.Passed = false
			eval.Violations = append(eval.Violations, PolicyViolation{
				Check:    "require_tls",
				Expected: "TLS enabled",
				Actual:   "no TLS data",
			})
		} else if !result.TLS.IsValid {
			eval.Passed = false
			eval.Violations = append(eval.Violations, PolicyViolation{
				Check:    "tls_valid",
				Expected: "valid certificate",
				Actual:   "invalid certificate",
			})
		}
	}

	// Check min TLS days remaining
	if rule.MinTLSDaysLeft > 0 && result.TLS != nil {
		if result.TLS.DaysRemaining < rule.MinTLSDaysLeft {
			eval.Passed = false
			eval.Violations = append(eval.Violations, PolicyViolation{
				Check:    "min_tls_days_left",
				Expected: fmt.Sprintf(">= %d days", rule.MinTLSDaysLeft),
				Actual:   fmt.Sprintf("%d days", result.TLS.DaysRemaining),
			})
		}
	}

	// Check content changing requirement
	if rule.RequireChanging && result.Fingerprint != nil {
		if !result.Fingerprint.IsChanging {
			eval.Passed = false
			eval.Violations = append(eval.Violations, PolicyViolation{
				Check:    "require_changing",
				Expected: "content changing between probes",
				Actual:   "static content",
			})
		}
	}

	return eval
}
