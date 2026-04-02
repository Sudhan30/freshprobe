package cli

import (
	"fmt"
	"os"

	"github.com/Sudhan30/freshprobe/internal/version"
	"github.com/spf13/cobra"
)

var (
	cfgStateless bool
	cfgDBPath    string
	cfgPolicyDir string
	cfgOutput    string
)

// NewRootCmd creates the root cobra command.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "freshprobe",
		Short: "Data freshness and endpoint liveness verification",
		Long:  "freshprobe probes endpoints for data freshness and liveness, producing deterministic JSON verdicts for AI agents and automated systems.",
		Version: version.Version,
	}

	root.PersistentFlags().BoolVar(&cfgStateless, "stateless", false, "Run without persistence (no SQLite)")
	root.PersistentFlags().StringVar(&cfgDBPath, "db", "", "SQLite database path (default: ~/.freshprobe/data.db)")
	root.PersistentFlags().StringVar(&cfgPolicyDir, "policy-dir", "", "Directory containing policy YAML files")
	root.PersistentFlags().StringVar(&cfgOutput, "output", "json", "Output format: json or text")

	root.AddCommand(newCheckCmd())
	root.AddCommand(newBatchCmd())
	root.AddCommand(newServeCmd())
	root.AddCommand(newWatchCmd())
	root.AddCommand(newHistoryCmd())

	return root
}

// Execute runs the root command.
func Execute() error {
	return NewRootCmd().Execute()
}

func buildDeps() (*Dependencies, error) {
	return NewDependencies(Config{
		Stateless: cfgStateless,
		DBPath:    cfgDBPath,
		PolicyDir: cfgPolicyDir,
	})
}

func exitErr(msg string, err error) {
	fmt.Fprintf(os.Stderr, "error: %s: %v\n", msg, err)
	os.Exit(1)
}
