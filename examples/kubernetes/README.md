# Kubernetes Deployment

This directory contains Kubernetes manifests for deploying the code indexer.

## Files

- `elasticsearch.yaml` - Single-node Elasticsearch StatefulSet and Service
- `deployment.yaml` - Code indexer Deployment
- `service.yaml` - Code indexer Service
- `servicemonitor.yaml` - Prometheus ServiceMonitor (requires Prometheus Operator)
- `secrets-example.yaml` - Example Secret manifests (DO NOT use as-is)

## Quick Start

1. Create secrets with your git SSH key:

```bash
kubectl create secret generic git-ssh-key \
  --from-file=id_ed25519=$HOME/.ssh/id_ed25519
```

2. Deploy Elasticsearch:

```bash
kubectl apply -f elasticsearch.yaml
```

3. Wait for Elasticsearch to be ready:

```bash
kubectl wait --for=condition=ready pod -l app=elasticsearch --timeout=300s
```

4. Update `deployment.yaml` with your configuration:
   - Set `GIT_ORG` to your GitHub organization
   - Set `GIT_REPOS` to comma-separated list of repos
   - Adjust resource limits as needed

5. Deploy code indexer:

```bash
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
```

6. Verify deployment:

```bash
kubectl get pods -l app=code-indexer
kubectl logs -l app=code-indexer -f
```

## Configuration

All configuration is done via environment variables in `deployment.yaml`:

**Required:**
- `ES_HOST` - Elasticsearch endpoint
- `REPOS_PATH` - Directory for cloned repos
- `GIT_ORG` - GitHub organization
- `GIT_REPOS` - Comma-separated repo list

**Optional:**
- `ES_USERNAME` / `ES_PASSWORD` - Elasticsearch auth
- `GIT_URL_FORMAT` - Git URL template (default: git@github.com:{org}/{repo}.git)
- `GIT_SSH_KEY_PATH` - Path to SSH key (default: /etc/git-secret/id_ed25519)
- `INDEX_INTERVAL` - Reindex interval (default: 5m)
- `HTTP_ADDR` - Listen address (default: :8080)

## Using Existing Repos (Flux Integration)

If you're using Flux or another tool that clones repos, mount the existing repos instead:

```yaml
volumes:
- name: repos
  persistentVolumeClaim:
    claimName: flux-system  # Your existing PVC
```

Then omit `GIT_ORG` and `GIT_REPOS` environment variables. The indexer will scan whatever is in `REPOS_PATH`.

## Prometheus Monitoring

If you have Prometheus Operator installed:

```bash
kubectl apply -f servicemonitor.yaml
```

This will automatically scrape metrics from `/metrics` every 30s.

## Production Considerations

1. **Elasticsearch sizing**: Single node is fine for < 50 repos. For larger deployments, use a proper ES cluster.

2. **Resource limits**: Adjust based on your repo sizes. Indexing is CPU-intensive.

3. **Persistent storage**: Use a PVC instead of emptyDir if you want to avoid reindexing on pod restart:

```yaml
volumes:
- name: repos
  persistentVolumeClaim:
    claimName: code-indexer-repos
```

4. **SSH key security**: Use Vault Secrets Operator or External Secrets Operator for better secret management.

5. **Multiple replicas**: The indexer uses a mutex to prevent concurrent indexing. Only run 1 replica for now.

## Troubleshooting

**Pods not starting:**
```bash
kubectl describe pod -l app=code-indexer
kubectl logs -l app=code-indexer
```

**Git clone failures:**
```bash
# Check SSH key is mounted correctly
kubectl exec -it <pod-name> -- ls -la /etc/git-secret

# Test git connectivity
kubectl exec -it <pod-name> -- ssh -T git@github.com
```

**Elasticsearch connection issues:**
```bash
# Check ES is reachable
kubectl exec -it <pod-name> -- curl http://elasticsearch:9200

# Check ES logs
kubectl logs -l app=elasticsearch
```

**Index not created:**
The indexer automatically creates the index on startup. Check logs for errors.
