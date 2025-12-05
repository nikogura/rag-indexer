# Code Indexer

A universal tool to index Go codebases into Elasticsearch, enabling searchable code examples for developers, AI coding assistants, and code quality analysis.

## Features

- **Universal deployment** - Works in Kubernetes, Docker, systemd, or bare metal
- **Flexible git integration** - Clone repos itself OR scan existing checkouts
- **Any Elasticsearch** - Self-hosted, Elastic Cloud, AWS OpenSearch, local docker
- **Production grade** - Metrics, health checks, graceful shutdown, proper error handling
- **12-factor app** - All configuration via environment variables
- **AST-based parsing** - Uses go/parser for accurate code analysis
- **Named returns detection** - Finds examples following best practices
- **Error handling detection** - Identifies functions with proper error handling
- **Prometheus metrics** - Full observability support
- **SSH & token auth** - Multiple git authentication methods

## Quick Start

### Docker Compose (Fastest)

```bash
git clone https://github.com/nikogura/rag-indexer.git
cd rag-indexer/examples/docker-compose
cp .env.example .env
# Edit .env with your repos
docker-compose up -d
```

Access API at http://localhost:8080

### Binary

```bash
# Build
go build -o code-indexer .

# Configure
export ES_HOST=http://localhost:9200
export REPOS_PATH=$HOME/src/myorg
export GIT_ORG=myorg
export GIT_REPOS=repo1,repo2,repo3

# Run
./code-indexer -mode serve
```

### Kubernetes

```bash
cd examples/kubernetes
kubectl apply -f elasticsearch.yaml
kubectl apply -f secrets-example.yaml  # Edit first!
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
```

## Run Modes

### Serve Mode (Long-Running Service)

```bash
./code-indexer -mode serve
```

- Clones/updates repos on startup
- Runs initial indexing
- Starts HTTP API server
- Periodic reindexing in background
- Exposes Prometheus metrics

### Index Mode (One-Shot)

```bash
./code-indexer -mode index
```

- Index repos once and exit
- Useful for CI/CD, cron jobs
- Exit code indicates success/failure

### Search Mode (CLI)

```bash
./code-indexer -mode search "error handling http"
```

- Search from command line
- Prints results to stdout
- Useful for testing, scripting

## Configuration

All configuration via environment variables:

### Required

```bash
ES_HOST=http://localhost:9200      # Elasticsearch endpoint
REPOS_PATH=/repos                  # Directory containing repos
```

### Git Cloning Mode

```bash
GIT_ORG=myorg                      # GitHub organization
GIT_REPOS=repo1,repo2,repo3        # Comma-separated repo list
GIT_URL_FORMAT=git@github.com:{org}/{repo}.git  # URL template
```

### Git Authentication

```bash
# Option 1: SSH key
GIT_SSH_KEY_PATH=/etc/git-secret/id_ed25519

# Option 2: Personal access token
GIT_TOKEN=ghp_your_token_here

# Option 3: Custom SSH command
GIT_SSH_COMMAND="ssh -i /key -o StrictHostKeyChecking=yes"
```

### Optional

```bash
ES_INDEX=code-index                # Index name (default: code-index)
ES_USERNAME=elastic                # Basic auth username
ES_PASSWORD=changeme               # Basic auth password
INDEX_INTERVAL=5m                  # Reindex interval (default: 5m)
HTTP_ADDR=:8080                    # Listen address (default: :8080)
```

## API Endpoints

### Search

```bash
curl -X POST http://localhost:8080/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{"query": "http handler error", "limit": 10}'
```

Returns functions matching the query, prioritizing:
1. Functions with named returns
2. Functions with error handling
3. Text relevance score

### Reindex

```bash
curl -X POST http://localhost:8080/api/v1/reindex
```

Triggers background reindex of all repos.

### Health Checks

```bash
curl http://localhost:8080/health  # Liveness probe
curl http://localhost:8080/ready   # Readiness probe
```

### Metrics

```bash
curl http://localhost:8080/metrics  # Prometheus metrics
```

## Search Examples

**Natural language:**
```bash
"http handler with error handling"
"context timeout examples"
"database transaction patterns"
```

**Function name:**
```bash
"NewClient"
"ParseConfig"
"HandleRequest"
```

**Package specific:**
```bash
"handlers authentication"
"kubernetes client"
"elasticsearch query"
```

## Architecture

```
pkg/
  config/       - Environment variable loading
  indexer/      - Core indexing logic, git operations, AST parsing
  elasticsearch/ - ES client with retry logic
  server/       - HTTP API server
  metrics/      - Prometheus metrics
  logging/      - Structured logging interface

examples/
  kubernetes/   - K8s deployment manifests
  docker-compose/ - Docker compose setup

docs/
  deployment.md - Deployment guide
  api.md        - API documentation
```

## Code Quality

This project exemplifies best practices:

- **✓ Zero linting violations** - Passes golangci-lint with strict config
- **✓ Named returns** - All functions use named return values
- **✓ Proper error handling** - Context propagation, wrapped errors
- **✓ No regex parsing** - Uses go/parser and go/ast
- **✓ Context cancellation** - All operations respect context
- **✓ Retry logic** - Exponential backoff for transient failures
- **✓ Structured logging** - JSON logs with levels
- **✓ Metrics** - Prometheus instrumentation
- **✓ Health checks** - Liveness and readiness probes

## Development

### Prerequisites

- Go 1.22+
- Elasticsearch 8.x
- golangci-lint
- namedreturns linter

### Build

```bash
go build -o code-indexer .
```

### Lint

```bash
make lint
```

Runs namedreturns linter first, then golangci-lint.

### Test

```bash
go test ./...
```

### Docker Build

```bash
docker build -t code-indexer:latest .
```

## Prometheus Metrics

- `code_indexer_functions_indexed_total{repo}` - Functions indexed per repo
- `code_indexer_repos_indexed_total` - Total repos indexed
- `code_indexer_indexing_duration_seconds{repo}` - Time to index repo
- `code_indexer_parse_errors_total{repo,file}` - Parse failures
- `code_indexer_elasticsearch_requests_total{operation,status}` - ES request stats
- `code_indexer_last_successful_index_timestamp{repo}` - Last successful index

## Elasticsearch Setup

The indexer automatically creates the index with proper mapping on startup.

**Manual creation (optional):**

```bash
curl -X PUT "localhost:9200/code-index" -H 'Content-Type: application/json' -d'
{
  "mappings": {
    "properties": {
      "repo": {"type": "keyword"},
      "file_path": {"type": "keyword"},
      "function_name": {"type": "keyword"},
      "code": {"type": "text"},
      "has_namedreturns": {"type": "boolean"},
      "has_error_handling": {"type": "boolean"},
      "package": {"type": "keyword"},
      "imports": {"type": "keyword"}
    }
  }
}
'
```

## Deployment Scenarios

### Production Kubernetes

- Proper ES cluster (not single node)
- Vault/External Secrets for credentials
- Resource requests and limits
- NetworkPolicies
- Pod Security Standards
- Prometheus scraping via ServiceMonitor

See [docs/deployment.md](docs/deployment.md) for details.

### Local Development

Use docker-compose for quick setup with integrated Elasticsearch.

See [examples/docker-compose/](examples/docker-compose/) for details.

### CI/CD Integration

Use **index mode** to index code after push:

```yaml
- name: Index Code
  env:
    ES_HOST: ${{ secrets.ES_HOST }}
    REPOS_PATH: ${{ github.workspace }}
  run: ./code-indexer -mode index
```

### Flux Integration

Mount Flux's git repos directly:

```yaml
volumes:
- name: repos
  persistentVolumeClaim:
    claimName: flux-system
```

Omit `GIT_ORG` and `GIT_REPOS`. The indexer scans existing repos.

## Security

- **SSH keys**: Use deploy keys (read-only), mount with 0400 permissions
- **Tokens**: Store in secrets management, rotate regularly
- **ES auth**: Use dedicated service account with minimal permissions
- **Network**: Don't expose ES or indexer publicly without auth
- **TLS**: Use TLS for production ES connections

## Troubleshooting

### Git clone failures

```bash
# Test SSH connectivity
ssh -T git@github.com

# Check key permissions
ls -la $GIT_SSH_KEY_PATH  # Should be 0400

# Verbose SSH
GIT_SSH_COMMAND="ssh -vvv" ./code-indexer -mode index
```

### ES connection issues

```bash
# Test ES directly
curl $ES_HOST

# Check ES health
curl $ES_HOST/_cluster/health?pretty
```

### Index not created

Check logs - the indexer auto-creates on startup. If it fails, ES may be out of disk or have permission issues.

## Performance

**Typical indexing speeds:**
- Small repo (50k LOC): ~2-5 seconds
- Medium repo (500k LOC): ~10-30 seconds
- Large repo (5M LOC): ~2-5 minutes

**ES sizing:**
- < 10 repos: 2GB RAM, 1 CPU
- 10-50 repos: 4GB RAM, 2 CPU
- 50-100 repos: 8GB RAM, 4 CPU
- 100+ repos: ES cluster

## Limitations

- **Go only** - Currently only indexes Go code (multi-language support planned)
- **Single replica** - Uses mutex, only run 1 replica (leader election planned)
- **No incremental indexing** - Reindexes entire repo (git diff-based indexing planned)
- **Simple lint detection** - Placeholder (golangci-lint integration planned)

## Roadmap

- [ ] Multi-language support (Python, Rust, TypeScript)
- [ ] Incremental indexing via git diff
- [ ] Lint compliance detection (golangci-lint integration)
- [ ] Semantic search with embeddings
- [ ] Web UI for search
- [ ] Usage analytics
- [ ] Leader election for multi-replica
- [ ] GitHub App authentication

## Documentation

- [Deployment Guide](docs/deployment.md) - Detailed deployment instructions
- [API Documentation](docs/api.md) - Complete API reference
- [CLAUDE.md](CLAUDE.md) - Project specification and architecture

## Contributing

Contributions welcome! Please:

1. Follow existing code style (use golangci-lint)
2. Add tests for new features
3. Update documentation
4. Ensure `make lint` passes
5. Use named returns in all functions

## License

Apache License 2.0 - see [LICENSE](LICENSE) file for details.

Copyright 2025 Nik Ogura

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

## Support

- Issues: https://github.com/nikogura/rag-indexer/issues
- Discussions: https://github.com/nikogura/rag-indexer/discussions

## Acknowledgments

Built with:
- [Elasticsearch](https://www.elastic.co/)
- [Prometheus](https://prometheus.io/)
- [Go](https://golang.org/)

Designed for use with:
- [Claude Code](https://claude.com/claude-code)
- [GitHub Copilot](https://github.com/features/copilot)
- [Cursor](https://cursor.sh/)
