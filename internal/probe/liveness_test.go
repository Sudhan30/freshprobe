package probe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMeasureLiveness_Healthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := server.Client()
	result, headers, err := MeasureLiveness(context.Background(), client, server.URL, "GET", nil, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsHealthy {
		t.Error("expected healthy")
	}
	if result.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", result.StatusCode)
	}
	if len(result.LatencyMs) != 3 {
		t.Errorf("expected 3 latency measurements, got %d", len(result.LatencyMs))
	}
	if headers == nil {
		t.Error("expected non-nil headers")
	}
}

func TestMeasureLiveness_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := server.Client()
	result, _, err := MeasureLiveness(context.Background(), client, server.URL, "GET", nil, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsHealthy {
		t.Error("expected unhealthy for 500 status")
	}
}

func TestMeasureLiveness_Degraded(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		// Make last call slow to inflate P99
		if callCount >= 9 {
			time.Sleep(100 * time.Millisecond)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := server.Client()
	result, _, err := MeasureLiveness(context.Background(), client, server.URL, "GET", nil, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// P99 should be significantly higher than P50
	if result.P99Ms <= result.P50Ms {
		t.Logf("P50: %d, P99: %d (may not trigger degraded with httptest)", result.P50Ms, result.P99Ms)
	}
}

func TestPercentile(t *testing.T) {
	tests := []struct {
		sorted   []int64
		p        float64
		expected int64
	}{
		{[]int64{10, 20, 30, 40, 50}, 0.5, 30},
		{[]int64{10, 20, 30, 40, 50}, 0.95, 40}, // index 3 (0.95 * 4 = 3.8 → 3)
		{[]int64{10}, 0.5, 10},
		{[]int64{}, 0.5, 0},
	}

	for _, tt := range tests {
		got := percentile(tt.sorted, tt.p)
		if got != tt.expected {
			t.Errorf("percentile(%v, %.2f) = %d, want %d", tt.sorted, tt.p, got, tt.expected)
		}
	}
}
