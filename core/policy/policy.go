package policy

import (
	"context"

	"github.com/Ayush10212/receipts/core/report"
)

type Backend interface {
	Evaluate(ctx context.Context, ectx report.ExecutionContext, r report.Report) (report.Decision, []string, error)
}

// NoneBackend always passes regardless of verdicts (--fail-on none).
type NoneBackend struct{}

func (NoneBackend) Evaluate(_ context.Context, _ report.ExecutionContext, _ report.Report) (report.Decision, []string, error) {
	return report.DecisionPass, []string{"fail-on-none"}, nil
}

// LocalBackend applies fail-on-contradicted: returns fail iff >=1 contradicted claim.
type LocalBackend struct{}

func (LocalBackend) Evaluate(_ context.Context, _ report.ExecutionContext, r report.Report) (report.Decision, []string, error) {
	rules := []string{"fail-on-contradicted"}
	if r.Summary.Contradicted > 0 {
		return report.DecisionFail, rules, nil
	}
	if r.Summary.Unverifiable > 0 {
		return report.DecisionWarn, rules, nil
	}
	return report.DecisionPass, rules, nil
}
