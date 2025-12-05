# Docker Compose Deployment

This directory contains Docker Compose configuration for local development and testing.

## Quick Start

1. Copy the example environment file:

```bash
cp .env.example .env
```

2. Edit `.env` with your configuration:

```bash
GIT_ORG=myorg
GIT_REPOS=repo1,repo2,repo3
```

3. Build and start services:

```bash
docker-compose up -d
```

4. Watch logs:

```bash
docker-compose logs -f code-indexer
```

5. Access services:
   - Code indexer API: http://localhost:8080
   - Elasticsearch: http://localhost:9200
   - Metrics: http://localhost:8080/metrics

## Configuration

### Using SSH Keys (Default)

Mount your SSH directory (read-only):

```yaml
volumes:
  - ~/.ssh:/root/.ssh:ro
```

Make sure your SSH key has access to the repos.

### Using GitHub Token

Set environment variables:

```bash
GIT_TOKEN=ghp_your_token_here
GIT_URL_FORMAT=https://github.com/{org}/{repo}.git
```

The token will be embedded in the clone URL automatically.

### Using Local Repos

If you already have repos checked out locally, mount them instead:

```yaml
volumes:
  - /path/to/your/repos:/repos:ro
```

Then remove `GIT_ORG` and `GIT_REPOS` environment variables. The indexer will scan whatever is in `/repos`.

## Building the Image

The docker-compose file expects a Dockerfile in the project root. Create one:

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o code-indexer .

FROM alpine:latest
RUN apk --no-cache add ca-certificates git openssh-client
WORKDIR /app
COPY --from=builder /build/code-indexer .
EXPOSE 8080
ENTRYPOINT ["./code-indexer"]
```

## API Examples

### Search

```bash
curl -X POST http://localhost:8080/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{"query": "error handling", "limit": 10}'
```

### Trigger Reindex

```bash
curl -X POST http://localhost:8080/api/v1/reindex
```

### Health Check

```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

### View Metrics

```bash
curl http://localhost:8080/metrics
```

## Elasticsearch Management

### Query the index directly

```bash
curl http://localhost:9200/code-index/_search?pretty
```

### Delete the index

```bash
curl -X DELETE http://localhost:9200/code-index
```

The indexer will recreate it on next startup.

### View index mapping

```bash
curl http://localhost:9200/code-index/_mapping?pretty
```

## Troubleshooting

### Git clone failures

```bash
# Check SSH connectivity
docker-compose exec code-indexer ssh -T git@github.com

# Check SSH key permissions
docker-compose exec code-indexer ls -la /root/.ssh
```

### Elasticsearch not ready

```bash
# Check ES health
curl http://localhost:9200/_cluster/health?pretty

# Check ES logs
docker-compose logs elasticsearch
```

### Container keeps restarting

```bash
# Check logs
docker-compose logs code-indexer

# Run one-shot index to debug
docker-compose run --rm code-indexer -mode index
```

## Development Workflow

### Make code changes

```bash
# Rebuild and restart
docker-compose up -d --build code-indexer

# Watch logs
docker-compose logs -f code-indexer
```

### Run tests

```bash
docker-compose run --rm code-indexer go test ./...
```

### Run linters

```bash
docker-compose run --rm code-indexer make lint
```

## Stopping Services

```bash
# Stop services (keeps data)
docker-compose stop

# Stop and remove containers (keeps data)
docker-compose down

# Remove everything including volumes
docker-compose down -v
```

## Resource Requirements

- **RAM**: 1GB minimum (512MB for ES + 256MB for indexer)
- **Disk**: 10GB minimum for ES data
- **CPU**: 1 core minimum

For large codebases, increase Elasticsearch heap:

```yaml
environment:
  - ES_JAVA_OPTS=-Xms1g -Xmx1g
```
