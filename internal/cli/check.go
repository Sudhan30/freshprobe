package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	policyPkg "github.com/Sudhan30/freshprobe/internal/policy"
	"github.com/Sudhan30/freshprobe/internal/probe"
	"github.com/Sudhan30/freshprobe/internal/verdict"
	"github.com/spf13/cobra"
)

func newCheckCmd() *cobra.Command {
	var (
		repeat   int
		interval time.Duration
		timeout  time.Duration
		skipTLS  bool
		skipDNS  bool
		policy   string
	)

	cmd := &cobra.Command{
		Use:   "check <url>",
		Short: "Probe a single endpoint for data freshness and liveness",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := buildDeps()
			if err != nil {
				return fmt.Errorf("init: %w", err)
			}
			defer deps.Store.Close()

			req := probe.ProbeRequest{
				URL:            args[0],
				Timeout:        timeout,
				RepeatCount:    repeat,
				RepeatInterval: interval,
				SkipTLS:        skipTLS,
				SkipDNS:        skipDNS,
			}

			ctx := context.Background()
			result, err := deps.Engine.Probe(ctx, req)
			if err != nil {
				return fmt.Errorf("probe: %w", err)
			}

			// Get policy rule if specified
			var policyRule *policyPkg.PolicyRule
			if policy != "" {
				rule, ok := deps.PolicyLoader.Get(policy)
				if !ok {
					return fmt.Errorf("policy %q not found (available: %v)", policy, deps.PolicyLoader.List())
				}
				policyRule = rule
			}

			v := verdict.ComputeVerdict(result, policyRule)

			// Save to store
			if err := deps.Store.SaveProbe(ctx, v); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to save probe: %v\n", err)
			}

			return outputVerdict(v)
		},
	}

	cmd.Flags().IntVar(&repeat, "repeat", 1, "Number of repeat probes for fingerprinting")
	cmd.Flags().DurationVar(&interval, "interval", 2*time.Second, "Interval between repeat probes")
	cmd.Flags().DurationVar(&timeout, "timeout", 10*time.Second, "Probe timeout")
	cmd.Flags().BoolVar(&skipTLS, "skip-tls", false, "Skip TLS certificate checks")
	cmd.Flags().BoolVar(&skipDNS, "skip-dns", false, "Skip DNS resolution timing")
	cmd.Flags().StringVar(&policy, "policy", "", "Policy name to evaluate against")

	return cmd
}

func outputVerdict(v *verdict.ProbeVerdict) error {
	if cfgOutput == "text" {
		fmt.Printf("Endpoint:  %s\n", v.Endpoint)
		fmt.Printf("Verdict:   %s\n", v.Verdict)
		fmt.Printf("Confidence: %.2f\n", v.Confidence)
		if v.Freshness != nil {
			fmt.Printf("Data Age:  %ds\n", v.Freshness.DataAgeSeconds)
			fmt.Printf("Freshness: %.2f\n", v.Freshness.FreshnessScore)
		}
		if v.Liveness != nil {
			fmt.Printf("Liveness:  %s [%d] (P50: %dms, P95: %dms)\n", v.Liveness.Status, v.Liveness.StatusCode, v.Liveness.P50Ms, v.Liveness.P95Ms)
		}
		if v.PolicyResult != nil {
			if v.PolicyResult.Passed {
				fmt.Printf("Policy:    PASS (%s)\n", v.PolicyResult.PolicyName)
			} else {
				fmt.Printf("Policy:    FAIL (%s)\n", v.PolicyResult.PolicyName)
				for _, viol := range v.PolicyResult.Violations {
					fmt.Printf("  - %s: expected %s, got %s\n", viol.Check, viol.Expected, viol.Actual)
				}
			}
		}
		return nil
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
