package elasticsearch

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCodeDocumentJSON(t *testing.T) {
	doc := CodeDocument{
		Repo:             "test-repo",
		FilePath:         "pkg/test/file.go",
		FunctionName:     "TestFunction",
		Code:             "func TestFunction() (result string, err error) { return result, err }",
		HasNamedReturns:  true,
		HasErrorHandling: true,
		Package:          "test",
		Imports:          []string{"context", "errors"},
		LintCompliant:    false,
		IndexedAt:        time.Date(2025, 10, 28, 12, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("Failed to marshal CodeDocument: %v", err)
	}

	var decoded CodeDocument
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal CodeDocument: %v", err)
	}

	if decoded.Repo != doc.Repo {
		t.Errorf("Repo = %v, want %v", decoded.Repo, doc.Repo)
	}
	if decoded.FilePath != doc.FilePath {
		t.Errorf("FilePath = %v, want %v", decoded.FilePath, doc.FilePath)
	}
	if decoded.FunctionName != doc.FunctionName {
		t.Errorf("FunctionName = %v, want %v", decoded.FunctionName, doc.FunctionName)
	}
	if decoded.Code != doc.Code {
		t.Errorf("Code = %v, want %v", decoded.Code, doc.Code)
	}
	if decoded.HasNamedReturns != doc.HasNamedReturns {
		t.Errorf("HasNamedReturns = %v, want %v", decoded.HasNamedReturns, doc.HasNamedReturns)
	}
	if decoded.HasErrorHandling != doc.HasErrorHandling {
		t.Errorf("HasErrorHandling = %v, want %v", decoded.HasErrorHandling, doc.HasErrorHandling)
	}
	if decoded.Package != doc.Package {
		t.Errorf("Package = %v, want %v", decoded.Package, doc.Package)
	}
	if decoded.LintCompliant != doc.LintCompliant {
		t.Errorf("LintCompliant = %v, want %v", decoded.LintCompliant, doc.LintCompliant)
	}

	if len(decoded.Imports) != len(doc.Imports) {
		t.Errorf("Imports length = %v, want %v", len(decoded.Imports), len(doc.Imports))
	} else {
		for i := range decoded.Imports {
			if decoded.Imports[i] != doc.Imports[i] {
				t.Errorf("Imports[%d] = %v, want %v", i, decoded.Imports[i], doc.Imports[i])
			}
		}
	}

	if !decoded.IndexedAt.Equal(doc.IndexedAt) {
		t.Errorf("IndexedAt = %v, want %v", decoded.IndexedAt, doc.IndexedAt)
	}
}

func TestSearchRequestJSON(t *testing.T) {
	req := SearchRequest{
		Query: "error handling",
		Limit: 20,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal SearchRequest: %v", err)
	}

	var decoded SearchRequest
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal SearchRequest: %v", err)
	}

	if decoded.Query != req.Query {
		t.Errorf("Query = %v, want %v", decoded.Query, req.Query)
	}
	if decoded.Limit != req.Limit {
		t.Errorf("Limit = %v, want %v", decoded.Limit, req.Limit)
	}
}

func TestSearchResponseJSON(t *testing.T) {
	jsonData := `{
		"hits": {
			"hits": [
				{
					"_source": {
						"repo": "test-repo",
						"file_path": "test.go",
						"function_name": "TestFunc",
						"code": "func TestFunc() {}",
						"has_namedreturns": true,
						"has_error_handling": false,
						"package": "test",
						"imports": ["context"],
						"lint_compliant": false,
						"indexed_at": "2025-10-28T12:00:00Z"
					}
				}
			]
		}
	}`

	var resp SearchResponse
	err := json.Unmarshal([]byte(jsonData), &resp)
	if err != nil {
		t.Fatalf("Failed to unmarshal SearchResponse: %v", err)
	}

	if len(resp.Hits.Hits) != 1 {
		t.Fatalf("Expected 1 hit, got %d", len(resp.Hits.Hits))
	}

	doc := resp.Hits.Hits[0].Source
	if doc.Repo != "test-repo" {
		t.Errorf("Repo = %v, want test-repo", doc.Repo)
	}
	if doc.FunctionName != "TestFunc" {
		t.Errorf("FunctionName = %v, want TestFunc", doc.FunctionName)
	}
	if !doc.HasNamedReturns {
		t.Error("HasNamedReturns = false, want true")
	}
	if doc.HasErrorHandling {
		t.Error("HasErrorHandling = true, want false")
	}
}

func TestCodeDocumentEmptyImports(t *testing.T) {
	doc := CodeDocument{
		Repo:         "test-repo",
		FilePath:     "test.go",
		FunctionName: "TestFunc",
		Imports:      []string{},
	}

	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded CodeDocument
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Imports == nil {
		t.Error("Imports is nil, should be empty slice")
	}
}
