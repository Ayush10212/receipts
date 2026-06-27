package policy_test

import (
	"context"
	"testing"

	"github.com/Ayush10212/receipts/core/policy"
	"github.com/Ayush10212/receipts/core/report"
)

func TestLocalBackend_FailOnContradicted(t *testing.T) {
	b := policy.LocalBackend{}
	ctx := context.Background()
	ectx := report.ExecutionContext{}

	tests := []struct {
		name     string
		summary  report.Summary
		wantDec  report.Decision
	}{
		{"all grounded", report.Summary{Grounded: 3}, report.DecisionPass},
		{"one contradicted", report.Summary{Contradicted: 1}, report.DecisionFail},
		{"multiple contradicted", report.Summary{Contradicted: 2, Grounded: 1}, report.DecisionFail},
		{"only unverifiable", report.Summary{Unverifiable: 1}, report.DecisionWarn},
		{"contradicted beats unverifiable", report.Summary{Contradicted: 1, Unverifiable: 1}, report.DecisionFail},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := report.Report{Summary: tt.summary}
			got, _, err := b.Evaluate(ctx, ectx, r)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantDec {
				t.Errorf("got %q want %q", got, tt.wantDec)
			}
		})
	}
}
