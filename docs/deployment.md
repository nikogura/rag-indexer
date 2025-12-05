# Deployment Guide

This guide covers deploying the code indexer in various environments.

## Overview

The code indexer can run in three modes:

- **serve** - Long-running service with HTTP API and periodic reindexing
- **index** - One-shot indexing (useful for CI/CD or cron)
- **search** - CLI search tool

## Prerequisites

- **Elasticsearch 8.x** (or compatible like OpenSearch)
- **Git repositories** to index
- **Git authentication** (SSH key or token for private repos)

## Configuration

All configuration is via environment variables (12-factor app):

### Required

| Variable | Description | Example |
|----------|-------------|---------|
| `ES_HOST` | Elasticsearch endpoint | `http://localhost:9200` |
| `REPOS_PATH` | Directory containing repos | `/repos` |

### Git Cloning Mode

| Variable | Description | Example |
|----------|-------------|---------|
| `GIT_ORG` | GitHub organization | `myorg` |
| `GIT_REPOS` | Comma-separated repo list | `repo1,repo2,repo3` |
| `GIT_URL_FORMAT` | URL template | `git@github.com:{org}/{repo}.git` |

### Git Authentication

| Variable | Description | Example |
|----------|-------------|---------|
| `GIT_SSH_KEY_PATH` | Path to SSH private key | `/etc/git-secret/id_ed25519` |
| `GIT_TOKEN` | GitHub personal access token | `ghp_...` |
| `GIT_SSH_COMMAND` | Custom SSH command | `ssh -i /key -o StrictHostKeyChecking=yes` |

### Optional

| Variable | Default | Description |
|----------|---------|-------------|
| `ES_INDEX` | `code-index` | Elasticsearch index name |
| `ES_USERNAME` | - | Basic auth username |
| `ES_PASSWORD` | - | Basic auth password |
| `INDEX_INTERVAL` | `5m` | Reindex interval (serve mode) |
| `HTTP_ADDR` | `:8080` | Listen address (serve mode) |

## Deployment Scenarios

### Kubernetes (Recommended for Production)

See [examples/kubernetes/](../examples/kubernetes/) for complete manifests.

**Features:**
- Automatic restarts
- Health checks
- Prometheus metrics
- Secret management
- Resource limits

**Quick deploy:**

```bash
cd examples/kubernetes
kubectl apply -f elasticsearch.yaml
kubectl apply -f secrets-example.yaml  # Edit first!
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
kubectl apply -f servicemonitor.yaml  # If using Prometheus Operator
```

**Production considerations:**
- Use proper Elasticsearch cluster (not single node)
- Manage secrets with Vault or External Secrets Operator
- Set appropriate resource requests/limits
- Enable Pod Security Standards
- Use network policies to restrict traffic

### Docker Compose (Local Development)

See [examples/docker-compose/](../examples/docker-compose/) for complete setup.

**Features:**
- Quick local setup
- Integrated Elasticsearch
- Volume persistence
- Hot reload during development

**Quick start:**

```bash
cd examples/docker-compose
cp .env.example .env
# Edit .env with your repos
docker-compose up -d
docker-compose logs -f code-indexer
```

### Systemd (Bare Metal)

**1. Build binary:**

```bash
go build -o /usr/local/bin/code-indexer .
```

**2. Create service user:**

```bash
useradd --system --no-create-home --shell /usr/sbin/nologin code-indexer
```

**3. Create directories:**

```bash
mkdir -p /var/lib/code-indexer/repos
chown code-indexer:code-indexer /var/lib/code-indexer
```

**4. Create systemd unit:**

`/etc/systemd/system/code-indexer.service`:

```ini
[Unit]
Description=Code Indexer
After=network.target elasticsearch.service
Wants=elasticsearch.service

[Service]
Type=simple
User=code-indexer
Group=code-indexer
WorkingDirectory=/var/lib/code-indexer

Environment="ES_HOST=http://localhost:9200"
Environment="REPOS_PATH=/var/lib/code-indexer/repos"
Environment="GIT_ORG=myorg"
Environment="GIT_REPOS=repo1,repo2,repo3"
Environment="GIT_SSH_KEY_PATH=/var/lib/code-indexer/.ssh/id_ed25519"
Environment="INDEX_INTERVAL=5m"

ExecStart=/usr/local/bin/code-indexer -mode serve

Restart=always
RestartSec=10

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/code-indexer

[Install]
WantedBy=multi-user.target
```

**5. Start service:**

```bash
systemctl daemon-reload
systemctl enable code-indexer
systemctl start code-indexer
systemctl status code-indexer
```

**6. View logs:**

```bash
journalctl -u code-indexer -f
```

### CI/CD Pipeline

Use **index mode** for one-shot indexing after code changes:

**GitHub Actions:**

```yaml
name: Index Code

on:
  push:
    branches: [main]

jobs:
  index:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Download indexer
        run: |
          curl -L https://github.com/myorg/code-indexer/releases/latest/download/code-indexer-linux-amd64 -o code-indexer
          chmod +x code-indexer

      - name: Run indexing
        env:
          ES_HOST: ${{ secrets.ES_HOST }}
          ES_USERNAME: ${{ secrets.ES_USERNAME }}
          ES_PASSWORD: ${{ secrets.ES_PASSWORD }}
          REPOS_PATH: ${{ github.workspace }}
        run: ./code-indexer -mode index
```

**GitLab CI:**

```yaml
index:
  stage: deploy
  image: golang:1.22
  script:
    - go build -o code-indexer .
    - ./code-indexer -mode index
  variables:
    ES_HOST: $ES_HOST
    ES_USERNAME: $ES_USERNAME
    ES_PASSWORD: $ES_PASSWORD
    REPOS_PATH: $CI_PROJECT_DIR
  only:
    - main
```

### Flux Integration

If using Flux GitRepository CRDs, mount the Flux repos directly:

```yaml
volumes:
- name: repos
  persistentVolumeClaim:
    claimName: flux-system

volumeMounts:
- name: repos
  mountPath: /repos
  subPath: repos  # Adjust based on your Flux setup
```

Then omit `GIT_ORG` and `GIT_REPOS`. The indexer will scan existing repos.

## Git Authentication Methods

### SSH Keys (Recommended)

**Generate deploy key:**

```bash
ssh-keygen -t ed25519 -f deploy_key -C "code-indexer"
```

**Add to GitHub:**
- Go to repo Settings → Deploy keys
- Add public key (deploy_key.pub)
- Grant read-only access

**Mount in container/pod:**

```yaml
volumes:
- name: git-secret
  secret:
    secretName: git-ssh-key
    defaultMode: 0400
```

**Configure:**

```bash
GIT_SSH_KEY_PATH=/etc/git-secret/id_ed25519
```

### Personal Access Token

**Create token:**
- GitHub Settings → Developer settings → Personal access tokens
- Generate with `repo` scope (or `public_repo` for public repos only)

**Configure:**

```bash
GIT_TOKEN=ghp_your_token_here
GIT_URL_FORMAT=https://github.com/{org}/{repo}.git
```

Token is automatically embedded in clone URLs.

### GitHub App (Future)

Not yet implemented. Use PAT or SSH for now.

## Elasticsearch Setup

### Self-Hosted

**Docker:**

```bash
docker run -d \
  --name elasticsearch \
  -e "discovery.type=single-node" \
  -e "xpack.security.enabled=false" \
  -p 9200:9200 \
  docker.elastic.co/elasticsearch/elasticsearch:8.11.0
```

**Configuration:**

```bash
ES_HOST=http://localhost:9200
```

### Elastic Cloud

**Configure:**

```bash
ES_HOST=https://my-deployment.es.us-east-1.aws.found.io:9243
ES_USERNAME=elastic
ES_PASSWORD=changeme
```

### AWS OpenSearch

**Configure:**

```bash
ES_HOST=https://search-my-domain.us-east-1.es.amazonaws.com
ES_USERNAME=admin
ES_PASSWORD=changeme
```

OpenSearch is API-compatible with Elasticsearch.

## Security Considerations

### Network Security

**Kubernetes:**
- Use NetworkPolicies to restrict Elasticsearch access
- Don't expose Elasticsearch publicly
- Use TLS for production

**Docker:**
- Use bridge networks, not host networking
- Don't bind ES to 0.0.0.0 if not needed

### Secret Management

**Kubernetes:**
- Use Sealed Secrets, Vault, or External Secrets Operator
- Never commit secrets to git
- Rotate credentials regularly

**Docker:**
- Use Docker secrets or env files
- Don't put secrets in docker-compose.yaml
- Use `.env` file (add to .gitignore)

### SSH Key Security

**Best practices:**
- Use deploy keys (read-only)
- One key per deployment environment
- Rotate keys periodically
- Set file permissions to 0400
- Never commit private keys

### Elasticsearch Authentication

**If using ES auth:**
- Create dedicated service account
- Grant minimal permissions (index + search on one index)
- Use TLS in production
- Rotate passwords regularly

## Monitoring

### Health Checks

**Liveness probe:**
```bash
curl http://localhost:8080/health
```

Always returns 200 OK if process is running.

**Readiness probe:**
```bash
curl http://localhost:8080/ready
```

Returns 200 OK if Elasticsearch is reachable, 503 otherwise.

### Prometheus Metrics

**Endpoint:**
```bash
curl http://localhost:8080/metrics
```

**Key metrics:**
- `code_indexer_functions_indexed_total{repo}` - Total functions indexed
- `code_indexer_repos_indexed_total` - Total repos indexed
- `code_indexer_indexing_duration_seconds{repo}` - Time to index repo
- `code_indexer_elasticsearch_requests_total{operation,status}` - ES request stats
- `code_indexer_last_successful_index_timestamp{repo}` - Last successful index time

**Alerts:**

```yaml
groups:
- name: code-indexer
  rules:
  - alert: IndexingStale
    expr: time() - code_indexer_last_successful_index_timestamp > 3600
    for: 5m
    annotations:
      summary: "Code index is stale"

  - alert: ElasticsearchErrors
    expr: rate(code_indexer_elasticsearch_requests_total{status="error"}[5m]) > 0.1
    for: 5m
    annotations:
      summary: "High Elasticsearch error rate"
```

### Logging

**Structured JSON logs to stdout:**

```json
{"time":"2025-10-30T10:30:00Z","level":"INFO","msg":"Starting HTTP server","address":":8080"}
{"time":"2025-10-30T10:30:05Z","level":"INFO","msg":"Indexing repository","repo":"kms"}
{"time":"2025-10-30T10:30:10Z","level":"WARN","msg":"Failed to parse file","repo":"kms","file":"internal/broken.go","error":"syntax error"}
```

**Integrate with log aggregation:**
- Loki (Kubernetes)
- CloudWatch Logs (AWS)
- Elasticsearch (via Filebeat)

## Troubleshooting

### Git Clone Failures

**Symptoms:**
- "Permission denied" errors
- "Host key verification failed"

**Checks:**

```bash
# Test SSH connectivity
ssh -T git@github.com

# Check key permissions
ls -la /path/to/key  # Should be 0400 or 0600

# Test with verbose SSH
GIT_SSH_COMMAND="ssh -vvv" ./code-indexer -mode index
```

**Solutions:**
- Ensure deploy key is added to repos
- Check SSH key has correct permissions
- Verify GIT_SSH_KEY_PATH is correct
- For StrictHostKeyChecking=yes, ensure known_hosts contains github.com

### Elasticsearch Connection Issues

**Symptoms:**
- "Connection refused"
- "503 Service Unavailable" on /ready

**Checks:**

```bash
# Test ES directly
curl http://elasticsearch:9200

# Check ES health
curl http://elasticsearch:9200/_cluster/health?pretty

# Check ES logs
kubectl logs -l app=elasticsearch
docker-compose logs elasticsearch
```

**Solutions:**
- Verify ES_HOST is correct
- Check ES is running and healthy
- Verify network connectivity
- Check ES authentication if enabled

### Index Creation Failures

**Symptoms:**
- "Failed to create index" errors
- Indexer crashes on startup

**Checks:**

```bash
# Check index exists
curl http://elasticsearch:9200/_cat/indices?v

# Check index mapping
curl http://elasticsearch:9200/code-index/_mapping?pretty
```

**Solutions:**
- Delete and recreate: `curl -X DELETE http://elasticsearch:9200/code-index`
- Check ES has sufficient disk space
- Verify ES user has create index permission

### High Memory Usage

**Symptoms:**
- OOMKilled pods
- Slow indexing

**Solutions:**
- Increase memory limits in deployment
- Reduce ES heap size if needed
- Index fewer repos simultaneously
- Implement batch processing (future enhancement)

### Slow Indexing

**Symptoms:**
- Takes hours to index
- High CPU usage

**Causes:**
- Large codebases
- Many repos
- Slow network to ES

**Solutions:**
- Increase INDEX_INTERVAL to reduce frequency
- Use incremental indexing (future enhancement)
- Increase ES resources
- Use local/nearby ES instance

## Scaling Considerations

### Single Replica

Currently, indexing uses a mutex - only run 1 replica.

**Reasons:**
- Prevents concurrent indexing conflicts
- Simplifies state management
- ES is the source of truth

**Future:** Leader election for multi-replica deployments.

### Large Codebases

**Recommendations:**

| Repo Count | ES Specs | Indexer Specs |
|------------|----------|---------------|
| < 10 | 2GB RAM, 1 CPU | 256MB RAM, 100m CPU |
| 10-50 | 4GB RAM, 2 CPU | 512MB RAM, 500m CPU |
| 50-100 | 8GB RAM, 4 CPU | 1GB RAM, 1 CPU |
| 100+ | ES cluster | Multiple indexers (future) |

### Elasticsearch Sizing

**Index size estimation:**
- Small codebase (50k LOC): ~50MB
- Medium (500k LOC): ~500MB
- Large (5M LOC): ~5GB

**Disk requirements:**
- 2x index size minimum (for merges)
- Add 50% buffer for growth
- Example: 5GB index → 15GB disk

## Backup and Recovery

### No Backups Needed

**Rationale:**
- Git is the source of truth
- Index is derived data
- Can rebuild from repos in minutes

**Process:**
1. Delete index: `curl -X DELETE http://es:9200/code-index`
2. Restart indexer: `kubectl rollout restart deployment/code-indexer`
3. Index is automatically recreated

### Optional: Export for Analysis

```bash
# Export all indexed functions
curl -X POST http://es:9200/code-index/_search \
  -H "Content-Type: application/json" \
  -d '{"size": 10000}' > functions.json
```

## Maintenance

### Updating the Indexer

**Kubernetes:**

```bash
kubectl set image deployment/code-indexer code-indexer=code-indexer:v2.0.0
kubectl rollout status deployment/code-indexer
```

**Docker Compose:**

```bash
docker-compose pull code-indexer
docker-compose up -d code-indexer
```

**Systemd:**

```bash
systemctl stop code-indexer
cp new-binary /usr/local/bin/code-indexer
systemctl start code-indexer
```

### Reindexing Everything

**API:**

```bash
curl -X POST http://localhost:8080/api/v1/reindex
```

**Manual:**

```bash
# Delete index
curl -X DELETE http://es:9200/code-index

# Restart indexer (it will recreate and populate)
kubectl rollout restart deployment/code-indexer
```

### Adding Repos

**Update configuration:**

```bash
# Kubernetes: edit deployment
kubectl edit deployment code-indexer

# Add to GIT_REPOS environment variable
GIT_REPOS=repo1,repo2,repo3,new-repo

# Save and pods will restart automatically
```

No manual reindex needed - new repos are picked up on next indexing cycle.

### Removing Repos

**Update configuration first, then clean index:**

```bash
# Remove from GIT_REPOS
kubectl edit deployment code-indexer

# Delete docs for removed repo
curl -X POST http://es:9200/code-index/_delete_by_query \
  -H "Content-Type: application/json" \
  -d '{"query": {"term": {"repo": "old-repo"}}}'
```

## Performance Tuning

### Elasticsearch

**Increase refresh interval (faster indexing):**

```bash
curl -X PUT http://es:9200/code-index/_settings \
  -H "Content-Type: application/json" \
  -d '{"index": {"refresh_interval": "60s"}}'
```

**Disable replicas for single-node:**

```bash
curl -X PUT http://es:9200/code-index/_settings \
  -H "Content-Type: application/json" \
  -d '{"index": {"number_of_replicas": 0}}'
```

### Indexer

**Increase INDEX_INTERVAL (reduce load):**

```bash
INDEX_INTERVAL=15m  # Instead of default 5m
```

**Reduce git operation timeouts (fail faster):**

Edit `pkg/indexer/git.go` constants if needed (not recommended).

## Next Steps

- Read [API Documentation](api.md) for integration details
- See [examples/](../examples/) for deployment templates
- Check [CLAUDE.md](../CLAUDE.md) for project architecture
- Open issues on GitHub for bugs or feature requests
