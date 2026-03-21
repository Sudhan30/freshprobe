package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Sudhan30/freshprobe/internal/probe"
	"github.com/Sudhan30/freshprobe/internal/verdict"
	"github.com/spf13/cobra"
)

func newBatchCmd() *cobra.Command {
	var (
		repeat      int
		timeout     time.Duration
		concurrency int
	)

	cmd := &cobra.Command{
		Use:   "batch <urls or file>",
		Short: "Probe multiple endpoints concurrently",
		Long:  "Provide URLs as arguments or pass a file path containing one URL per line.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := buildDeps()
			if err != nil {
				return fmt.Errorf("init: %w", err)
			}
			defer deps.Store.Close()

			urls, err := resolveURLs(args)
			if err != nil {
				return err
			}

			if concurrency < 1 {
				concurrency = 5
			}

			ctx := context.Background()
			results := make([]*verdict.ProbeVerdict, len(urls))
			var mu sync.Mutex
			sem := make(chan struct{}, concurrency)
			var wg sync.WaitGroup

			for i, u := range urls {
				wg.Add(1)
				go func(idx int, url string) {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()

					req := probe.ProbeRequest{
						URL:         url,
						Timeout:     timeout,
						RepeatCount: repeat,
					}

					result, err := deps.Engine.Probe(ctx, req)
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

					_ = deps.Store.SaveProbe(ctx, v)
				}(i, u)
			}

			wg.Wait()

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(results)
		},
	}

	cmd.Flags().IntVar(&repeat, "repeat", 1, "Number of repeat probes")
	cmd.Flags().DurationVar(&timeout, "timeout", 10*time.Second, "Probe timeout")
	cmd.Flags().IntVar(&concurrency, "concurrency", 5, "Max concurrent probes")

	return cmd
}

// resolveURLs handles both direct URL args and file paths.
func resolveURLs(args []string) ([]string, error) {
	// If single arg and it looks like a file path (no scheme), try reading as file
	if len(args) == 1 && !strings.Contains(args[0], "://") {
		if _, err := os.Stat(args[0]); err == nil {
			return readURLFile(args[0])
		}
	}
	return args, nil
}

func readURLFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open url file: %w", err)
	}
	defer f.Close()

	var urls []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			urls = append(urls, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(urls) == 0 {
		return nil, fmt.Errorf("no URLs found in %s", path)
	}
	return urls, nil
}
