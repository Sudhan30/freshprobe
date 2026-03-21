package policy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Loader manages policy rules loaded from YAML files.
type Loader struct {
	policies map[string]PolicyRule
}

// NewLoader loads all .yaml/.yml files from the given directories.
func NewLoader(dirs ...string) (*Loader, error) {
	l := &Loader{
		policies: make(map[string]PolicyRule),
	}

	for _, dir := range dirs {
		if err := l.loadDir(dir); err != nil {
			return nil, fmt.Errorf("load policy dir %s: %w", dir, err)
		}
	}

	return l, nil
}

// NewEmptyLoader creates a loader with no policies.
func NewEmptyLoader() *Loader {
	return &Loader{policies: make(map[string]PolicyRule)}
}

func (l *Loader) loadDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		var pf PolicyFile
		if err := yaml.Unmarshal(data, &pf); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}

		for name, rule := range pf.Policies {
			if rule.Name == "" {
				rule.Name = name
			}
			l.policies[name] = rule
		}
	}

	return nil
}

// Get returns a policy by name.
func (l *Loader) Get(name string) (*PolicyRule, bool) {
	p, ok := l.policies[name]
	if !ok {
		return nil, false
	}
	return &p, true
}

// Match finds the first policy whose domain patterns match the given URL.
func (l *Loader) Match(rawURL string) *PolicyRule {
	for _, rule := range l.policies {
		for _, pattern := range rule.Domains {
			if matchDomain(rawURL, pattern) {
				return &rule
			}
		}
	}
	return nil
}

// List returns all loaded policy names.
func (l *Loader) List() []string {
	names := make([]string, 0, len(l.policies))
	for name := range l.policies {
		names = append(names, name)
	}
	return names
}

// matchDomain checks if a URL's host matches a glob pattern.
// Supports simple prefix wildcards like "api.*" or "*.example.com".
func matchDomain(rawURL, pattern string) bool {
	// Extract host from URL
	host := rawURL
	if idx := strings.Index(host, "://"); idx != -1 {
		host = host[idx+3:]
	}
	if idx := strings.Index(host, "/"); idx != -1 {
		host = host[:idx]
	}
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Simple glob matching
	if pattern == "*" {
		return true
	}
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:] // ".example.com"
		return strings.HasSuffix(host, suffix)
	}
	if strings.HasSuffix(pattern, ".*") {
		prefix := pattern[:len(pattern)-2] // "api"
		return strings.HasPrefix(host, prefix+".")
	}

	matched, _ := filepath.Match(pattern, host)
	return matched
}
