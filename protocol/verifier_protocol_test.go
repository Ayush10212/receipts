package protocol_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func loadVerifierProtocol(t *testing.T) *jsonschema.Schema {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Dir(file)

	data, err := os.ReadFile(filepath.Join(dir, "verifier-protocol.v0.json"))
	if err != nil {
		t.Fatalf("read verifier protocol: %v", err)
	}
	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal verifier protocol: %v", err)
	}

	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	if err := c.AddResource("verifier-protocol.v0.json", doc); err != nil {
		t.Fatalf("add resource: %v", err)
	}
	// Compile the initialize_result sub-schema as a representative check.
	sch, err := c.Compile("verifier-protocol.v0.json#/$defs/initialize_result")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	return sch
}

func TestVerifierProtocol_InitializeResult(t *testing.T) {
	sch := loadVerifierProtocol(t)

	valid := map[string]any{
		"name": "python-symbols", "version": "0.1.0",
		"capabilities": []any{"python"}, "determinism": "deterministic",
	}
	if err := sch.Validate(valid); err != nil {
		t.Fatalf("valid initialize result rejected: %v", err)
	}

	invalid := map[string]any{
		"name": "", "version": "0.1.0",
		"capabilities": []any{}, "determinism": "nondeterministic",
	}
	if err := sch.Validate(invalid); err == nil {
		t.Fatal("invalid initialize result accepted")
	}
}

func TestVerifierProtocol_AnalyzeResult(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Dir(file)

	data, err := os.ReadFile(filepath.Join(dir, "verifier-protocol.v0.json"))
	if err != nil {
		t.Fatalf("read protocol: %v", err)
	}
	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	if err := c.AddResource("verifier-protocol.v0.json", doc); err != nil {
		t.Fatalf("add resource: %v", err)
	}
	sch, err := c.Compile("verifier-protocol.v0.json#/$defs/analyze_result")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	valid := map[string]any{
		"claims": []any{
			map[string]any{
				"id": "c-1", "type": "symbol", "subtype": "attribute",
				"text": "pandas.DataFrame.append",
				"locus": map[string]any{"file": "x.py", "line": 2, "col": 0},
				"verdict": "contradicted", "confidence": 1.0,
				"evidence": []any{
					map[string]any{"kind": "introspection", "detail": "removed in 2.0", "determinism": "deterministic"},
				},
			},
		},
	}
	if err := sch.Validate(valid); err != nil {
		t.Fatalf("valid analyze result rejected: %v", err)
	}
}
