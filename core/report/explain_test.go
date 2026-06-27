package report

import (
	"strings"
	"testing"
)

func failingReport() Report {
	return Report{
		Run: Run{TargetEnv: TargetEnv{Language: "python"}},
		Claims: []Claim{{
			Text:    "pandas.DataFrame.append",
			Subtype: SubtypeAttribute,
			Verdict: VerdictContradicted,
			Locus:   Locus{File: "etl.py", Line: 6},
			Evidence: []Evidence{{
				Kind:   "introspection",
				Detail: "'pandas.DataFrame.append' not found",
			}},
		}},
		Policy:  Policy{Decision: DecisionFail},
		Summary: Summary{Grounded: 4, Contradicted: 1},
	}
}

func TestExplain_FailIsLoudAndPlain(t *testing.T) {
	out := Explain(failingReport())

	for _, want := range []string{
		"plain-English review",
		"etl.py",
		"WRONG",
		"DO NOT use this code yet",
		"AttributeError",    // the everyday meaning
		"technical detail:", // raw evidence preserved
	} {
		if !strings.Contains(out, want) {
			t.Errorf("Explain() missing %q\n--- got ---\n%s", want, out)
		}
	}
}

func TestExplain_PassIsReassuring(t *testing.T) {
	r := Report{
		Run:     Run{TargetEnv: TargetEnv{Language: "python"}},
		Claims:  []Claim{{Text: "pandas.concat", Verdict: VerdictGrounded}},
		Policy:  Policy{Decision: DecisionPass},
		Summary: Summary{Grounded: 1},
	}
	out := Explain(r)
	if !strings.Contains(out, "Looks good") {
		t.Errorf("Explain() pass case should reassure, got:\n%s", out)
	}
	// A clean report shows no per-claim details section.
	if strings.Contains(out, "Details:") {
		t.Errorf("Explain() should omit Details when nothing is wrong, got:\n%s", out)
	}
}

func TestExplain_UnverifiableIsNotAlarming(t *testing.T) {
	r := Report{
		Run: Run{TargetEnv: TargetEnv{Language: "python"}},
		Claims: []Claim{{
			Text:    "scipy.special.expit",
			Subtype: SubtypeAttribute,
			Verdict: VerdictUnverifiable,
			Locus:   Locus{File: "m.py", Line: 2},
		}},
		Policy:  Policy{Decision: DecisionWarn},
		Summary: Summary{Unverifiable: 1},
	}
	out := Explain(r)
	if strings.Contains(out, "[WRONG]") {
		t.Errorf("unverifiable must not be labelled [WRONG], got:\n%s", out)
	}
	if !strings.Contains(out, "wasn't checked") {
		t.Errorf("unverifiable should read as unchecked, not wrong, got:\n%s", out)
	}
}
