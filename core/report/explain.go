package report

import (
	"fmt"
	"strings"
)

// Explain renders a Report as a plain-English review that a non-developer can
// understand. It invents nothing: it only restates, in everyday language, the
// deterministic results already computed. Verdicts, counts, and the decision
// are read straight from the Report — this function never changes them.
func Explain(r Report) string {
	var b strings.Builder

	file := "your code"
	if len(r.Claims) > 0 {
		file = r.Claims[0].Locus.File
	}

	fmt.Fprintf(&b, "Receipts — plain-English review\n")
	fmt.Fprintf(&b, "%s\n\n", strings.Repeat("=", 32))
	fmt.Fprintf(&b, "File checked:    %s\n", file)
	fmt.Fprintf(&b, "Checked against: your installed %s packages\n\n", envName(r))

	fmt.Fprintf(&b, "What I found:\n")
	fmt.Fprintf(&b, "  OK     %2d  - confirmed real and correct\n", r.Summary.Grounded)
	fmt.Fprintf(&b, "  WRONG  %2d  - uses something that does NOT exist\n", r.Summary.Contradicted)
	fmt.Fprintf(&b, "  ?      %2d  - couldn't be checked (not necessarily wrong)\n\n", r.Summary.Unverifiable)

	if r.Summary.Contradicted > 0 || r.Summary.Unverifiable > 0 {
		fmt.Fprintf(&b, "Details:\n\n")
		for _, c := range r.Claims {
			if c.Verdict == VerdictGrounded {
				continue
			}
			explainClaim(&b, c)
		}
	}

	fmt.Fprintf(&b, "Bottom line: %s\n", decisionAdvice(r))
	return b.String()
}

func envName(r Report) string {
	if lang := r.Run.TargetEnv.Language; lang != "" {
		return lang
	}
	return "Python"
}

func explainClaim(b *strings.Builder, c Claim) {
	label := "WRONG"
	if c.Verdict == VerdictUnverifiable {
		label = "?    "
	}
	fmt.Fprintf(b, "  [%s] %s   (line %d)\n", label, c.Text, c.Locus.Line)
	fmt.Fprintf(b, "         %s\n", plainMeaning(c))

	for _, e := range c.Evidence {
		switch e.Kind {
		case "llm-note":
			fmt.Fprintf(b, "         note: %s\n", e.Detail)
		default:
			fmt.Fprintf(b, "         technical detail: %s\n", e.Detail)
		}
	}
	fmt.Fprintf(b, "\n")
}

// plainMeaning turns a single claim into one everyday sentence. It branches only
// on the verdict and subtype the verifier already decided — it never re-judges.
func plainMeaning(c Claim) string {
	switch c.Verdict {
	case VerdictContradicted:
		if c.Subtype == SubtypeKwarg {
			return "This passes an option the function won't accept (or the function itself doesn't exist). Running it would crash. See the technical detail below for which."
		}
		return "This uses a name that isn't in the installed package - it was likely removed or renamed. Running it would crash with an AttributeError."
	case VerdictUnverifiable:
		return "I couldn't confirm this one - the package may not be installed, or it's built in C and can't be inspected. It might be fine; it just wasn't checked."
	default:
		return "Confirmed present in the installed package."
	}
}

func decisionAdvice(r Report) string {
	switch r.Policy.Decision {
	case DecisionFail:
		return "DO NOT use this code yet. It calls something that does not exist and will crash. Fix the WRONG items above first."
	case DecisionWarn:
		return "Mostly OK, but some items couldn't be checked. Skim the '?' items before trusting it."
	default:
		return "Looks good - everything that could be checked is real and correct."
	}
}
