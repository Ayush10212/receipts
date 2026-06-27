package llm

import (
	"context"
	"fmt"

	"github.com/Ayush10212/receipts/core/report"
)

// Annotate adds a kind:"llm-note", determinism:"subjective" evidence entry to each
// contradicted or unverifiable claim. Verdicts, confidence, and summary counts are
// computed before this runs and are never touched here.
func Annotate(ctx context.Context, r *report.Report, p Provider) {
	for i := range r.Claims {
		c := &r.Claims[i]
		if c.Verdict == report.VerdictGrounded {
			continue
		}
		note, err := generateNote(ctx, c, p)
		if err != nil || note == "" {
			continue
		}
		c.Evidence = append(c.Evidence, report.Evidence{
			Kind:        "llm-note",
			Detail:      note,
			Determinism: "subjective",
		})
	}
}

func generateNote(ctx context.Context, c *report.Claim, p Provider) (string, error) {
	existingDetail := ""
	for _, e := range c.Evidence {
		if e.Kind != "llm-note" {
			existingDetail = e.Detail
			break
		}
	}

	prompt := fmt.Sprintf(
		"A static analysis tool found this Python claim: %q\nVerdict: %s\nDetails: %s\n\n"+
			"In one sentence, explain what this means for a developer and what to do about it.",
		c.Text, c.Verdict, existingDetail,
	)

	messages := []Message{
		{Role: "system", Content: "You are a concise code analysis assistant. Respond in one sentence only."},
		{Role: "user", Content: prompt},
	}

	resp, err := p.Complete(ctx, messages, Options{MaxTokens: 120})
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}
