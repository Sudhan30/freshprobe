package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Sudhan30/freshprobe/internal/policy"
	"github.com/Sudhan30/freshprobe/internal/probe"
	"github.com/Sudhan30/freshprobe/internal/store"
	"github.com/Sudhan30/freshprobe/internal/verdict"
)

type checkRequest struct {
	URL         string `json:"url"`
	Repeat      int    `json:"repeat,omitempty"`
	TimeoutSecs int    `json:"timeout_secs,omitempty"`
	SkipTLS     bool   `json:"skip_tls,omitempty"`
	SkipDNS     bool   `json:"skip_dns,omitempty"`
}

type batchRequest struct {
	URLs        []string `json:"urls"`
	Repeat      int      `json:"repeat,omitempty"`
	Concurrency int      `json:"concurrency,omitempty"`
}

type policyRequest struct {
	URL        string `json:"url"`
	PolicyName string `json:"policy_name"`
}

func handleCheck(engine *probe.Engine, st store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req checkRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.URL == "" {
			httpError(w, "url is required", http.StatusBadRequest)
			return
		}

		timeout := 10 * time.Second
		if req.TimeoutSecs > 0 {
			timeout = time.Duration(req.TimeoutSecs) * time.Second
		}
		repeat := req.Repeat
		if repeat < 1 {
			repeat = 1
		}

		probeReq := probe.ProbeRequest{
			URL:         req.URL,
			Timeout:     timeout,
			RepeatCount: repeat,
			SkipTLS:     req.SkipTLS,
			SkipDNS:     req.SkipDNS,
		}

		result, err := engine.Probe(r.Context(), probeReq)
		if err != nil {
			httpError(w, fmt.Sprintf("probe failed: %v", err), http.StatusInternalServerError)
			return
		}

		v := verdict.ComputeVerdict(result, nil)
		_ = st.SaveProbe(r.Context(), v)

		writeJSON(w, v)
	}
}

func handleBatch(engine *probe.Engine, st store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req batchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if len(req.URLs) == 0 {
			httpError(w, "urls is required", http.StatusBadRequest)
			return
		}

		concurrency := req.Concurrency
		if concurrency < 1 {
			concurrency = 5
		}

		results := make([]*verdict.ProbeVerdict, len(req.URLs))
		sem := make(chan struct{}, concurrency)
		var wg sync.WaitGroup
		var mu sync.Mutex

		for i, u := range req.URLs {
			wg.Add(1)
			go func(idx int, url string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				probeReq := probe.ProbeRequest{
					URL:         url,
					RepeatCount: req.Repeat,
				}

				result, err := engine.Probe(r.Context(), probeReq)
				if err != nil {
					mu.Lock()
					results[idx] = &verdict.ProbeVerdict{
						Endpoint: url,
						Verdict:  verdict.VerdictUnknown,
					}
					mu.Unlock()
					return
				}

				v := verdict.ComputeVerdict(result, nil)
				mu.Lock()
				results[idx] = v
				mu.Unlock()

				_ = st.SaveProbe(r.Context(), v)
			}(i, u)
		}

		wg.Wait()
		writeJSON(w, results)
	}
}

func handlePolicy(engine *probe.Engine, st store.Store, loader *policy.Loader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req policyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.URL == "" || req.PolicyName == "" {
			httpError(w, "url and policy_name are required", http.StatusBadRequest)
			return
		}

		rule, ok := loader.Get(req.PolicyName)
		if !ok {
			httpError(w, fmt.Sprintf("policy %q not found", req.PolicyName), http.StatusNotFound)
			return
		}

		probeReq := probe.ProbeRequest{
			URL:         req.URL,
			RepeatCount: 1,
		}

		result, err := engine.Probe(r.Context(), probeReq)
		if err != nil {
			httpError(w, fmt.Sprintf("probe failed: %v", err), http.StatusInternalServerError)
			return
		}

		v := verdict.ComputeVerdict(result, rule)
		_ = st.SaveProbe(r.Context(), v)

		writeJSON(w, v)
	}
}

func handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"status": "ok"})
	}
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(data)
}

func httpError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
