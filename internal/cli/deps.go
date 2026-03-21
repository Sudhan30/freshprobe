package cli

import (
	"github.com/Sudhan30/freshprobe/internal/policy"
	"github.com/Sudhan30/freshprobe/internal/probe"
	"github.com/Sudhan30/freshprobe/internal/store"
)

// Dependencies holds shared resources for all CLI commands.
type Dependencies struct {
	Engine       *probe.Engine
	Store        store.Store
	PolicyLoader *policy.Loader
}

// Config holds CLI configuration.
type Config struct {
	Stateless bool
	DBPath    string
	PolicyDir string
}

// NewDependencies creates the shared dependency container.
func NewDependencies(cfg Config) (*Dependencies, error) {
	engine := probe.NewEngine()

	var st store.Store
	if cfg.Stateless {
		st = store.NewNoopStore()
	} else {
		var err error
		st, err = store.NewSQLiteStore(cfg.DBPath)
		if err != nil {
			return nil, err
		}
	}

	var loader *policy.Loader
	if cfg.PolicyDir != "" {
		var err error
		loader, err = policy.NewLoader(cfg.PolicyDir)
		if err != nil {
			return nil, err
		}
	} else {
		loader = policy.NewEmptyLoader()
	}

	return &Dependencies{
		Engine:       engine,
		Store:        st,
		PolicyLoader: loader,
	}, nil
}
