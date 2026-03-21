package policy

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// PolicyFile represents a YAML policy configuration file.
type PolicyFile struct {
	Version  string                `yaml:"version"`
	Policies map[string]PolicyRule `yaml:"policies"`
}

// PolicyRule defines freshness thresholds for a category of endpoints.
type PolicyRule struct {
	Name              string   `yaml:"name"`
	Description       string   `yaml:"description,omitempty"`
	Domains           []string `yaml:"domains,omitempty"`
	PathPatterns      []string `yaml:"path_patterns,omitempty"`
	MaxStaleness      Duration `yaml:"max_staleness"`
	MinFreshnessScore float64  `yaml:"min_freshness_score"`
	MaxLatencyP95Ms   int64    `yaml:"max_latency_p95_ms"`
	RequireTLS        bool     `yaml:"require_tls"`
	MinTLSDaysLeft    int      `yaml:"min_tls_days_left"`
	RequireChanging   bool     `yaml:"require_changing"`
}

// Duration wraps time.Duration for YAML unmarshaling of strings like "5m", "24h".
type Duration struct {
	time.Duration
}

// UnmarshalYAML parses duration strings.
func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	d.Duration = parsed
	return nil
}

// MarshalYAML outputs the duration as a string.
func (d Duration) MarshalYAML() (interface{}, error) {
	return d.Duration.String(), nil
}
