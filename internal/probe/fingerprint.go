package probe

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"time"
)

// FingerprintResult holds content hash data for change detection.
type FingerprintResult struct {
	Hashes      []string `json:"hashes"`
	IsChanging  bool     `json:"is_changing"`
	UniqueCount int      `json:"unique_count"`
}

// Fingerprint hashes response bodies across repeated probes to detect
// whether an endpoint is actually updating or serving static cached content.
func Fingerprint(ctx context.Context, client *http.Client, rawURL, method string, headers map[string]string, count int, interval time.Duration) (*FingerprintResult, error) {
	if count < 2 {
		count = 2
	}
	if interval == 0 {
		interval = 1 * time.Second
	}

	hashes := make([]string, 0, count)
	uniqueSet := make(map[string]struct{})

	for i := 0; i < count; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if i > 0 {
			select {
			case <-time.After(interval):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := client.Do(req)
		if err != nil {
			hashes = append(hashes, "error")
			continue
		}

		h := sha256.New()
		io.Copy(h, resp.Body)
		resp.Body.Close()

		hash := fmt.Sprintf("sha256:%x", h.Sum(nil))
		hashes = append(hashes, hash)
		uniqueSet[hash] = struct{}{}
	}

	return &FingerprintResult{
		Hashes:      hashes,
		IsChanging:  len(uniqueSet) > 1,
		UniqueCount: len(uniqueSet),
	}, nil
}
