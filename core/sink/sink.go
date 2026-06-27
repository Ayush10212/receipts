package sink

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/Ayush10212/receipts/core/report"
)

type ReportSink interface {
	Emit(ctx context.Context, ectx report.ExecutionContext, r report.Report) error
}

// PrettySink writes a human-readable summary to w.
type PrettySink struct{ W io.Writer }

func (s PrettySink) Emit(_ context.Context, _ report.ExecutionContext, r report.Report) error {
	fmt.Fprintf(s.W, "Receipts %s  run=%s\n", r.Run.ToolVersion, r.Run.ID)
	fmt.Fprintf(s.W, "Target env: %s %s (%s)\n", r.Run.TargetEnv.Language, r.Run.TargetEnv.Version, r.Run.TargetEnv.Prefix)
	fmt.Fprintf(s.W, "Summary: grounded=%d  contradicted=%d  unverifiable=%d\n",
		r.Summary.Grounded, r.Summary.Contradicted, r.Summary.Unverifiable)
	fmt.Fprintf(s.W, "Decision: %s\n", r.Policy.Decision)
	for _, c := range r.Claims {
		if c.Verdict != report.VerdictGrounded {
			fmt.Fprintf(s.W, "  [%s] %s  %s:%d\n", c.Verdict, c.Text, c.Locus.File, c.Locus.Line)
			for _, e := range c.Evidence {
				fmt.Fprintf(s.W, "    evidence: %s\n", e.Detail)
			}
		}
	}
	return nil
}

// PlainSink writes a plain-English review for non-developers to w. It delegates
// wording to report.Explain so the CLI and the MCP server stay in sync.
type PlainSink struct{ W io.Writer }

func (s PlainSink) Emit(_ context.Context, _ report.ExecutionContext, r report.Report) error {
	_, err := io.WriteString(s.W, report.Explain(r))
	return err
}

// JSONSink writes the Report as compact JSON to w.
type JSONSink struct{ W io.Writer }

func (s JSONSink) Emit(_ context.Context, _ report.ExecutionContext, r report.Report) error {
	enc := json.NewEncoder(s.W)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}
