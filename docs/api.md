# API Documentation

The code indexer exposes an HTTP API for searching indexed code and triggering operations.

## Base URL

Default: `http://localhost:8080`

Configure with `HTTP_ADDR` environment variable.

## Authentication

Currently no authentication. Intended for internal/private network use.

**Security note:** Do not expose to public internet without adding authentication layer (reverse proxy, API gateway, etc.).

## Endpoints

### Health Check

```
GET /health
```

Liveness probe - checks if the service is running.

**Response:**

```
200 OK
OK
```

**Use case:** Kubernetes liveness probe, monitoring systems

**Example:**

```bash
curl http://localhost:8080/health
```

---

### Readiness Check

```
GET /ready
```

Readiness probe - checks if service can handle requests (Elasticsearch connectivity).

**Response:**

Success:
```
200 OK
READY
```

Failure:
```
503 Service Unavailable
Elasticsearch unavailable
```

**Use case:** Kubernetes readiness probe, load balancer health checks

**Example:**

```bash
curl http://localhost:8080/ready
```

---

### Search Code

```
POST /api/v1/search
```

Search indexed code with natural language or specific terms.

**Request:**

```json
{
  "query": "error handling http",
  "limit": 10
}
```

**Parameters:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| query | string | Yes | Search query (natural language or keywords) |
| limit | integer | No | Max results (default: 10, max: 100) |

**Response:**

```json
[
  {
    "repo": "api-service",
    "file_path": "pkg/handlers/auth.go",
    "function_name": "HandleLogin",
    "code": "func HandleLogin(ctx context.Context, req LoginRequest) (resp LoginResponse, err error) {\n\t// Implementation...\n\treturn resp, err\n}",
    "has_namedreturns": true,
    "has_error_handling": true,
    "package": "handlers",
    "imports": ["context", "net/http", "errors"],
    "lint_compliant": false,
    "indexed_at": "2025-10-30T10:30:00Z"
  }
]
```

**Response Fields:**

| Field | Type | Description |
|-------|------|-------------|
| repo | string | Repository name |
| file_path | string | File path relative to repo root |
| function_name | string | Function name |
| code | string | Complete function source code |
| has_namedreturns | boolean | Uses named return values |
| has_error_handling | boolean | Contains error handling (heuristic) |
| package | string | Go package name |
| imports | array | List of imported packages |
| lint_compliant | boolean | Passes golangci-lint (placeholder, always false) |
| indexed_at | string | ISO 8601 timestamp of indexing |

**Status Codes:**

- `200 OK` - Success (even if 0 results)
- `400 Bad Request` - Invalid request (missing query, invalid limit)
- `500 Internal Server Error` - Search failed (ES error)

**Examples:**

```bash
# Natural language query
curl -X POST http://localhost:8080/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{"query": "http handler with error handling", "limit": 5}'

# Function name search
curl -X POST http://localhost:8080/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{"query": "ParseConfig", "limit": 10}'

# Package-specific search
curl -X POST http://localhost:8080/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{"query": "context timeout handlers"}'
```

**Search Tips:**

- Use natural language: "http client with retry logic"
- Search by function name: "NewClient"
- Search by package: "handlers authentication"
- Combine terms: "database transaction error handling"
- Results prioritize:
  1. Functions with named returns
  2. Functions with error handling
  3. Relevance score from Elasticsearch

---

### Trigger Reindex

```
POST /api/v1/reindex
```

Triggers a full reindex of all repositories in the background.

**Request:** Empty body

**Response:**

```
202 Accepted
Reindex triggered
```

**Status Codes:**

- `202 Accepted` - Reindex started (runs in background)
- `405 Method Not Allowed` - Wrong HTTP method

**Behavior:**

- Returns immediately, reindex runs asynchronously
- Logs progress and results
- Check logs for completion status
- Safe to call multiple times (mutex prevents concurrent indexing)

**Example:**

```bash
curl -X POST http://localhost:8080/api/v1/reindex
```

**Use cases:**

- Manual reindex after repo updates
- CI/CD pipeline integration
- Webhook triggers on git push
- Recovery after ES issues

---

### Prometheus Metrics

```
GET /metrics
```

Exposes Prometheus metrics in standard format.

**Response:**

```
# HELP code_indexer_functions_indexed_total Total number of functions indexed
# TYPE code_indexer_functions_indexed_total counter
code_indexer_functions_indexed_total{repo="api-service"} 1234

# HELP code_indexer_repos_indexed_total Total number of repositories indexed
# TYPE code_indexer_repos_indexed_total counter
code_indexer_repos_indexed_total 5

# HELP code_indexer_indexing_duration_seconds Time taken to index repository
# TYPE code_indexer_indexing_duration_seconds histogram
code_indexer_indexing_duration_seconds_bucket{repo="api-service",le="10"} 0
code_indexer_indexing_duration_seconds_bucket{repo="api-service",le="30"} 1
code_indexer_indexing_duration_seconds_sum{repo="api-service"} 23.5
code_indexer_indexing_duration_seconds_count{repo="api-service"} 1

# HELP code_indexer_elasticsearch_requests_total Total Elasticsearch requests
# TYPE code_indexer_elasticsearch_requests_total counter
code_indexer_elasticsearch_requests_total{operation="index",status="success"} 1234
code_indexer_elasticsearch_requests_total{operation="search",status="success"} 56

# HELP code_indexer_last_successful_index_timestamp Last successful index time
# TYPE code_indexer_last_successful_index_timestamp gauge
code_indexer_last_successful_index_timestamp{repo="api-service"} 1698662400
```

**Metrics:**

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `code_indexer_functions_indexed_total` | Counter | repo | Functions indexed per repo |
| `code_indexer_repos_indexed_total` | Counter | - | Total repos indexed |
| `code_indexer_indexing_duration_seconds` | Histogram | repo | Time to index repo |
| `code_indexer_parse_errors_total` | Counter | repo, file | Parse failures |
| `code_indexer_elasticsearch_requests_total` | Counter | operation, status | ES request stats |
| `code_indexer_last_successful_index_timestamp` | Gauge | repo | Last successful index (Unix timestamp) |

**Status Codes:**

- `200 OK` - Always (metrics format doesn't use error codes)

**Example:**

```bash
curl http://localhost:8080/metrics
```

**Prometheus scrape config:**

```yaml
scrape_configs:
- job_name: 'code-indexer'
  static_configs:
  - targets: ['code-indexer:8080']
  metrics_path: /metrics
  scrape_interval: 30s
```

---

## Integration Examples

### Claude Code MCP Integration

Query the indexer from Claude Code sessions:

**1. Define MCP server in Claude Code config:**

```json
{
  "mcpServers": {
    "code-indexer": {
      "command": "npx",
      "args": ["-y", "@anthropic-ai/mcp-server-fetch"],
      "env": {
        "CODE_INDEXER_URL": "http://localhost:8080"
      }
    }
  }
}
```

**2. Use in Claude Code:**

```
User: Find examples of HTTP handlers with proper error handling

Claude: [Queries code-indexer API and shows relevant examples]
```

### curl Scripts

**Search and format results:**

```bash
#!/bin/bash
query="$1"
limit="${2:-10}"

curl -s -X POST http://localhost:8080/api/v1/search \
  -H "Content-Type: application/json" \
  -d "{\"query\": \"$query\", \"limit\": $limit}" \
  | jq -r '.[] | "\n=== \(.repo)/\(.file_path) - \(.function_name) ===\n\(.code)\n"'
```

Usage:
```bash
./search.sh "error handling" 5
```

### Python Client

```python
import requests

class CodeIndexer:
    def __init__(self, base_url="http://localhost:8080"):
        self.base_url = base_url

    def search(self, query, limit=10):
        """Search for code examples"""
        resp = requests.post(
            f"{self.base_url}/api/v1/search",
            json={"query": query, "limit": limit}
        )
        resp.raise_for_status()
        return resp.json()

    def reindex(self):
        """Trigger background reindex"""
        resp = requests.post(f"{self.base_url}/api/v1/reindex")
        resp.raise_for_status()
        return resp.text

    def health(self):
        """Check if service is running"""
        resp = requests.get(f"{self.base_url}/health")
        return resp.status_code == 200

    def ready(self):
        """Check if service is ready"""
        resp = requests.get(f"{self.base_url}/ready")
        return resp.status_code == 200

# Usage
indexer = CodeIndexer()

# Search
results = indexer.search("context timeout", limit=5)
for result in results:
    print(f"{result['repo']}/{result['file_path']} - {result['function_name']}")
    print(result['code'])

# Trigger reindex
indexer.reindex()

# Health check
if indexer.ready():
    print("Service is ready")
```

### Go Client

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type SearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

type CodeDocument struct {
	Repo             string   `json:"repo"`
	FilePath         string   `json:"file_path"`
	FunctionName     string   `json:"function_name"`
	Code             string   `json:"code"`
	HasNamedReturns  bool     `json:"has_namedreturns"`
	HasErrorHandling bool     `json:"has_error_handling"`
	Package          string   `json:"package"`
	Imports          []string `json:"imports"`
	IndexedAt        string   `json:"indexed_at"`
}

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(baseURL string) (client *Client) {
	client = &Client{
		BaseURL:    baseURL,
		HTTPClient: http.DefaultClient,
	}
	return client
}

func (c *Client) Search(query string, limit int) (results []CodeDocument, err error) {
	req := SearchRequest{Query: query, Limit: limit}

	var body []byte
	body, err = json.Marshal(req)
	if err != nil {
		return results, err
	}

	var resp *http.Response
	resp, err = c.HTTPClient.Post(
		c.BaseURL+"/api/v1/search",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return results, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("search failed: %s", resp.Status)
		return results, err
	}

	err = json.NewDecoder(resp.Body).Decode(&results)
	return results, err
}

func (c *Client) Reindex() (err error) {
	var resp *http.Response
	resp, err = c.HTTPClient.Post(c.BaseURL+"/api/v1/reindex", "", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		err = fmt.Errorf("reindex failed: %s", resp.Status)
		return err
	}

	return err
}

func main() {
	client := NewClient("http://localhost:8080")

	results, err := client.Search("http handler", 10)
	if err != nil {
		panic(err)
	}

	for _, result := range results {
		fmt.Printf("%s/%s - %s\n", result.Repo, result.FilePath, result.FunctionName)
	}
}
```

### GitHub Actions Integration

**Automatic reindex on push:**

```yaml
name: Reindex Code

on:
  push:
    branches: [main]
    paths:
      - '**.go'

jobs:
  reindex:
    runs-on: ubuntu-latest
    steps:
      - name: Trigger Reindex
        run: |
          curl -f -X POST ${{ secrets.CODE_INDEXER_URL }}/api/v1/reindex
```

### Slack Bot

**Search from Slack:**

```python
from slack_bolt import App
import requests

app = App(token=os.environ["SLACK_BOT_TOKEN"])

@app.command("/codesearch")
def handle_search(ack, command, respond):
    ack()

    query = command["text"]
    resp = requests.post(
        "http://code-indexer:8080/api/v1/search",
        json={"query": query, "limit": 3}
    )
    results = resp.json()

    if not results:
        respond("No results found")
        return

    blocks = []
    for result in results:
        blocks.append({
            "type": "section",
            "text": {
                "type": "mrkdwn",
                "text": f"*{result['repo']}/{result['file_path']}* - `{result['function_name']}`\n```{result['code'][:500]}```"
            }
        })

    respond(blocks=blocks)

app.start(port=3000)
```

## Rate Limiting

Currently no rate limiting. For production:

1. Use API gateway (Kong, Traefik, etc.)
2. Configure rate limits based on usage
3. Monitor `/metrics` for request volume

Example Traefik rate limit:

```yaml
http:
  middlewares:
    rate-limit:
      rateLimit:
        average: 100
        burst: 50
```

## Caching

Search results are not cached. Elasticsearch handles query caching internally.

**For better performance:**
- Increase Elasticsearch query cache size
- Use Elasticsearch request cache
- Implement application-level cache (Redis) if needed

## Error Handling

### Client Errors (4xx)

**400 Bad Request:**
```json
{
  "error": "Query is required"
}
```

Cause: Missing or invalid parameters

**405 Method Not Allowed:**
```
Method not allowed
```

Cause: Wrong HTTP method (e.g., GET on POST endpoint)

### Server Errors (5xx)

**500 Internal Server Error:**
```
Search failed
```

Cause: Elasticsearch error, network issue, or internal bug

**503 Service Unavailable:**
```
Elasticsearch unavailable
```

Cause: Cannot connect to Elasticsearch (readiness check only)

### Retry Strategy

For 5xx errors:
- Wait 1-5 seconds
- Retry up to 3 times
- Use exponential backoff

The indexer already implements retry logic for Elasticsearch internally.

## Versioning

API version is in the path: `/api/v1/`

**Compatibility promise:**
- v1 endpoints remain backward compatible
- New fields may be added to responses (clients should ignore unknown fields)
- Breaking changes require new version (v2)

## CORS

Not enabled by default. If needed, add reverse proxy with CORS headers:

**nginx:**

```nginx
location /api/ {
    proxy_pass http://code-indexer:8080;

    add_header Access-Control-Allow-Origin *;
    add_header Access-Control-Allow-Methods "GET, POST, OPTIONS";
    add_header Access-Control-Allow-Headers "Content-Type";

    if ($request_method = OPTIONS) {
        return 204;
    }
}
```

## WebSocket Support

Not currently supported. All endpoints are HTTP REST.

## GraphQL Support

Not currently supported. Use REST API.

## Next Steps

- Read [Deployment Guide](deployment.md) for setup instructions
- See [examples/](../examples/) for deployment templates
- Check [CLAUDE.md](../CLAUDE.md) for project architecture
- Open issues on GitHub for API feature requests