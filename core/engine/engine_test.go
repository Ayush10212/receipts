package engine_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Ayush10212/receipts/core/config"
	"github.com/Ayush10212/receipts/core/engine"
	"github.com/Ayush10212/receipts/core/policy"
	"github.com/Ayush10212/receipts/core/report"
	"github.com/Ayush10212/receipts/core/sink"
	"github.com/Ayush10212/receipts/core/verifier"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// fakeVerifierScript responds with one contradicted claim for any file.
const fakeVerifierScript = `
import sys, json
for line in sys.stdin:
    req = json.loads(line)
    m = req["method"]
    id_ = req["id"]
    if m == "initialize":
        print(json.dumps({"jsonrpc":"2.0","id":id_,"result":{"name":"fake","version":"0.1.0","capabilities":["python"],"determinism":"deterministic"}}), flush=True)
    elif m == "analyze":
        path = req["params"]["artifact"]["path"]
        print(json.dumps({"jsonrpc":"2.0","id":id_,"result":{"claims":[{
            "id":"c-1","type":"symbol","subtype":"attribute",
            "text":"pandas.DataFrame.append",
            "locus":{"file":path,"line":5,"col":0},
            "verdict":"contradicted","confidence":1.0,
            "evidence":[{"kind":"introspection","detail":"removed in pandas 2.0","determinism":"deterministic"}]
        }]}}), flush=True)
    elif m == "shutdown":
        print(json.dumps({"jsonrpc":"2.0","id":id_,"result":{}}), flush=True)
        break
`

func fakeClientFactory(script string) verifier.ClientFactory {
	return func(ctx context.Context, _ report.ExecutionContext, _ string) (*verifier.Client, error) {
		return verifier.NewClient(ctx, 5*time.Second, "python", "-c", script)
	}
}

func loadSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	schemaPath := filepath.Join(filepath.Dir(file), "..", "..", "protocol", "report.v0.json")
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var doc any
	json.Unmarshal(data, &doc)
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	c.AddResource("report.v0.json", doc)
	sch, err := c.Compile("report.v0.json")
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}
	return sch
}

func TestEngine_EndToEnd_SchemaValidAndSummaryCorrect(t *testing.T) {
	sch := loadSchema(t)
	var buf bytes.Buffer

	backends := engine.Backends{
		Policy:    policy.LocalBackend{},
		Sink:      sink.JSONSink{W: &buf},
		NewClient: fakeClientFactory(fakeVerifierScript),
	}

	artifacts := []engine.Artifact{
		{Path: "main.py", Content: []byte("import pandas\npandas.DataFrame.append()"), Language: "python"},
	}

	r, err := engine.Run(context.Background(), report.ExecutionContext{}, artifacts, config.Config{}, backends)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Summary counts must be correct.
	if r.Summary.Contradicted != 1 {
		t.Errorf("contradicted: got %d want 1", r.Summary.Contradicted)
	}
	if r.Summary.Grounded != 0 {
		t.Errorf("grounded: got %d want 0", r.Summary.Grounded)
	}

	// Policy must be fail (1 contradicted claim).
	if r.Policy.Decision != report.DecisionFail {
		t.Errorf("decision: got %q want fail", r.Policy.Decision)
	}

	// Output must be schema-valid JSON.
	var v any
	if err := json.Unmarshal(buf.Bytes(), &v); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if err := sch.Validate(v); err != nil {
		t.Fatalf("output failed schema validation: %v", err)
	}
}

func TestEngine_InputsHashIsDeterministic(t *testing.T) {
	backends := engine.Backends{
		Policy:    policy.LocalBackend{},
		NewClient: fakeClientFactory(fakeVerifierScript),
	}
	artifacts := []engine.Artifact{
		{Path: "a.py", Content: []byte("x=1"), Language: "python"},
		{Path: "b.py", Content: []byte("y=2"), Language: "python"},
	}

	r1, _ := engine.Run(context.Background(), report.ExecutionContext{}, artifacts, config.Config{}, backends)
	// Re-run with same inputs in reversed order — hash should be identical (sorted).
	artifacts2 := []engine.Artifact{artifacts[1], artifacts[0]}
	r2, _ := engine.Run(context.Background(), report.ExecutionContext{}, artifacts2, config.Config{}, backends)

	if r1.Run.InputsHash != r2.Run.InputsHash {
		t.Errorf("inputs hash not deterministic:\n  %s\n  %s", r1.Run.InputsHash, r2.Run.InputsHash)
	}
}

func TestEngine_VerifierStartFailureDegrades(t *testing.T) {
	backends := engine.Backends{
		Policy: policy.LocalBackend{},
		NewClient: func(ctx context.Context, _ report.ExecutionContext, _ string) (*verifier.Client, error) {
			// Try to spawn a non-existent binary — will fail to start.
			return verifier.NewClient(ctx, 2*time.Second, "nonexistent-binary-xyz")
		},
	}
	artifacts := []engine.Artifact{
		{Path: "fail.py", Content: []byte("import pandas"), Language: "python"},
	}

	r, err := engine.Run(context.Background(), report.ExecutionContext{}, artifacts, config.Config{}, backends)
	if err != nil {
		t.Fatalf("Run should not error on verifier start failure: %v", err)
	}
	// Must degrade: 1 unverifiable claim, decision warn (no contradicted).
	if r.Summary.Unverifiable != 1 {
		t.Errorf("unverifiable: got %d want 1", r.Summary.Unverifiable)
	}
	if r.Policy.Decision == report.DecisionFail {
		t.Error("verifier start failure should not produce fail decision")
	}
}
