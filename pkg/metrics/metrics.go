// Package metrics provides Prometheus metrics for code indexing operations.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds Prometheus metrics for the code indexer.
type Metrics struct {
	FunctionsIndexed    *prometheus.CounterVec
	ReposIndexed        prometheus.Counter
	IndexingDuration    *prometheus.HistogramVec
	ParseErrors         *prometheus.CounterVec
	ESRequests          *prometheus.CounterVec
	LastSuccessfulIndex *prometheus.GaugeVec
}

// New creates and registers new Prometheus metrics.
func New() (metrics *Metrics) {
	metrics = &Metrics{
		FunctionsIndexed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "code_indexer_functions_indexed_total",
				Help: "Total number of functions indexed",
			},
			[]string{"repo"},
		),
		ReposIndexed: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "code_indexer_repos_indexed_total",
				Help: "Total number of repositories indexed",
			},
		),
		IndexingDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "code_indexer_indexing_duration_seconds",
				Help:    "Time taken to index a repository",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"repo"},
		),
		ParseErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "code_indexer_parse_errors_total",
				Help: "Total number of parse errors",
			},
			[]string{"repo", "file"},
		),
		ESRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "code_indexer_elasticsearch_requests_total",
				Help: "Total number of Elasticsearch requests",
			},
			[]string{"operation", "status"},
		),
		LastSuccessfulIndex: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "code_indexer_last_successful_index_timestamp",
				Help: "Timestamp of last successful index",
			},
			[]string{"repo"},
		),
	}
	return metrics
}
