package policy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewLoader_ValidDir(t *testing.T) {
	dir := t.TempDir()
	writeTestPolicy(t, dir, "test.yaml", `
version: "1"
policies:
  my-policy:
    name: "Test Policy"
    max_staleness: "5m"
    min_freshness_score: 0.8
    max_latency_p95_ms: 500
    require_tls: true
`)

	loader, err := NewLoader(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rule, ok := loader.Get("my-policy")
	if !ok {
		t.Fatal("expected to find my-policy")
	}
	if rule.Name != "Test Policy" {
		t.Errorf("expected name %q, got %q", "Test Policy", rule.Name)
	}
	if rule.MaxStaleness.Seconds() != 300 {
		t.Errorf("expected max_staleness 300s, got %v", rule.MaxStaleness.Seconds())
	}
	if rule.MinFreshnessScore != 0.8 {
		t.Errorf("expected min_freshness_score 0.8, got %f", rule.MinFreshnessScore)
	}
}

func TestNewLoader_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	loader, err := NewLoader(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(loader.List()) != 0 {
		t.Errorf("expected 0 policies, got %d", len(loader.List()))
	}
}

func TestNewLoader_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	writeTestPolicy(t, dir, "bad.yaml", `not: valid: yaml: [`)

	_, err := NewLoader(dir)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoader_Match(t *testing.T) {
	dir := t.TempDir()
	writeTestPolicy(t, dir, "test.yaml", `
version: "1"
policies:
  api-policy:
    name: "API Policy"
    domains: ["api.*"]
    max_staleness: "60s"
  cdn-policy:
    name: "CDN Policy"
    domains: ["cdn.*", "static.*"]
    max_staleness: "24h"
`)

	loader, err := NewLoader(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		url      string
		expected string
	}{
		{"https://api.example.com/v1/data", "API Policy"},
		{"https://cdn.example.com/img.png", "CDN Policy"},
		{"https://static.example.com/style.css", "CDN Policy"},
	}

	for _, tt := range tests {
		rule := loader.Match(tt.url)
		if rule == nil {
			t.Errorf("Match(%q): expected policy, got nil", tt.url)
			continue
		}
		if rule.Name != tt.expected {
			t.Errorf("Match(%q): expected %q, got %q", tt.url, tt.expected, rule.Name)
		}
	}
}

func TestLoader_MatchNoMatch(t *testing.T) {
	dir := t.TempDir()
	writeTestPolicy(t, dir, "test.yaml", `
version: "1"
policies:
  api-policy:
    name: "API Policy"
    domains: ["api.*"]
    max_staleness: "60s"
`)

	loader, err := NewLoader(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rule := loader.Match("https://www.example.com")
	if rule != nil {
		t.Errorf("expected no match, got %q", rule.Name)
	}
}

func TestDuration_UnmarshalYAML(t *testing.T) {
	dir := t.TempDir()
	writeTestPolicy(t, dir, "test.yaml", `
version: "1"
policies:
  p1:
    name: "P1"
    max_staleness: "30s"
  p2:
    name: "P2"
    max_staleness: "24h"
  p3:
    name: "P3"
    max_staleness: "5m30s"
`)

	loader, err := NewLoader(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		name     string
		expected float64
	}{
		{"p1", 30},
		{"p2", 86400},
		{"p3", 330},
	}

	for _, tt := range tests {
		rule, ok := loader.Get(tt.name)
		if !ok {
			t.Errorf("policy %q not found", tt.name)
			continue
		}
		if rule.MaxStaleness.Seconds() != tt.expected {
			t.Errorf("policy %q: expected %.0fs, got %.0fs", tt.name, tt.expected, rule.MaxStaleness.Seconds())
		}
	}
}

func writeTestPolicy(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write test policy: %v", err)
	}
}
