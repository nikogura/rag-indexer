package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/nikogura/rag-indexer/pkg/metrics"
)

const (
	maxRetries      = 3
	retryBackoff    = 500 * time.Millisecond
	retryMultiplier = 2
)

// Client handles Elasticsearch operations.
type Client struct {
	host     string
	index    string
	username string
	password string
	client   *http.Client
	metrics  *metrics.Metrics
}

// NewClient creates a new Elasticsearch client and verifies connectivity.
func NewClient(host string, index string, username string, password string, m *metrics.Metrics) (client *Client, err error) {
	client = &Client{
		host:     host,
		index:    index,
		username: username,
		password: password,
		metrics:  m,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	err = client.Ping()
	if err != nil {
		client = nil
		err = fmt.Errorf("failed to connect to Elasticsearch: %w", err)
		return client, err
	}

	return client, err
}

// doRequestWithRetry executes an HTTP request with exponential backoff retry for 5xx errors.
func (es *Client) doRequestWithRetry(req *http.Request) (resp *http.Response, err error) {
	backoff := retryBackoff

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-req.Context().Done():
				err = req.Context().Err()
				return resp, err
			case <-time.After(backoff):
				backoff *= retryMultiplier
			}
		}

		resp, err = es.client.Do(req)
		if err != nil {
			// Network error - retry
			continue
		}

		// Success or client error (4xx) - don't retry
		if resp.StatusCode < http.StatusInternalServerError {
			return resp, err
		}

		// Server error (5xx) - close body and retry
		_ = resp.Body.Close()
	}

	// All retries exhausted
	if err == nil && resp != nil {
		err = fmt.Errorf("elasticsearch request failed after %d retries: status %d", maxRetries, resp.StatusCode)
	}

	return resp, err
}

// Ping verifies that Elasticsearch is reachable.
func (es *Client) Ping() (err error) {
	var req *http.Request
	req, err = http.NewRequestWithContext(context.Background(), http.MethodGet, es.host, nil)
	if err != nil {
		return err
	}

	if es.username != "" {
		req.SetBasicAuth(es.username, es.password)
	}

	var resp *http.Response
	resp, err = es.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		err = fmt.Errorf("elasticsearch returned status %d", resp.StatusCode)
		return err
	}

	return err
}

// IndexDocument indexes a single code document into Elasticsearch.
func (es *Client) IndexDocument(ctx context.Context, doc CodeDocument) (err error) {
	var data []byte
	data, err = json.Marshal(doc)
	if err != nil {
		err = fmt.Errorf("failed to marshal document: %w", err)
		return err
	}

	url := fmt.Sprintf("%s/%s/_doc", es.host, es.index)

	var req *http.Request
	req, err = http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		err = fmt.Errorf("failed to create request: %w", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if es.username != "" {
		req.SetBasicAuth(es.username, es.password)
	}

	var resp *http.Response
	resp, err = es.doRequestWithRetry(req)
	if err != nil {
		es.metrics.ESRequests.WithLabelValues("index", "error").Inc()
		err = fmt.Errorf("failed to send request: %w", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(resp.Body)
		es.metrics.ESRequests.WithLabelValues("index", "error").Inc()
		err = fmt.Errorf("elasticsearch error: %s - %s", resp.Status, string(body))
		return err
	}

	es.metrics.ESRequests.WithLabelValues("index", "success").Inc()
	return err
}

// Search performs a search query against Elasticsearch.
func (es *Client) Search(ctx context.Context, query string, limit int) (results []CodeDocument, err error) {
	if limit <= 0 {
		limit = 10
	}

	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  query,
				"fields": []string{"function_name^3", "code^2", "package"},
			},
		},
		"size": limit,
		"sort": []map[string]interface{}{
			{"has_namedreturns": "desc"},
			{"has_error_handling": "desc"},
		},
	}

	var data []byte
	data, err = json.Marshal(searchQuery)
	if err != nil {
		err = fmt.Errorf("failed to marshal query: %w", err)
		return results, err
	}

	url := fmt.Sprintf("%s/%s/_search", es.host, es.index)

	var req *http.Request
	req, err = http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		err = fmt.Errorf("failed to create request: %w", err)
		return results, err
	}

	req.Header.Set("Content-Type", "application/json")
	if es.username != "" {
		req.SetBasicAuth(es.username, es.password)
	}

	var resp *http.Response
	resp, err = es.doRequestWithRetry(req)
	if err != nil {
		es.metrics.ESRequests.WithLabelValues("search", "error").Inc()
		err = fmt.Errorf("failed to execute search: %w", err)
		return results, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(resp.Body)
		es.metrics.ESRequests.WithLabelValues("search", "error").Inc()
		err = fmt.Errorf("elasticsearch error: %s - %s", resp.Status, string(body))
		return results, err
	}

	var searchResp SearchResponse
	err = json.NewDecoder(resp.Body).Decode(&searchResp)
	if err != nil {
		err = fmt.Errorf("failed to decode response: %w", err)
		return results, err
	}

	es.metrics.ESRequests.WithLabelValues("search", "success").Inc()

	for _, hit := range searchResp.Hits.Hits {
		results = append(results, hit.Source)
	}

	return results, err
}
