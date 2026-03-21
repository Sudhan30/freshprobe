package probe

import (
	"net/http"
	"testing"
	"time"
)

func TestAnalyzeCacheHeaders_EmptyHeaders(t *testing.T) {
	h := http.Header{}
	result := AnalyzeCacheHeaders(h)

	if result.FreshnessScore != 0.5 {
		t.Errorf("expected freshness score 0.5 for empty headers, got %f", result.FreshnessScore)
	}
}

func TestAnalyzeCacheHeaders_MaxAgeAndAge(t *testing.T) {
	h := http.Header{}
	h.Set("Cache-Control", "max-age=3600")
	h.Set("Age", "1800")

	result := AnalyzeCacheHeaders(h)

	if result.MaxAgeSeconds == nil || *result.MaxAgeSeconds != 3600 {
		t.Errorf("expected max-age 3600, got %v", result.MaxAgeSeconds)
	}
	if result.AgeSeconds == nil || *result.AgeSeconds != 1800 {
		t.Errorf("expected age 1800, got %v", result.AgeSeconds)
	}
	// 50% of max-age consumed, so freshness score should be ~0.5
	if result.FreshnessScore < 0.4 || result.FreshnessScore > 0.6 {
		t.Errorf("expected freshness score near 0.5, got %f", result.FreshnessScore)
	}
}

func TestAnalyzeCacheHeaders_FreshMaxAge(t *testing.T) {
	h := http.Header{}
	h.Set("Cache-Control", "max-age=3600")
	h.Set("Age", "0")

	result := AnalyzeCacheHeaders(h)

	if result.FreshnessScore < 0.9 {
		t.Errorf("expected high freshness score for age=0, got %f", result.FreshnessScore)
	}
}

func TestAnalyzeCacheHeaders_StaleMaxAge(t *testing.T) {
	h := http.Header{}
	h.Set("Cache-Control", "max-age=100")
	h.Set("Age", "200")

	result := AnalyzeCacheHeaders(h)

	if result.FreshnessScore > 0.1 {
		t.Errorf("expected low freshness score for expired cache, got %f", result.FreshnessScore)
	}
}

func TestAnalyzeCacheHeaders_LastModified(t *testing.T) {
	h := http.Header{}
	h.Set("Last-Modified", time.Now().Add(-30*time.Second).UTC().Format(http.TimeFormat))

	result := AnalyzeCacheHeaders(h)

	if result.LastModified == nil {
		t.Fatal("expected LastModified to be set")
	}
	if result.FreshnessScore < 0.9 {
		t.Errorf("expected high freshness for 30s old data, got %f", result.FreshnessScore)
	}
}

func TestAnalyzeCacheHeaders_ETag(t *testing.T) {
	h := http.Header{}
	h.Set("ETag", `"abc123"`)

	result := AnalyzeCacheHeaders(h)

	if result.ETag != `"abc123"` {
		t.Errorf("expected ETag %q, got %q", `"abc123"`, result.ETag)
	}
}

func TestAnalyzeCacheHeaders_Expires(t *testing.T) {
	h := http.Header{}
	h.Set("Expires", time.Now().Add(12*time.Hour).UTC().Format(http.TimeFormat))

	result := AnalyzeCacheHeaders(h)

	if result.Expires == nil {
		t.Fatal("expected Expires to be set")
	}
	// 12 hours remaining out of 24 hour scale = ~0.5
	if result.FreshnessScore < 0.4 || result.FreshnessScore > 0.6 {
		t.Errorf("expected freshness around 0.5 for 12h remaining, got %f", result.FreshnessScore)
	}
}

func TestParseCacheControlMaxAge(t *testing.T) {
	tests := []struct {
		cc       string
		expected *int
	}{
		{"max-age=3600", intPtr(3600)},
		{"public, max-age=300", intPtr(300)},
		{"no-cache", nil},
		{"", nil},
		{"max-age=0", intPtr(0)},
	}

	for _, tt := range tests {
		got := parseCacheControlMaxAge(tt.cc)
		if tt.expected == nil && got != nil {
			t.Errorf("cc=%q: expected nil, got %d", tt.cc, *got)
		} else if tt.expected != nil && got == nil {
			t.Errorf("cc=%q: expected %d, got nil", tt.cc, *tt.expected)
		} else if tt.expected != nil && got != nil && *tt.expected != *got {
			t.Errorf("cc=%q: expected %d, got %d", tt.cc, *tt.expected, *got)
		}
	}
}

func intPtr(i int) *int { return &i }
