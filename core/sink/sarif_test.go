package sink_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Ayush10212/receipts/core/report"
	"github.com/Ayush10212/receipts/core/sink"
)

func makeReport(verdict report.Verdict) report.Report {
	ts, _ := time.Parse(time.RFC3339, "2026-06-26T00:00:00Z")
	return report.Report{
		SchemaVersion: "0.1.0",
		Run: report.Run{
			ID: "run-sarif-test", Timestamp: ts,
			ToolVersion: "0.1.0", InputsHash: "abc",
			TargetEnv: report.TargetEnv{Language: "python", Version: "3.11.4", Prefix: "/venv"},
		},
		Claims: []report.Claim{
			{
				ID: "c-1", Type: "symbol", Subtype: report.SubtypeAttribute,
				Text:       "pandas.DataFrame.append",
				Locus:      report.Locus{File: "script.py", Line: 12, Col: 0, EndLine: 12, EndCol: 22},
				Verdict:    verdict,
				Confidence: 1.0,
				Evidence:   []report.Evidence{{Kind: "introspection", Detail: "removed in pandas 2.0", Determinism: report.DeterminismDeterministic}},
				Verifier:   report.VerifierInfo{Name: "python-symbols", Version: "0.1.0", Determinism: report.DeterminismDeterministic},
			},
		},
		Policy:  report.Policy{Backend: "local", Decision: report.DecisionFail, RulesApplied: []string{"fail-on-contradicted"}},
		Summary: report.Summary{Contradicted: 1},
	}
}

func TestSARIFSink_ContradictedIsError(t *testing.T) {
	var buf bytes.Buffer
	s := sink.SARIFSink{W: &buf}
	r := makeReport(report.VerdictContradicted)

	if err := s.Emit(context.Background(), report.ExecutionContext{}, r); err != nil {
		t.Fatalf("Emit: %v", err)
	}

	var sarif map[string]any
	if err := json.Unmarshal(buf.Bytes(), &sarif); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if sarif["version"] != "2.1.0" {
		t.Errorf("expected SARIF version 2.1.0, got %v", sarif["version"])
	}
	runs := sarif["runs"].([]any)
	results := runs[0].(map[string]any)["results"].([]any)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	res := results[0].(map[string]any)
	if res["level"] != "error" {
		t.Errorf("contradicted should be error level, got %v", res["level"])
	}
}

func TestSARIFSink_UnverifiableIsWarning(t *testing.T) {
	var buf bytes.Buffer
	s := sink.SARIFSink{W: &buf}
	r := makeReport(report.VerdictUnverifiable)
	r.Policy.Decision = report.DecisionWarn
	r.Summary = report.Summary{Unverifiable: 1}

	if err := s.Emit(context.Background(), report.ExecutionContext{}, r); err != nil {
		t.Fatalf("Emit: %v", err)
	}
	var sarif map[string]any
	json.Unmarshal(buf.Bytes(), &sarif)
	runs := sarif["runs"].([]any)
	results := runs[0].(map[string]any)["results"].([]any)
	res := results[0].(map[string]any)
	if res["level"] != "warning" {
		t.Errorf("unverifiable should be warning level, got %v", res["level"])
	}
}

func TestSARIFSink_GroundedNotEmitted(t *testing.T) {
	var buf bytes.Buffer
	s := sink.SARIFSink{W: &buf}
	r := makeReport(report.VerdictGrounded)
	r.Policy.Decision = report.DecisionPass
	r.Summary = report.Summary{Grounded: 1}

	if err := s.Emit(context.Background(), report.ExecutionContext{}, r); err != nil {
		t.Fatalf("Emit: %v", err)
	}
	var sarif map[string]any
	json.Unmarshal(buf.Bytes(), &sarif)
	runs := sarif["runs"].([]any)
	results := runs[0].(map[string]any)["results"]
	if results != nil && len(results.([]any)) != 0 {
		t.Error("grounded claims should not appear in SARIF results")
	}
}

func TestSARIFSink_PhysicalLocation(t *testing.T) {
	var buf bytes.Buffer
	s := sink.SARIFSink{W: &buf}
	r := makeReport(report.VerdictContradicted)

	s.Emit(context.Background(), report.ExecutionContext{}, r)
	var sarif map[string]any
	json.Unmarshal(buf.Bytes(), &sarif)

	runs := sarif["runs"].([]any)
	results := runs[0].(map[string]any)["results"].([]any)
	locs := results[0].(map[string]any)["locations"].([]any)
	phys := locs[0].(map[string]any)["physicalLocation"].(map[string]any)
	uri := phys["artifactLocation"].(map[string]any)["uri"]
	if uri != "script.py" {
		t.Errorf("expected uri=script.py got %v", uri)
	}
	region := phys["region"].(map[string]any)
	if region["startLine"].(float64) != 12 {
		t.Errorf("expected startLine=12 got %v", region["startLine"])
	}
}
