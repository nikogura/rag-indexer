package indexer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"testing"
	"time"

	"github.com/nikogura/rag-indexer/pkg/elasticsearch"
)

func TestHasNamedReturns(t *testing.T) {
	tests := []struct {
		name     string
		funcCode string
		want     bool
	}{
		{
			name: "named returns",
			funcCode: `package test
func Foo() (result string, err error) {
	return result, err
}`,
			want: true,
		},
		{
			name: "unnamed returns",
			funcCode: `package test
func Foo() (string, error) {
	return "", nil
}`,
			want: false,
		},
		{
			name: "single named return",
			funcCode: `package test
func Foo() (result string) {
	return result
}`,
			want: true,
		},
		{
			name: "no returns",
			funcCode: `package test
func Foo() {
}`,
			want: false,
		},
		{
			name: "mixed named and unnamed",
			funcCode: `package test
func Foo() (result string, err error) {
	return result, err
}`,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, "", tt.funcCode, 0)
			if err != nil {
				t.Fatalf("Failed to parse code: %v", err)
			}

			var funcDecl *ast.FuncDecl
			ast.Inspect(node, func(n ast.Node) (shouldContinue bool) {
				if fd, ok := n.(*ast.FuncDecl); ok {
					funcDecl = fd
					shouldContinue = false
					return shouldContinue
				}
				shouldContinue = true
				return shouldContinue
			})

			if funcDecl == nil {
				t.Fatal("No function declaration found")
			}

			got := hasNamedReturns(funcDecl)
			if got != tt.want {
				t.Errorf("hasNamedReturns() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractFunctionDoc(t *testing.T) {
	funcCode := `package test

import (
	"context"
	"errors"
)

func TestFunc(ctx context.Context, input string) (result string, err error) {
	if err != nil {
		err = errors.New("empty input")
		return result, err
	}
	result = input
	return result, err
}`

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "test.go", funcCode, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	var funcDecl *ast.FuncDecl
	ast.Inspect(node, func(n ast.Node) (shouldContinue bool) {
		if fd, ok := n.(*ast.FuncDecl); ok {
			funcDecl = fd
			shouldContinue = false
			return shouldContinue
		}
		shouldContinue = true
		return shouldContinue
	})

	if funcDecl == nil {
		t.Fatal("No function declaration found")
	}

	imports := []string{"context", "errors"}
	content := []byte(funcCode)

	doc := extractFunctionDoc(funcDecl, fset, content, "testrepo", "test.go", "test", imports)

	if doc.Repo != "testrepo" {
		t.Errorf("Repo = %v, want testrepo", doc.Repo)
	}
	if doc.FilePath != "test.go" {
		t.Errorf("FilePath = %v, want test.go", doc.FilePath)
	}
	if doc.FunctionName != "TestFunc" {
		t.Errorf("FunctionName = %v, want TestFunc", doc.FunctionName)
	}
	if doc.Package != "test" {
		t.Errorf("Package = %v, want test", doc.Package)
	}
	if len(doc.Imports) != 2 {
		t.Errorf("Imports length = %v, want 2", len(doc.Imports))
	}
	if !doc.HasNamedReturns {
		t.Error("HasNamedReturns = false, want true")
	}
	if !doc.HasErrorHandling {
		t.Error("HasErrorHandling = false, want true")
	}
	if doc.Code == "" {
		t.Error("Code is empty")
	}
	if doc.IndexedAt.IsZero() {
		t.Error("IndexedAt is zero")
	}

	timeDiff := time.Since(doc.IndexedAt)
	if timeDiff > time.Second {
		t.Errorf("IndexedAt is too old: %v", timeDiff)
	}
}

func TestExtractFunctionDocNoErrorHandling(t *testing.T) {
	funcCode := `package test

func Simple(x int) (result int) {
	result = x * 2
	return result
}`

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "test.go", funcCode, 0)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	var funcDecl *ast.FuncDecl
	ast.Inspect(node, func(n ast.Node) (shouldContinue bool) {
		if fd, ok := n.(*ast.FuncDecl); ok {
			funcDecl = fd
			shouldContinue = false
			return shouldContinue
		}
		shouldContinue = true
		return shouldContinue
	})

	if funcDecl == nil {
		t.Fatal("No function declaration found")
	}

	content := []byte(funcCode)
	doc := extractFunctionDoc(funcDecl, fset, content, "testrepo", "test.go", "test", nil)

	if doc.HasErrorHandling {
		t.Error("HasErrorHandling = true, want false")
	}
	if !doc.HasNamedReturns {
		t.Error("HasNamedReturns = false, want true")
	}
}

func TestExtractFunctionDocRealFile(t *testing.T) {
	testFile := "testdata/sample.go"
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, testFile, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test file: %v", err)
	}

	expectedFuncs := map[string]struct {
		hasNamedReturns  bool
		hasErrorHandling bool
	}{
		"FunctionWithNamedReturns":   {true, false},
		"FunctionWithUnnamedReturns": {false, false},
		"FunctionWithErrorHandling":  {true, true},
		"process":                    {true, false},
		"FunctionNoErrorHandling":    {true, false},
		"FunctionNoReturns":          {false, false},
		"ComplexFunction":            {true, false},
	}

	foundFuncs := make(map[string]elasticsearch.CodeDocument)

	ast.Inspect(node, func(n ast.Node) (shouldContinue bool) {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			doc := extractFunctionDoc(funcDecl, fset, content, "testrepo", testFile, "testdata", nil)
			foundFuncs[doc.FunctionName] = doc
		}
		shouldContinue = true
		return shouldContinue
	})

	for funcName, expected := range expectedFuncs {
		doc, found := foundFuncs[funcName]
		if !found {
			t.Errorf("Function %s not found", funcName)
			continue
		}

		if doc.HasNamedReturns != expected.hasNamedReturns {
			t.Errorf("%s: HasNamedReturns = %v, want %v", funcName, doc.HasNamedReturns, expected.hasNamedReturns)
		}
		if doc.HasErrorHandling != expected.hasErrorHandling {
			t.Errorf("%s: HasErrorHandling = %v, want %v", funcName, doc.HasErrorHandling, expected.hasErrorHandling)
		}
		if doc.Code == "" {
			t.Errorf("%s: Code is empty", funcName)
		}
		if doc.FunctionName != funcName {
			t.Errorf("%s: FunctionName = %v, want %v", funcName, doc.FunctionName, funcName)
		}
	}

	if len(foundFuncs) != len(expectedFuncs) {
		t.Errorf("Found %d functions, expected %d", len(foundFuncs), len(expectedFuncs))
	}
}

func TestExtractFunctionDocCodeExtraction(t *testing.T) {
	funcCode := `package test

// TestFunc is a test function
func TestFunc() (result string) {
	result = "hello"
	return result
}`

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "test.go", funcCode, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	var funcDecl *ast.FuncDecl
	ast.Inspect(node, func(n ast.Node) (shouldContinue bool) {
		if fd, ok := n.(*ast.FuncDecl); ok {
			funcDecl = fd
			shouldContinue = false
			return shouldContinue
		}
		shouldContinue = true
		return shouldContinue
	})

	content := []byte(funcCode)
	doc := extractFunctionDoc(funcDecl, fset, content, "testrepo", "test.go", "test", nil)

	if doc.Code == "" {
		t.Fatal("Code is empty")
	}

	expectedSubstrings := []string{"func TestFunc()", "result = \"hello\"", "return result"}
	for _, substr := range expectedSubstrings {
		if !containsString(doc.Code, substr) {
			t.Errorf("Code does not contain expected substring: %q", substr)
		}
	}
}

func containsString(s string, substr string) (result bool) {
	result = len(s) >= len(substr) && findString(s, substr)
	return result
}

func findString(s string, substr string) (found bool) {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			found = true
			return found
		}
	}
	found = false
	return found
}
