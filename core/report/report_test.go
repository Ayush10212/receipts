package report_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Ayush10212/receipts/core/report"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func loadReportSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	schemaPath := filepath.Join(filepath.Dir(file), "..", "..", "protocol", "report.v0.json")

	data, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schemaDoc any
	if err := json.Unmarshal(data, &schemaDoc); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	if err := c.AddResource("report.v0.json", schemaDoc); err != nil {
		t.Fatalf("add resource: %v", err)
	}
	sch, err := c.Compile("report.v0.json")
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}
	return sch
}

func sampleReport() report.Report {
	ts, _ := time.Parse(time.RFC3339, "2026-06-26T00:00:00Z")
	return report.Report{
		SchemaVersion: "0.1.0",
		Run: report.Run{
			ID:          "run-test-001",
			Timestamp:   ts,
			ToolVersion: "0.1.0",
			InputsHash:  "sha256:abc123",
			TargetEnv: report.TargetEnv{
				Language: "python",
				Version:  "3.11.4",
				Prefix:   "/home/user/.venv",
			},
		},
		Claims: []report.Claim{
			{
				ID:         "c-001",
				Type:       "symbol",
				Subtype:    report.SubtypeAttribute,
				Text:       "pandas.DataFrame.merge",
				Locus:      report.Locus{File: "main.py", Line: 5, Col: 4, EndLine: 5, EndCol: 24},
				Verdict:    report.VerdictGrounded,
				Confidence: 1.0,
				Evidence: []report.Evidence{
					{Kind: "introspection", Detail: "pandas 2.1.0: merge exists", Determinism: report.DeterminismDeterministic},
				},
				Verifier: report.VerifierInfo{
					Name: "python-symbols", Version: "0.1.0", Determinism: report.DeterminismDeterministic,
				},
			},
		},
		Policy: report.Policy{
			Backend:      "local",
			Decision:     report.DecisionPass,
			RulesApplied: []string{"fail-on-contradicted"},
		},
		Summary: report.Summary{Grounded: 1, Contradicted: 0, Unverifiable: 0},
	}
}

func TestReport_RoundTrip(t *testing.T) {
	sch := loadReportSchema(t)
	original := sampleReport()

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Validate against schema
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("unmarshal for validation: %v", err)
	}
	if err := sch.Validate(v); err != nil {
		t.Fatalf("schema validation failed: %v", err)
	}

	// Unmarshal back
	var got report.Report
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Assert structural equality on key fields
	if got.SchemaVersion != original.SchemaVersion {
		t.Errorf("schema_version: got %q want %q", got.SchemaVersion, original.SchemaVersion)
	}
	if got.Run.ID != original.Run.ID {
		t.Errorf("run.id: got %q want %q", got.Run.ID, original.Run.ID)
	}
	if len(got.Claims) != len(original.Claims) {
		t.Fatalf("claims len: got %d want %d", len(got.Claims), len(original.Claims))
	}
	if got.Claims[0].Verdict != original.Claims[0].Verdict {
		t.Errorf("verdict: got %q want %q", got.Claims[0].Verdict, original.Claims[0].Verdict)
	}
	if got.Summary.Grounded != original.Summary.Grounded {
		t.Errorf("summary.grounded: got %d want %d", got.Summary.Grounded, original.Summary.Grounded)
	}
	if got.Policy.Decision != original.Policy.Decision {
		t.Errorf("policy.decision: got %q want %q", got.Policy.Decision, original.Policy.Decision)
	}
}
