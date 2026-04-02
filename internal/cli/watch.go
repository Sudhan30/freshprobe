package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Sudhan30/freshprobe/internal/probe"
	"github.com/Sudhan30/freshprobe/internal/verdict"
	"github.com/spf13/cobra"
)

func newWatchCmd() *cobra.Command {
	var (
		interval time.Duration
		count    int
		timeout  time.Duration
		onChange bool
		skipTLS  bool
		skipDNS  bool
	)

	cmd := &cobra.Command{
		Use:   "watch <url>",
		Short: "Continuously monitor an endpoint for freshness changes",
		Long:  "Watch probes an endpoint at a regular interval, printing verdicts. Use --on-change to only print when the verdict changes (e.g. FRESH to STALE).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := buildDeps()
			if err != nil {
				return fmt.Errorf("init: %w", err)
			}
			defer deps.Store.Close()

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			req := probe.ProbeRequest{
				URL:     args[0],
				Timeout: timeout,
				SkipTLS: skipTLS,
				SkipDNS: skipDNS,
			}

			var lastVerdict verdict.Verdict
			iteration := 0

			fmt.Fprintf(os.Stderr, "Watching %s every %s", args[0], interval)
			if onChange {
				fmt.Fprintf(os.Stderr, " (changes only)")
			}
			if count > 0 {
				fmt.Fprintf(os.Stderr, " [%d iterations]", count)
			}
			fmt.Fprintln(os.Stderr)

			for {
				select {
				case <-ctx.Done():
					fmt.Fprintln(os.Stderr, "\nWatch stopped.")
					return nil
				default:
				}

				result, err := deps.Engine.Probe(ctx, req)
				if err != nil {
					fmt.Fprintf(os.Stderr, "probe error: %v\n", err)
				} else {
					v := verdict.ComputeVerdict(result, nil)
					_ = deps.Store.SaveProbe(ctx, v)

					if !onChange || v.Verdict != lastVerdict {
						if cfgOutput == "text" {
							ts := v.ProbeMetadata.Timestamp.Format("15:04:05")
							score := ""
							if v.Freshness != nil {
								score = fmt.Sprintf(" score=%.2f", v.Freshness.FreshnessScore)
							}
							latency := ""
							if v.Liveness != nil {
								latency = fmt.Sprintf(" p95=%dms", v.Liveness.P95Ms)
							}

							change := ""
							if lastVerdict != "" && v.Verdict != lastVerdict {
								change = fmt.Sprintf(" [%s -> %s]", lastVerdict, v.Verdict)
							}

							fmt.Printf("[%s] %s conf=%.2f%s%s%s\n",
								ts, v.Verdict, v.Confidence, score, latency, change)
						} else {
							enc := json.NewEncoder(os.Stdout)
							enc.Encode(v)
						}
					}
					lastVerdict = v.Verdict
				}

				iteration++
				if count > 0 && iteration >= count {
					return nil
				}

				select {
				case <-time.After(interval):
				case <-ctx.Done():
					fmt.Fprintln(os.Stderr, "\nWatch stopped.")
					return nil
				}
			}
		},
	}

	cmd.Flags().DurationVar(&interval, "interval", 30*time.Second, "Time between probes")
	cmd.Flags().IntVar(&count, "count", 0, "Number of probes (0 = infinite)")
	cmd.Flags().DurationVar(&timeout, "timeout", 10*time.Second, "Probe timeout")
	cmd.Flags().BoolVar(&onChange, "on-change", false, "Only print when verdict changes")
	cmd.Flags().BoolVar(&skipTLS, "skip-tls", false, "Skip TLS certificate checks")
	cmd.Flags().BoolVar(&skipDNS, "skip-dns", false, "Skip DNS resolution timing")

	return cmd
}
