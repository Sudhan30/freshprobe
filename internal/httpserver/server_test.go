package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Sudhan30/freshprobe/internal/policy"
	"github.com/Sudhan30/freshprobe/internal/probe"
	"github.com/Sudhan30/freshprobe/internal/store"
	"github.com/Sudhan30/freshprobe/internal/verdict"
)

func startTestServer(t *testing.T) (string, func()) {
	t.Helper()

	engine := probe.NewEngine()
	st := store.NewNoopStore()
	loader := policy.NewEmptyLoader()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, engine, st, loader, addr)
	}()

	// Wait for server to start
	for i := 0; i < 50; i++ {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	return addr, func() {
		cancel()
		<-done
	}
}

func TestHealthz(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	resp, err := http.Get(fmt.Sprintf("http://%s/healthz", addr))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("expected ok, got %s", body["status"])
	}
}

func TestCheckEndpoint(t *testing.T) {
	// Create a target endpoint to probe
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=3600")
		w.Header().Set("Last-Modified", time.Now().Add(-5*time.Minute).UTC().Format(http.TimeFormat))
		w.WriteHeader(200)
		fmt.Fprintln(w, `{"data": "fresh"}`)
	}))
	defer target.Close()

	addr, cleanup := startTestServer(t)
	defer cleanup()

	reqBody, _ := json.Marshal(map[string]any{
		"url":      target.URL,
		"skip_tls": true,
		"skip_dns": true,
	})

	resp, err := http.Post(fmt.Sprintf("http://%s/api/v1/check", addr), "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	var v verdict.ProbeVerdict
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if v.Verdict != verdict.VerdictFresh && v.Verdict != verdict.VerdictUnknown {
		t.Errorf("expected FRESH or UNKNOWN for healthy endpoint, got %s", v.Verdict)
	}
	if v.Endpoint != target.URL {
		t.Errorf("expected endpoint %s, got %s", target.URL, v.Endpoint)
	}
	if v.Liveness == nil {
		t.Error("expected liveness data")
	}
	if v.Freshness == nil {
		t.Error("expected freshness data")
	}
	if v.NISTMapping == nil {
		t.Error("expected NIST mapping")
	}
}

func TestBatchEndpoint(t *testing.T) {
	target1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprintln(w, "ok1")
	}))
	defer target1.Close()

	target2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprintln(w, "ok2")
	}))
	defer target2.Close()

	addr, cleanup := startTestServer(t)
	defer cleanup()

	reqBody, _ := json.Marshal(map[string]any{
		"urls":        []string{target1.URL, target2.URL},
		"concurrency": 2,
	})

	resp, err := http.Post(fmt.Sprintf("http://%s/api/v1/batch", addr), "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	var results []verdict.ProbeVerdict
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestCheckEndpoint_BadRequest(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	// Missing URL
	reqBody, _ := json.Marshal(map[string]any{})

	resp, err := http.Post(fmt.Sprintf("http://%s/api/v1/check", addr), "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	resp, err := http.Get(fmt.Sprintf("http://%s/metrics", addr))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	text := string(body)

	if !bytes.Contains(body, []byte("freshprobe_probes_total")) {
		t.Errorf("expected freshprobe_probes_total in metrics, got: %s", text)
	}
	if !bytes.Contains(body, []byte("freshprobe_verdict_total")) {
		t.Errorf("expected freshprobe_verdict_total in metrics, got: %s", text)
	}
	if !bytes.Contains(body, []byte("freshprobe_freshness_score")) {
		t.Errorf("expected freshprobe_freshness_score in metrics, got: %s", text)
	}
}

func TestCheckThenMetrics(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprintln(w, "ok")
	}))
	defer target.Close()

	addr, cleanup := startTestServer(t)
	defer cleanup()

	// Do a probe first
	reqBody, _ := json.Marshal(map[string]any{"url": target.URL})
	resp, err := http.Post(fmt.Sprintf("http://%s/api/v1/check", addr), "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	resp.Body.Close()

	// Then check metrics show the probe
	resp, err = http.Get(fmt.Sprintf("http://%s/metrics", addr))
	if err != nil {
		t.Fatalf("metrics: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	// Metrics are global and accumulate across tests, just verify the count is nonzero
	if bytes.Contains(body, []byte("freshprobe_probes_total 0")) {
		t.Errorf("expected probes_total > 0, got: %s", string(body))
	}
}
