package probe

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEngine_BasicProbe(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=3600")
		w.Header().Set("Last-Modified", time.Now().Add(-2*time.Minute).UTC().Format(http.TimeFormat))
		w.WriteHeader(200)
		fmt.Fprintln(w, `{"status": "ok"}`)
	}))
	defer ts.Close()

	engine := NewEngine(WithDefaultTimeout(5 * time.Second))
	result, err := engine.Probe(context.Background(), ProbeRequest{
		URL:     ts.URL,
		SkipTLS: true,
		SkipDNS: true,
	})
	if err != nil {
		t.Fatalf("probe error: %v", err)
	}

	if result.URL != ts.URL {
		t.Errorf("expected URL %s, got %s", ts.URL, result.URL)
	}
	if result.Error != "" {
		t.Errorf("unexpected error: %s", result.Error)
	}
	if result.Liveness == nil {
		t.Fatal("expected liveness result")
	}
	if !result.Liveness.IsHealthy {
		t.Error("expected healthy")
	}
	if result.Cache == nil {
		t.Fatal("expected cache result")
	}
	if result.Cache.FreshnessScore <= 0 {
		t.Errorf("expected positive freshness score, got %f", result.Cache.FreshnessScore)
	}
	if result.DurationMs < 0 {
		t.Error("expected non-negative duration")
	}
}

func TestEngine_UnhealthyEndpoint(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprintln(w, "error")
	}))
	defer ts.Close()

	engine := NewEngine()
	result, err := engine.Probe(context.Background(), ProbeRequest{
		URL:     ts.URL,
		SkipTLS: true,
		SkipDNS: true,
	})
	if err != nil {
		t.Fatalf("probe error: %v", err)
	}
	if result.Liveness == nil {
		t.Fatal("expected liveness result")
	}
	if result.Liveness.IsHealthy {
		t.Error("expected unhealthy for 500 status")
	}
}

func TestEngine_WithRedirect(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprintln(w, "final")
	}))
	defer final.Close()

	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusMovedPermanently)
	}))
	defer redirect.Close()

	engine := NewEngine()
	result, err := engine.Probe(context.Background(), ProbeRequest{
		URL:     redirect.URL,
		SkipTLS: true,
		SkipDNS: true,
	})
	if err != nil {
		t.Fatalf("probe error: %v", err)
	}

	// The Go HTTP client follows redirects by default, so liveness should be healthy
	if result.Liveness == nil {
		t.Fatal("expected liveness result")
	}

	// Redirect tracking should have detected the redirect
	if result.Redirects == nil {
		t.Fatal("expected redirect result")
	}
	if !result.Redirects.HasRedirect {
		t.Error("expected has_redirect=true")
	}
	if result.Redirects.TotalHops < 1 {
		t.Error("expected at least 1 hop")
	}
}

func TestEngine_BodySizeTracked(t *testing.T) {
	body := `{"data": "hello world", "items": [1, 2, 3]}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, body)
	}))
	defer ts.Close()

	engine := NewEngine()
	result, err := engine.Probe(context.Background(), ProbeRequest{
		URL:     ts.URL,
		SkipTLS: true,
		SkipDNS: true,
	})
	if err != nil {
		t.Fatalf("probe error: %v", err)
	}
	if result.Liveness == nil {
		t.Fatal("expected liveness")
	}
	if result.Liveness.BodySizeBytes == 0 {
		t.Error("expected non-zero body size")
	}
}

func TestEngine_Fingerprinting(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprintln(w, "static content")
	}))
	defer ts.Close()

	engine := NewEngine()
	result, err := engine.Probe(context.Background(), ProbeRequest{
		URL:            ts.URL,
		SkipTLS:        true,
		SkipDNS:        true,
		RepeatCount:    3,
		RepeatInterval: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("probe error: %v", err)
	}
	if result.Fingerprint == nil {
		t.Fatal("expected fingerprint result with repeat > 1")
	}
	if result.Fingerprint.IsChanging {
		t.Error("expected static content to not be changing")
	}
}

func TestEngine_SlowEndpoint(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(200)
	}))
	defer ts.Close()

	client := &http.Client{Timeout: 200 * time.Millisecond}
	engine := NewEngine(WithClient(client), WithDefaultTimeout(200*time.Millisecond))
	result, err := engine.Probe(context.Background(), ProbeRequest{
		URL:     ts.URL,
		Timeout: 200 * time.Millisecond,
		SkipTLS: true,
		SkipDNS: true,
	})
	if err != nil {
		t.Fatalf("probe error: %v", err)
	}
	// Should either have error set or be unhealthy due to client timeout
	if result.Error == "" && result.Liveness != nil && result.Liveness.IsHealthy {
		t.Error("expected unhealthy or error for slow endpoint with tight timeout")
	}
}
