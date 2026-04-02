package httpserver

import (
	"context"
	"net/http"

	"github.com/Sudhan30/freshprobe/internal/policy"
	"github.com/Sudhan30/freshprobe/internal/probe"
	"github.com/Sudhan30/freshprobe/internal/store"
)

// Run starts the HTTP server and blocks until ctx is cancelled.
func Run(ctx context.Context, engine *probe.Engine, st store.Store, loader *policy.Loader, addr string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/check", handleCheck(engine, st))
	mux.HandleFunc("POST /api/v1/batch", handleBatch(engine, st))
	mux.HandleFunc("POST /api/v1/policy", handlePolicy(engine, st, loader))
	mux.HandleFunc("GET /healthz", handleHealth())
	mux.HandleFunc("GET /metrics", handleMetrics())

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		server.Close()
	}()

	err := server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}
