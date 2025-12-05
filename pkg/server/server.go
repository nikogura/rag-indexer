// Package server provides HTTP API endpoints for the code indexer.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/nikogura/rag-indexer/pkg/config"
	"github.com/nikogura/rag-indexer/pkg/elasticsearch"
	"github.com/nikogura/rag-indexer/pkg/indexer"
	"github.com/nikogura/rag-indexer/pkg/logging"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server handles HTTP API requests.
type Server struct {
	indexer *indexer.Indexer
	es      *elasticsearch.Client
	config  config.Config
	logger  logging.Logger
}

// New creates a new HTTP server instance.
func New(idx *indexer.Indexer, es *elasticsearch.Client, cfg config.Config, logger logging.Logger) (server *Server) {
	server = &Server{
		indexer: idx,
		es:      es,
		config:  cfg,
		logger:  logger,
	}
	return server
}

// Start starts the HTTP server and blocks until context is cancelled.
func (s *Server) Start(ctx context.Context) (err error) {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ready", s.handleReady)
	mux.HandleFunc("/api/v1/search", s.handleSearch)
	mux.HandleFunc("/api/v1/reindex", s.handleReindex)
	mux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:    s.config.HTTPAddr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	s.logger.Info("Starting HTTP server", "address", s.config.HTTPAddr)
	err = srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		err = fmt.Errorf("server error: %w", err)
		return err
	}

	return err
}

// handleHealth is the liveness probe endpoint.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, "OK")
}

// handleReady is the readiness probe endpoint.
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	readyErr := s.es.Ping()
	if readyErr != nil {
		http.Error(w, "Elasticsearch unavailable", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, "READY")
}

// handleSearch handles search requests.
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req elasticsearch.SearchRequest
	decodeErr := json.NewDecoder(r.Body).Decode(&req)
	if decodeErr != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	results, searchErr := s.es.Search(r.Context(), req.Query, req.Limit)
	if searchErr != nil {
		s.logger.Error("Search error", "query", req.Query, "error", searchErr)
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(results)
}

// handleReindex triggers a background reindex operation.
func (s *Server) handleReindex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	go func() {
		count, indexErr := s.indexer.IndexAllRepos(context.Background())
		if indexErr != nil {
			s.logger.Error("Reindex error", "error", indexErr)
		} else {
			s.logger.Info("Reindex complete", "functions", count)
		}
	}()

	w.WriteHeader(http.StatusAccepted)
	_, _ = fmt.Fprintf(w, "Reindex triggered")
}
