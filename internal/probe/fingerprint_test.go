package probe

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestFingerprint_StaticContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("static content that never changes"))
	}))
	defer server.Close()

	client := server.Client()
	result, err := Fingerprint(context.Background(), client, server.URL, "GET", nil, 3, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsChanging {
		t.Error("expected static content to not be changing")
	}
	if result.UniqueCount != 1 {
		t.Errorf("expected 1 unique hash, got %d", result.UniqueCount)
	}
	if len(result.Hashes) != 3 {
		t.Errorf("expected 3 hashes, got %d", len(result.Hashes))
	}
}

func TestFingerprint_ChangingContent(t *testing.T) {
	var counter atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := counter.Add(1)
		w.Write([]byte(fmt.Sprintf("content version %d", n)))
	}))
	defer server.Close()

	client := server.Client()
	result, err := Fingerprint(context.Background(), client, server.URL, "GET", nil, 3, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsChanging {
		t.Error("expected changing content")
	}
	if result.UniqueCount != 3 {
		t.Errorf("expected 3 unique hashes, got %d", result.UniqueCount)
	}
}
