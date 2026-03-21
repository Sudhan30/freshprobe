package store

import (
	"context"
	"testing"
	"time"

	"github.com/Sudhan30/freshprobe/internal/verdict"
)

func TestSQLiteStore_SaveAndGetProbe(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	v := &verdict.ProbeVerdict{
		Verdict:    verdict.VerdictFresh,
		Confidence: 0.95,
		Endpoint:   "https://example.com",
		Freshness: &verdict.FreshnessSummary{
			DataAgeSeconds: 10,
			FreshnessScore: 0.9,
		},
		ProbeMetadata: verdict.ProbeMetadata{
			ProbeID:   "fp_test123",
			Timestamp: time.Now(),
		},
	}

	if err := store.SaveProbe(ctx, v); err != nil {
		t.Fatalf("SaveProbe: %v", err)
	}

	history, err := store.GetHistory(ctx, "https://example.com", 10)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}

	if len(history) != 1 {
		t.Fatalf("expected 1 result, got %d", len(history))
	}
	if history[0].Verdict != verdict.VerdictFresh {
		t.Errorf("expected FRESH, got %s", history[0].Verdict)
	}
	if history[0].Endpoint != "https://example.com" {
		t.Errorf("expected endpoint https://example.com, got %s", history[0].Endpoint)
	}
}

func TestSQLiteStore_SaveAndGetFingerprint(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	if err := store.SaveFingerprint(ctx, "https://example.com", "sha256:abc123"); err != nil {
		t.Fatalf("SaveFingerprint: %v", err)
	}
	if err := store.SaveFingerprint(ctx, "https://example.com", "sha256:def456"); err != nil {
		t.Fatalf("SaveFingerprint: %v", err)
	}

	hashes, err := store.GetFingerprints(ctx, "https://example.com", 10)
	if err != nil {
		t.Fatalf("GetFingerprints: %v", err)
	}

	if len(hashes) != 2 {
		t.Fatalf("expected 2 fingerprints, got %d", len(hashes))
	}
}

func TestSQLiteStore_GetHistoryEmpty(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	history, err := store.GetHistory(context.Background(), "https://nonexistent.com", 10)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("expected empty history, got %d", len(history))
	}
}
