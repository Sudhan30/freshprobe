package store

import (
	"context"

	"github.com/Sudhan30/freshprobe/internal/verdict"
)

// Store defines the persistence interface for probe results.
type Store interface {
	SaveProbe(ctx context.Context, v *verdict.ProbeVerdict) error
	GetHistory(ctx context.Context, url string, limit int) ([]*verdict.ProbeVerdict, error)
	SaveFingerprint(ctx context.Context, url string, hash string) error
	GetFingerprints(ctx context.Context, url string, limit int) ([]string, error)
	Close() error
}
