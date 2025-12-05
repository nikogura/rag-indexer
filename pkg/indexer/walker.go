package indexer

import (
	"context"
	"os"
	"path/filepath"

	"github.com/nikogura/rag-indexer/pkg/elasticsearch"
	"github.com/nikogura/rag-indexer/pkg/logging"
	"github.com/nikogura/rag-indexer/pkg/metrics"
)

// fileWalker handles walking a repository tree and indexing Go files.
type fileWalker struct {
	ctx        context.Context
	es         *elasticsearch.Client
	repoName   string
	metrics    *metrics.Metrics
	logger     logging.Logger
	totalCount int
}

// walk processes a single file or directory in the tree.
func (fw *fileWalker) walk(path string, info os.FileInfo, pathErr error) (procErr error) {
	if pathErr != nil {
		procErr = pathErr
		return procErr
	}

	if info.IsDir() && (info.Name() == "vendor" || info.Name() == ".git") {
		procErr = filepath.SkipDir
		return procErr
	}

	if filepath.Ext(path) != ".go" {
		return procErr
	}

	fileCount, indexErr := indexFile(fw.ctx, fw.es, fw.logger, fw.repoName, path)
	if indexErr != nil {
		fw.logger.Warn("Failed to index file", "file", path, "error", indexErr)
		fw.metrics.ParseErrors.WithLabelValues(fw.repoName, path).Inc()
		return procErr
	}

	fw.totalCount += fileCount
	return procErr
}
