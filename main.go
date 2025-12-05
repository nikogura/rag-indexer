package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/nikogura/rag-indexer/pkg/config"
	"github.com/nikogura/rag-indexer/pkg/elasticsearch"
	"github.com/nikogura/rag-indexer/pkg/indexer"
	"github.com/nikogura/rag-indexer/pkg/logging"
	"github.com/nikogura/rag-indexer/pkg/metrics"
	"github.com/nikogura/rag-indexer/pkg/server"
)

//nolint:gochecknoglobals // Command-line flag
var mode string

//nolint:gochecknoinits // Flag initialization
func init() {
	flag.StringVar(&mode, "mode", "serve", "Run mode: serve, index, or search")
}

func main() {
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create structured logger
	slogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	logger := logging.New(slogger)

	m := metrics.New()

	es, err := elasticsearch.NewClient(cfg.ESHost, cfg.ESIndex, cfg.ESUsername, cfg.ESPassword, m)
	if err != nil {
		log.Fatalf("Failed to connect to Elasticsearch: %v", err)
	}

	// Ensure ES index exists with proper mapping
	err = es.EnsureIndex(context.Background())
	if err != nil {
		log.Fatalf("Failed to ensure Elasticsearch index: %v", err)
	}

	idx := indexer.New(cfg, es, m, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutdown signal received")
		cancel()
	}()

	switch mode {
	case "serve":
		runServeMode(ctx, cfg, idx, es, logger)

	case "index":
		runIndexMode(ctx, idx)

	case "search":
		runSearchMode(ctx, es)

	default:
		log.Fatalf("Unknown mode: %s (use serve, index, or search)", mode)
	}
}

func runServeMode(ctx context.Context, cfg config.Config, idx *indexer.Indexer, es *elasticsearch.Client, logger logging.Logger) {
	if cfg.GitOrg != "" && len(cfg.GitRepos) > 0 {
		log.Println("Cloning/updating repositories...")
		err := idx.CloneRepos(ctx)
		if err != nil {
			log.Printf("Warning: failed to clone repos: %v", err)
		}
	}

	log.Println("Running initial index...")
	count, err := idx.IndexAllRepos(ctx)
	if err != nil {
		log.Printf("Warning: initial index failed: %v", err)
	} else {
		log.Printf("Initial index complete: %d functions", count)
	}

	go idx.RunIndexingLoop(ctx)

	srv := server.New(idx, es, cfg, logger)
	err = srv.Start(ctx)
	if err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func runIndexMode(ctx context.Context, idx *indexer.Indexer) {
	log.Println("Running one-shot index...")
	count, err := idx.IndexAllRepos(ctx)
	if err != nil {
		log.Fatalf("Index failed: %v", err)
	}
	log.Printf("Index complete: %d functions indexed", count)
}

func runSearchMode(ctx context.Context, es *elasticsearch.Client) {
	query := strings.Join(flag.Args(), " ")
	if query == "" {
		log.Fatal("Search query required")
	}

	results, err := es.Search(ctx, query, 10)
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		fmt.Println("No results found")
		return
	}

	for i, result := range results {
		fmt.Printf("\n=== Result %d: %s/%s - %s ===\n",
			i+1, result.Repo, result.FilePath, result.FunctionName)
		fmt.Printf("Named Returns: %v\n", result.HasNamedReturns)
		fmt.Printf("\n%s\n", result.Code)
	}
}
