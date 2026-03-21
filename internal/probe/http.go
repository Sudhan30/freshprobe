package probe

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// CacheAnalysis holds parsed cache freshness data from HTTP headers.
type CacheAnalysis struct {
	LastModified   *time.Time `json:"last_modified,omitempty"`
	AgeSeconds     *int       `json:"age_seconds,omitempty"`
	MaxAgeSeconds  *int       `json:"max_age_seconds,omitempty"`
	ETag           string     `json:"etag,omitempty"`
	Date           *time.Time `json:"date,omitempty"`
	Expires        *time.Time `json:"expires,omitempty"`
	CacheControl   string     `json:"cache_control,omitempty"`
	DataAgeSeconds int        `json:"data_age_seconds"`
	FreshnessScore float64    `json:"freshness_score"`
}

// AnalyzeCacheHeaders parses HTTP response headers and computes a freshness score.
// Score is 0.0 (stale) to 1.0 (fresh). 0.5 means unknown/insufficient data.
func AnalyzeCacheHeaders(h http.Header) *CacheAnalysis {
	a := &CacheAnalysis{}
	now := time.Now().UTC()

	// Parse Last-Modified
	if lm := h.Get("Last-Modified"); lm != "" {
		if t, err := http.ParseTime(lm); err == nil {
			a.LastModified = &t
		}
	}

	// Parse Date
	if d := h.Get("Date"); d != "" {
		if t, err := http.ParseTime(d); err == nil {
			a.Date = &t
		}
	}

	// Parse Expires
	if ex := h.Get("Expires"); ex != "" {
		if t, err := http.ParseTime(ex); err == nil {
			a.Expires = &t
		}
	}

	// Parse Age
	if age := h.Get("Age"); age != "" {
		if v, err := strconv.Atoi(age); err == nil {
			a.AgeSeconds = &v
		}
	}

	// Parse ETag
	a.ETag = h.Get("ETag")

	// Parse Cache-Control
	cc := h.Get("Cache-Control")
	a.CacheControl = cc
	if cc != "" {
		a.MaxAgeSeconds = parseCacheControlMaxAge(cc)
	}

	// Compute data age in seconds
	a.DataAgeSeconds = computeDataAge(a, now)

	// Compute freshness score
	a.FreshnessScore = computeFreshnessScore(a, now)

	return a
}

func parseCacheControlMaxAge(cc string) *int {
	for _, directive := range strings.Split(cc, ",") {
		directive = strings.TrimSpace(directive)
		if strings.HasPrefix(directive, "max-age=") {
			valStr := strings.TrimPrefix(directive, "max-age=")
			if v, err := strconv.Atoi(valStr); err == nil {
				return &v
			}
		}
	}
	return nil
}

func computeDataAge(a *CacheAnalysis, now time.Time) int {
	// Best estimate: use Age header directly if available
	if a.AgeSeconds != nil {
		return *a.AgeSeconds
	}

	// Second: compute from Last-Modified
	if a.LastModified != nil {
		age := int(now.Sub(*a.LastModified).Seconds())
		if age < 0 {
			age = 0
		}
		return age
	}

	// Third: compute from Date header
	if a.Date != nil {
		age := int(now.Sub(*a.Date).Seconds())
		if age < 0 {
			age = 0
		}
		return age
	}

	return 0
}

func computeFreshnessScore(a *CacheAnalysis, now time.Time) float64 {
	signals := 0
	totalScore := 0.0

	// Signal 1: Cache-Control max-age vs Age
	if a.MaxAgeSeconds != nil && *a.MaxAgeSeconds > 0 {
		age := 0
		if a.AgeSeconds != nil {
			age = *a.AgeSeconds
		}
		ratio := 1.0 - float64(age)/float64(*a.MaxAgeSeconds)
		if ratio < 0 {
			ratio = 0
		}
		if ratio > 1 {
			ratio = 1
		}
		totalScore += ratio
		signals++
	}

	// Signal 2: Last-Modified recency (newer = fresher)
	if a.LastModified != nil {
		ageSecs := now.Sub(*a.LastModified).Seconds()
		switch {
		case ageSecs < 60:
			totalScore += 1.0
		case ageSecs < 300:
			totalScore += 0.9
		case ageSecs < 3600:
			totalScore += 0.7
		case ageSecs < 86400:
			totalScore += 0.4
		default:
			totalScore += 0.1
		}
		signals++
	}

	// Signal 3: Expires header
	if a.Expires != nil {
		remaining := a.Expires.Sub(now).Seconds()
		if remaining <= 0 {
			totalScore += 0.0
		} else if remaining > 86400 {
			totalScore += 1.0
		} else {
			totalScore += remaining / 86400
		}
		signals++
	}

	// No signals: return 0.5 (unknown)
	if signals == 0 {
		return 0.5
	}

	score := totalScore / float64(signals)
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}
