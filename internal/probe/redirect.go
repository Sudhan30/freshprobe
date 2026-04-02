package probe

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// RedirectHop records a single HTTP redirect.
type RedirectHop struct {
	URL       string `json:"url"`
	Status    int    `json:"status_code"`
	LatencyMs int64  `json:"latency_ms"`
}

// RedirectResult holds the full redirect chain for a URL.
type RedirectResult struct {
	Hops       []RedirectHop `json:"hops"`
	FinalURL   string        `json:"final_url"`
	TotalHops  int           `json:"total_hops"`
	HasRedirect bool         `json:"has_redirect"`
}

// TraceRedirects follows the redirect chain for a URL without auto-following,
// recording each hop with status code and latency.
func TraceRedirects(ctx context.Context, rawURL, method string, headers map[string]string, maxHops int) (*RedirectResult, error) {
	if maxHops <= 0 {
		maxHops = 10
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	result := &RedirectResult{
		FinalURL: rawURL,
	}

	currentURL := rawURL
	for i := 0; i < maxHops; i++ {
		req, err := http.NewRequestWithContext(ctx, method, currentURL, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		start := time.Now()
		resp, err := client.Do(req)
		elapsed := time.Since(start).Milliseconds()
		if err != nil {
			return nil, fmt.Errorf("request %s: %w", currentURL, err)
		}
		resp.Body.Close()

		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			loc := resp.Header.Get("Location")
			if loc == "" {
				break
			}
			result.Hops = append(result.Hops, RedirectHop{
				URL:       currentURL,
				Status:    resp.StatusCode,
				LatencyMs: elapsed,
			})
			currentURL = loc
			continue
		}

		// Final destination reached
		result.FinalURL = currentURL
		break
	}

	result.TotalHops = len(result.Hops)
	result.HasRedirect = result.TotalHops > 0
	return result, nil
}
