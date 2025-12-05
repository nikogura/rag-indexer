package elasticsearch

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
)

// indexMapping defines the Elasticsearch index mapping per CLAUDE.md specification.
const indexMapping = `{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0,
    "refresh_interval": "30s"
  },
  "mappings": {
    "properties": {
      "repo": {"type": "keyword"},
      "file_path": {"type": "keyword"},
      "function_name": {"type": "keyword"},
      "code": {"type": "text", "analyzer": "standard"},
      "has_namedreturns": {"type": "boolean"},
      "has_error_handling": {"type": "boolean"},
      "package": {"type": "keyword"},
      "imports": {"type": "keyword"},
      "lint_compliant": {"type": "boolean"},
      "indexed_at": {"type": "date"}
    }
  }
}`

// EnsureIndex ensures the index exists with the correct mapping.
// If the index already exists, this is a no-op.
func (es *Client) EnsureIndex(ctx context.Context) (err error) {
	// Check if index exists
	exists, checkErr := es.indexExists(ctx)
	if checkErr != nil {
		err = fmt.Errorf("failed to check if index exists: %w", checkErr)
		return err
	}

	if exists {
		return err
	}

	// Create index with mapping
	url := fmt.Sprintf("%s/%s", es.host, es.index)

	var req *http.Request
	req, err = http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBufferString(indexMapping))
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
		err = fmt.Errorf("failed to create index: %w", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("elasticsearch error creating index: %s - %s", resp.Status, string(body))
		return err
	}

	return err
}

// indexExists checks if the index exists.
func (es *Client) indexExists(ctx context.Context) (exists bool, err error) {
	url := fmt.Sprintf("%s/%s", es.host, es.index)

	var req *http.Request
	req, err = http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return exists, err
	}

	if es.username != "" {
		req.SetBasicAuth(es.username, es.password)
	}

	var resp *http.Response
	resp, err = es.client.Do(req)
	if err != nil {
		return exists, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		exists = true
		return exists, err
	}

	if resp.StatusCode == http.StatusNotFound {
		exists = false
		return exists, err
	}

	err = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	return exists, err
}
