// Package indexer handles code indexing operations including repository scanning and parsing.
package indexer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/nikogura/rag-indexer/pkg/config"
	"github.com/nikogura/rag-indexer/pkg/elasticsearch"
	"github.com/nikogura/rag-indexer/pkg/logging"
	"github.com/nikogura/rag-indexer/pkg/metrics"
)

// ErrGitConfigRequired is returned when GIT_ORG and GIT_REPOS are not configured.
var ErrGitConfigRequired = errors.New("GIT_ORG and GIT_REPOS must be set for cloning")

// Indexer handles code indexing operations.
type Indexer struct {
	config  config.Config
	es      *elasticsearch.Client
	metrics *metrics.Metrics
	logger  logging.Logger
	mu      sync.Mutex
}

// New creates a new Indexer instance.
func New(cfg config.Config, es *elasticsearch.Client, m *metrics.Metrics, logger logging.Logger) (indexer *Indexer) {
	indexer = &Indexer{
		config:  cfg,
		es:      es,
		metrics: m,
		logger:  logger,
	}
	return indexer
}

// CloneRepos clones or updates git repositories configured in the application.
func (idx *Indexer) CloneRepos(ctx context.Context) (err error) {
	if idx.config.GitOrg == "" || len(idx.config.GitRepos) == 0 {
		err = ErrGitConfigRequired
		return err
	}

	err = os.MkdirAll(idx.config.ReposPath, 0755)
	if err != nil {
		err = fmt.Errorf("failed to create repos directory: %w", err)
		return err
	}

	for _, repo := range idx.config.GitRepos {
		cloneErr := idx.cloneOrUpdateRepo(ctx, repo)
		if cloneErr != nil {
			idx.logger.Warn("Failed to process repository", "repo", repo, "error", cloneErr)
		}
	}

	return err
}

// cloneOrUpdateRepo clones a repo if it doesn't exist, or updates it if it does.
func (idx *Indexer) cloneOrUpdateRepo(ctx context.Context, repo string) (err error) {
	repoURL := buildRepoURL(idx.config.GitURLFormat, idx.config.GitOrg, repo, idx.config.GitToken)
	targetDir := filepath.Join(idx.config.ReposPath, repo)

	var statErr error
	_, statErr = os.Stat(filepath.Join(targetDir, ".git"))
	if statErr == nil {
		idx.logger.Info("Repository already exists, fetching updates", "repo", repo)
		err = gitFetch(ctx, targetDir, idx.config.GitSSHKeyPath, os.Getenv("GIT_SSH_COMMAND"))
		if err != nil {
			err = fmt.Errorf("failed to fetch: %w", err)
			return err
		}
		return err
	}

	idx.logger.Info("Cloning repository", "repo", repo)
	err = gitClone(ctx, repoURL, targetDir, idx.config.GitSSHKeyPath, os.Getenv("GIT_SSH_COMMAND"))
	if err != nil {
		err = fmt.Errorf("failed to clone: %w", err)
		return err
	}

	return err
}

// IndexAllRepos indexes all git repositories found in the configured repos path.
func (idx *Indexer) IndexAllRepos(ctx context.Context) (totalCount int, err error) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	var entries []os.DirEntry
	entries, err = os.ReadDir(idx.config.ReposPath)
	if err != nil {
		err = fmt.Errorf("failed to read repos directory: %w", err)
		return totalCount, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		count, indexErr := idx.indexRepoIfValid(ctx, entry.Name())
		if indexErr != nil {
			idx.logger.Error("Failed to index repository", "repo", entry.Name(), "error", indexErr)
			continue
		}

		totalCount += count
		idx.metrics.ReposIndexed.Inc()
	}

	return totalCount, err
}

// indexRepoIfValid checks if a directory is a valid git repo and indexes it.
func (idx *Indexer) indexRepoIfValid(ctx context.Context, name string) (count int, err error) {
	repoPath := filepath.Join(idx.config.ReposPath, name)

	gitPath := filepath.Join(repoPath, ".git")
	var statErr error
	_, statErr = os.Stat(gitPath)
	if os.IsNotExist(statErr) {
		return count, err
	}

	count, err = idx.IndexRepository(ctx, repoPath)
	if err != nil {
		return count, err
	}

	return count, err
}

// IndexRepository indexes a single repository by walking its file tree.
func (idx *Indexer) IndexRepository(ctx context.Context, repoPath string) (count int, err error) {
	repoName := filepath.Base(repoPath)
	idx.logger.Info("Indexing repository", "repo", repoName)

	start := time.Now()
	count, err = idx.walkAndIndexRepo(ctx, repoName, repoPath)

	duration := time.Since(start)
	idx.metrics.IndexingDuration.WithLabelValues(repoName).Observe(duration.Seconds())
	if err == nil {
		idx.metrics.LastSuccessfulIndex.WithLabelValues(repoName).SetToCurrentTime()
		idx.metrics.FunctionsIndexed.WithLabelValues(repoName).Add(float64(count))
	}

	return count, err
}

// walkAndIndexRepo walks the repository tree and indexes Go files.
func (idx *Indexer) walkAndIndexRepo(ctx context.Context, repoName string, repoPath string) (totalFunctions int, walkErr error) {
	walker := &fileWalker{
		ctx:      ctx,
		es:       idx.es,
		repoName: repoName,
		metrics:  idx.metrics,
		logger:   idx.logger,
	}

	walkErr = filepath.Walk(repoPath, walker.walk)
	totalFunctions = walker.totalCount

	return totalFunctions, walkErr
}

// RunIndexingLoop runs periodic reindexing in the background.
func (idx *Indexer) RunIndexingLoop(ctx context.Context) {
	ticker := time.NewTicker(idx.config.IndexInterval)
	defer ticker.Stop()

	idx.logger.Info("Starting indexing loop", "interval", idx.config.IndexInterval)

	for {
		select {
		case <-ticker.C:
			idx.logger.Info("Running periodic reindex")

			if idx.config.GitOrg != "" && len(idx.config.GitRepos) > 0 {
				repoErr := idx.CloneRepos(ctx)
				if repoErr != nil {
					idx.logger.Error("Error updating repos", "error", repoErr)
				}
			}

			count, indexErr := idx.IndexAllRepos(ctx)
			if indexErr != nil {
				idx.logger.Error("Error indexing repos", "error", indexErr)
			} else {
				idx.logger.Info("Periodic reindex complete", "functions", count)
			}

		case <-ctx.Done():
			idx.logger.Info("Indexing loop stopped")
			return
		}
	}
}
