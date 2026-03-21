package store

import (
	"context"

	"github.com/Sudhan30/freshprobe/internal/verdict"
)

// NoopStore is a stateless store that discards all data.
type NoopStore struct{}

func NewNoopStore() *NoopStore { return &NoopStore{} }

func (n *NoopStore) SaveProbe(_ context.Context, _ *verdict.ProbeVerdict) error { return nil }
func (n *NoopStore) GetHistory(_ context.Context, _ string, _ int) ([]*verdict.ProbeVerdict, error) {
	return nil, nil
}
func (n *NoopStore) SaveFingerprint(_ context.Context, _, _ string) error { return nil }
func (n *NoopStore) GetFingerprints(_ context.Context, _ string, _ int) ([]string, error) {
	return nil, nil
}
func (n *NoopStore) Close() error { return nil }
