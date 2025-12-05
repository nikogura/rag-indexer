package indexer

import (
	"context"
	"go/ast"
	"go/token"

	"github.com/nikogura/rag-indexer/pkg/elasticsearch"
	"github.com/nikogura/rag-indexer/pkg/logging"
)

// astVisitor visits AST nodes and indexes functions.
type astVisitor struct {
	ctx       context.Context
	es        *elasticsearch.Client
	logger    logging.Logger
	fset      *token.FileSet
	content   []byte
	repo      string
	filePath  string
	pkgName   string
	imports   []string
	funcCount int
}

// Visit implements ast.Visitor interface for function indexing.
func (v *astVisitor) Visit(n ast.Node) (shouldContinue bool) {
	funcDecl, ok := n.(*ast.FuncDecl)
	if !ok {
		shouldContinue = true
		return shouldContinue
	}

	doc := extractFunctionDoc(funcDecl, v.fset, v.content, v.repo, v.filePath, v.pkgName, v.imports)

	indexErr := v.es.IndexDocument(v.ctx, doc)
	if indexErr != nil {
		v.logger.Warn("Failed to index function", "function", doc.FunctionName, "error", indexErr)
	} else {
		v.funcCount++
	}

	shouldContinue = true
	return shouldContinue
}
