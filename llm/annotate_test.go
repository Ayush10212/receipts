package llm_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Ayush10212/receipts/core/report"
	"github.com/Ayush10212/receipts/llm"
)

// staticProvider returns a fixed text for every call.
type staticProvider struct {
	text string
	err  error
}

func (s staticProvider) Complete(_ context.Context, _ []llm.Message, _ llm.Options) (llm.Response, error) {
	if s.err != nil {
		return llm.Response{}, s.err
	}
	return llm.Response{Text: s.text, Model: "test", Provider: "test"}, nil
}

func makeTestReport() report.Report {
	return report.Report{
		SchemaVersion: "0.1.0",
		Claims: []report.Claim{
			{
				ID:         "c1",
				Type:       "symbol",
				Subtype:    "attribute",
				Text:       "pd.DataFrame.merge",
				Verdict:    report.VerdictGrounded,
				Confidence: 1.0,
				Evidence:   []report.Evidence{{Kind: "introspection", Detail: "exists", Determinism: "deterministic"}},
			},
			{
				ID:         "c2",
				Type:       "symbol",
				Subtype:    "attribute",
				Text:       "pd.DataFrame.append",
				Verdict:    report.VerdictContradicted,
				Confidence: 1.0,
				Evidence:   []report.Evidence{{Kind: "introspection", Detail: "AttributeError", Determinism: "deterministic"}},
			},
			{
				ID:         "c3",
				Type:       "symbol",
				Subtype:    "import",
				Text:       "import nonexistent_pkg",
				Verdict:    report.VerdictUnverifiable,
				Confidence: 0.0,
				Evidence:   []report.Evidence{{Kind: "introspection", Detail: "ImportError", Determinism: "deterministic"}},
			},
		},
		Summary: report.Summary{Grounded: 1, Contradicted: 1, Unverifiable: 1},
		Policy:  report.Policy{Decision: report.DecisionFail},
	}
}

// TestAnnotate_LLMNotesOnNonGroundedOnly verifies llm-notes appear only on
// contradicted/unverifiable claims, never on grounded.
func TestAnnotate_LLMNotesOnNonGroundedOnly(t *testing.T) {
	r := makeTestReport()
	provider := staticProvider{text: "this is a helpful note"}

	llm.Annotate(context.Background(), &r, provider)

	for _, c := range r.Claims {
		hasNote := false
		for _, e := range c.Evidence {
			if e.Kind == "llm-note" {
				hasNote = true
				if e.Determinism != "subjective" {
					t.Errorf("claim %s: llm-note has determinism %q, want \"subjective\"", c.ID, e.Determinism)
				}
			}
		}
		if c.Verdict == report.VerdictGrounded && hasNote {
			t.Errorf("claim %s (grounded) must not have an llm-note", c.ID)
		}
		if (c.Verdict == report.VerdictContradicted || c.Verdict == report.VerdictUnverifiable) && !hasNote {
			t.Errorf("claim %s (%s) should have an llm-note", c.ID, c.Verdict)
		}
	}
}

// TestAnnotate_HonestyInvariant — the core credibility gate.
// Toggling the LLM provider changes only prose evidence; verdicts, confidence,
// and summary counts must be byte-identical.
func TestAnnotate_HonestyInvariant(t *testing.T) {
	// Snapshot before annotation.
	base := makeTestReport()
	verdictsBefore := extractVerdicts(base)
	summaryBefore := base.Summary
	decisionBefore := base.Policy.Decision

	// Run with LLM on.
	withLLM := makeTestReport()
	llm.Annotate(context.Background(), &withLLM, staticProvider{text: "advisory note"})

	// Run without LLM (disabled provider).
	withoutLLM := makeTestReport()
	llm.Annotate(context.Background(), &withoutLLM, staticProvider{err: llm.ErrDisabled})

	// Verdicts, summary, decision must be identical across all three.
	for _, r := range []report.Report{withLLM, withoutLLM} {
		if r.Summary != summaryBefore {
			t.Errorf("summary changed after LLM annotation: got %+v, want %+v", r.Summary, summaryBefore)
		}
		if r.Policy.Decision != decisionBefore {
			t.Errorf("policy decision changed after LLM annotation: got %q, want %q", r.Policy.Decision, decisionBefore)
		}
		for i, c := range r.Claims {
			if c.Verdict != verdictsBefore[i] {
				t.Errorf("claim %s verdict changed: got %q, want %q", c.ID, c.Verdict, verdictsBefore[i])
			}
			if c.Confidence != base.Claims[i].Confidence {
				t.Errorf("claim %s confidence changed: got %v, want %v", c.ID, c.Confidence, base.Claims[i].Confidence)
			}
		}
	}

	// withLLM should have llm-notes on non-grounded claims.
	for _, c := range withLLM.Claims {
		for _, e := range c.Evidence {
			if e.Kind == "llm-note" {
				if e.Determinism != "subjective" {
					t.Errorf("llm-note on claim %s has determinism %q; must be \"subjective\"", c.ID, e.Determinism)
				}
				if c.Verdict == report.VerdictGrounded {
					t.Errorf("llm-note must never appear on a grounded claim (claim %s)", c.ID)
				}
			}
		}
	}

	// withoutLLM (disabled) should have NO llm-notes added.
	for _, c := range withoutLLM.Claims {
		for _, e := range c.Evidence {
			if e.Kind == "llm-note" {
				t.Errorf("disabled provider still added llm-note to claim %s", c.ID)
			}
		}
	}
}

func extractVerdicts(r report.Report) []report.Verdict {
	v := make([]report.Verdict, len(r.Claims))
	for i, c := range r.Claims {
		v[i] = c.Verdict
	}
	return v
}

// TestAnnotate_ErrDisabled_NoNotesAdded verifies that a disabled provider
// leaves the report completely unchanged.
func TestAnnotate_ErrDisabled_NoNotesAdded(t *testing.T) {
	r := makeTestReport()
	claimCountBefore := len(r.Claims)

	llm.Annotate(context.Background(), &r, staticProvider{err: errors.New("some network error")})

	if len(r.Claims) != claimCountBefore {
		t.Errorf("claim count changed after failed annotation")
	}
	for _, c := range r.Claims {
		for _, e := range c.Evidence {
			if e.Kind == "llm-note" {
				t.Errorf("claim %s got llm-note despite provider error", c.ID)
			}
		}
	}
}
