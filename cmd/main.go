package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/dnywonnt/mcp-jira-tempo/internal/config"
	"github.com/dnywonnt/mcp-jira-tempo/internal/jira"
	"github.com/dnywonnt/mcp-jira-tempo/internal/mcp"
)

func main() {
	cfg, err := config.LoadDefault()
	if err != nil {
		slog.Error("Failed to load config", "err", err)
		os.Exit(1)
	}

	httpClient := &http.Client{
		Timeout: cfg.HTTPClientTimeout(),
	}
	client := jira.NewClient(cfg, httpClient)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := mcp.Serve(ctx, client); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("Failed to serve MCP", "err", err)
		os.Exit(1)
	}
}
