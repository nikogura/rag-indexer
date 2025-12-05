package indexer

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"time"

	"github.com/nikogura/rag-indexer/pkg/elasticsearch"
	"github.com/nikogura/rag-indexer/pkg/logging"
)

// indexFile parses a Go file and indexes all functions found within it.
func indexFile(ctx context.Context, es *elasticsearch.Client, logger logging.Logger, repo string, filePath string) (funcCount int, parseErr error) {
	fset := token.NewFileSet()

	var node *ast.File
	node, parseErr = parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if parseErr != nil {
		parseErr = fmt.Errorf("failed to parse file: %w", parseErr)
		return funcCount, parseErr
	}

	pkgName := node.Name.Name
	var imports []string
	for _, imp := range node.Imports {
		imports = append(imports, strings.Trim(imp.Path.Value, `"`))
	}

	var content []byte
	content, parseErr = os.ReadFile(filePath)
	if parseErr != nil {
		parseErr = fmt.Errorf("failed to read file: %w", parseErr)
		return funcCount, parseErr
	}

	visitor := &astVisitor{
		ctx:      ctx,
		es:       es,
		logger:   logger,
		fset:     fset,
		content:  content,
		repo:     repo,
		filePath: filePath,
		pkgName:  pkgName,
		imports:  imports,
	}

	ast.Inspect(node, visitor.Visit)
	funcCount = visitor.funcCount

	return funcCount, parseErr
}

// extractFunctionDoc extracts metadata and code from a function declaration.
func extractFunctionDoc(
	funcDecl *ast.FuncDecl,
	fset *token.FileSet,
	content []byte,
	repo string,
	filePath string,
	pkgName string,
	imports []string,
) (doc elasticsearch.CodeDocument) {
	doc = elasticsearch.CodeDocument{
		Repo:         repo,
		FilePath:     filePath,
		FunctionName: funcDecl.Name.Name,
		Package:      pkgName,
		Imports:      imports,
		IndexedAt:    time.Now(),
	}

	start := fset.Position(funcDecl.Pos()).Offset
	end := fset.Position(funcDecl.End()).Offset
	doc.Code = string(content[start:end])

	doc.HasNamedReturns = hasNamedReturns(funcDecl)
	doc.HasErrorHandling = strings.Contains(doc.Code, "if err != nil")
	doc.LintCompliant = false

	return doc
}

// hasNamedReturns checks if a function has named return values.
func hasNamedReturns(funcDecl *ast.FuncDecl) (named bool) {
	if funcDecl.Type.Results == nil {
		named = false
		return named
	}

	for _, field := range funcDecl.Type.Results.List {
		if len(field.Names) > 0 {
			named = true
			return named
		}
	}

	return named
}
