// Package elasticsearch provides Elasticsearch client and data models for code indexing.
package elasticsearch

import "time"

// CodeDocument represents a Go function indexed in Elasticsearch.
type CodeDocument struct {
	Repo             string    `json:"repo"`
	FilePath         string    `json:"file_path"`
	FunctionName     string    `json:"function_name"`
	Code             string    `json:"code"`
	HasNamedReturns  bool      `json:"has_namedreturns"`
	HasErrorHandling bool      `json:"has_error_handling"`
	Package          string    `json:"package"`
	Imports          []string  `json:"imports"`
	LintCompliant    bool      `json:"lint_compliant"`
	IndexedAt        time.Time `json:"indexed_at"`
}

// SearchRequest represents a search query request.
type SearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

// SearchResponse represents the Elasticsearch search response.
type SearchResponse struct {
	Hits struct {
		Hits []struct {
			Source CodeDocument `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}
