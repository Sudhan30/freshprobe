package probe

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTraceRedirects_NoRedirect(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprintln(w, "ok")
	}))
	defer ts.Close()

	result, err := TraceRedirects(context.Background(), ts.URL, "GET", nil, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasRedirect {
		t.Errorf("expected no redirect, got %d hops", result.TotalHops)
	}
	if result.FinalURL != ts.URL {
		t.Errorf("expected final URL %s, got %s", ts.URL, result.FinalURL)
	}
}

func TestTraceRedirects_SingleRedirect(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprintln(w, "final")
	}))
	defer final.Close()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusMovedPermanently)
	}))
	defer ts.Close()

	result, err := TraceRedirects(context.Background(), ts.URL, "GET", nil, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasRedirect {
		t.Fatal("expected redirect")
	}
	if result.TotalHops != 1 {
		t.Errorf("expected 1 hop, got %d", result.TotalHops)
	}
	if result.FinalURL != final.URL {
		t.Errorf("expected final URL %s, got %s", final.URL, result.FinalURL)
	}
	if result.Hops[0].Status != 301 {
		t.Errorf("expected status 301, got %d", result.Hops[0].Status)
	}
}

func TestTraceRedirects_ChainedRedirects(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer final.Close()

	mid := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusFound)
	}))
	defer mid.Close()

	start := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, mid.URL, http.StatusTemporaryRedirect)
	}))
	defer start.Close()

	result, err := TraceRedirects(context.Background(), start.URL, "GET", nil, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalHops != 2 {
		t.Errorf("expected 2 hops, got %d", result.TotalHops)
	}
	if result.Hops[0].Status != 307 {
		t.Errorf("hop 0: expected 307, got %d", result.Hops[0].Status)
	}
	if result.Hops[1].Status != 302 {
		t.Errorf("hop 1: expected 302, got %d", result.Hops[1].Status)
	}
}
