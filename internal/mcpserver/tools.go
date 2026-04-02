package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Sudhan30/freshprobe/internal/policy"
	"github.com/Sudhan30/freshprobe/internal/probe"
	"github.com/Sudhan30/freshprobe/internal/store"
	"github.com/Sudhan30/freshprobe/internal/verdict"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CheckInput is the input schema for freshprobe_check.
type CheckInput struct {
	URL         string `json:"url" jsonschema:"description=The URL to probe for freshness,required"`
	Repeat      int    `json:"repeat,omitempty" jsonschema:"description=Number of repeat probes for fingerprinting"`
	TimeoutSecs int    `json:"timeout_secs,omitempty" jsonschema:"description=Probe timeout in seconds"`
	SkipTLS     bool   `json:"skip_tls,omitempty" jsonschema:"description=Skip TLS certificate checks"`
	SkipDNS     bool   `json:"skip_dns,omitempty" jsonschema:"description=Skip DNS resolution timing"`
}

// BatchInput is the input schema for freshprobe_batch.
type BatchInput struct {
	URLs        []string `json:"urls" jsonschema:"description=List of URLs to probe,required"`
	Repeat      int      `json:"repeat,omitempty" jsonschema:"description=Number of repeat probes per URL"`
	Concurrency int      `json:"concurrency,omitempty" jsonschema:"description=Max concurrent probes"`
}

// PolicyInput is the input schema for freshprobe_policy.
type PolicyInput struct {
	URL        string `json:"url" jsonschema:"description=The URL to check,required"`
	PolicyName string `json:"policy_name" jsonschema:"description=Name of the freshness policy to evaluate against,required"`
}

func registerTools(server *mcp.Server, engine *probe.Engine, st store.Store, loader *policy.Loader) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "freshprobe_check",
		Description: "Probe a single endpoint for data freshness and liveness. Returns a deterministic JSON verdict with freshness score, latency percentiles, TLS status, and content fingerprint.",
	}, makeCheckHandler(engine, st))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "freshprobe_batch",
		Description: "Probe multiple endpoints concurrently for data freshness. Returns an array of JSON verdicts.",
	}, makeBatchHandler(engine, st))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "freshprobe_policy",
		Description: "Check an endpoint against a named freshness policy. Returns verdict with pass/fail evaluation against policy thresholds.",
	}, makePolicyHandler(engine, st, loader))
}

func makeCheckHandler(engine *probe.Engine, st store.Store) func(context.Context, *mcp.CallToolRequest, CheckInput) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input CheckInput) (*mcp.CallToolResult, any, error) {
		timeout := 10 * time.Second
		if input.TimeoutSecs > 0 {
			timeout = time.Duration(input.TimeoutSecs) * time.Second
		}
		repeat := input.Repeat
		if repeat < 1 {
			repeat = 1
		}

		probeReq := probe.ProbeRequest{
			URL:         input.URL,
			Timeout:     timeout,
			RepeatCount: repeat,
			SkipTLS:     input.SkipTLS,
			SkipDNS:     input.SkipDNS,
		}

		result, err := engine.Probe(ctx, probeReq)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("probe error: %v", err)}},
				IsError: true,
			}, nil, nil
		}

		v := verdict.ComputeVerdict(result, nil)
		_ = st.SaveProbe(ctx, v)

		return verdictToResult(v)
	}
}

func makeBatchHandler(engine *probe.Engine, st store.Store) func(context.Context, *mcp.CallToolRequest, BatchInput) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input BatchInput) (*mcp.CallToolResult, any, error) {
		concurrency := input.Concurrency
		if concurrency < 1 {
			concurrency = 5
		}

		results := make([]*verdict.ProbeVerdict, len(input.URLs))
		sem := make(chan struct{}, concurrency)
		var wg sync.WaitGroup
		var mu sync.Mutex

		for i, u := range input.URLs {
			wg.Add(1)
			go func(idx int, url string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				probeReq := probe.ProbeRequest{
					URL:         url,
					RepeatCount: input.Repeat,
				}

				result, err := engine.Probe(ctx, probeReq)
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

				_ = st.SaveProbe(ctx, v)
			}(i, u)
		}

		wg.Wait()

		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("marshal error: %v", err)}},
				IsError: true,
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil
	}
}

func makePolicyHandler(engine *probe.Engine, st store.Store, loader *policy.Loader) func(context.Context, *mcp.CallToolRequest, PolicyInput) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input PolicyInput) (*mcp.CallToolResult, any, error) {
		rule, ok := loader.Get(input.PolicyName)
		if !ok {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("policy %q not found (available: %v)", input.PolicyName, loader.List())}},
				IsError: true,
			}, nil, nil
		}

		probeReq := probe.ProbeRequest{
			URL:         input.URL,
			RepeatCount: 1,
		}

		result, err := engine.Probe(ctx, probeReq)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("probe error: %v", err)}},
				IsError: true,
			}, nil, nil
		}

		v := verdict.ComputeVerdict(result, rule)
		_ = st.SaveProbe(ctx, v)

		return verdictToResult(v)
	}
}

func verdictToResult(v *verdict.ProbeVerdict) (*mcp.CallToolResult, any, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("marshal error: %v", err)}},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}
