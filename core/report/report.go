package report

import "time"

type Verdict string

const (
	VerdictGrounded      Verdict = "grounded"
	VerdictContradicted  Verdict = "contradicted"
	VerdictUnverifiable  Verdict = "unverifiable"
)

type Decision string

const (
	DecisionPass Decision = "pass"
	DecisionWarn Decision = "warn"
	DecisionFail Decision = "fail"
)

type Determinism string

const (
	DeterminismDeterministic Determinism = "deterministic"
	DeterminismSubjective    Determinism = "subjective"
)

type Subtype string

const (
	SubtypeImport    Subtype = "import"
	SubtypeAttribute Subtype = "attribute"
	SubtypeKwarg     Subtype = "kwarg"
)

type TargetEnv struct {
	Language string `json:"language"`
	Version  string `json:"version"`
	Prefix   string `json:"prefix"`
}

type Run struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	ToolVersion string    `json:"tool_version"`
	InputsHash  string    `json:"inputs_hash"`
	TargetEnv   TargetEnv `json:"target_env"`
}

type Locus struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Col     int    `json:"col"`
	EndLine int    `json:"end_line,omitempty"`
	EndCol  int    `json:"end_col,omitempty"`
}

type Evidence struct {
	Kind        string      `json:"kind"`
	Detail      string      `json:"detail"`
	Determinism Determinism `json:"determinism,omitempty"`
}

type VerifierInfo struct {
	Name        string      `json:"name"`
	Version     string      `json:"version"`
	Determinism Determinism `json:"determinism"`
}

type Claim struct {
	ID         string       `json:"id"`
	Type       string       `json:"type"`
	Subtype    Subtype      `json:"subtype"`
	Text       string       `json:"text"`
	Locus      Locus        `json:"locus"`
	Verdict    Verdict      `json:"verdict"`
	Confidence float64      `json:"confidence"`
	Evidence   []Evidence   `json:"evidence"`
	Verifier   VerifierInfo `json:"verifier"`
}

type Policy struct {
	Backend      string   `json:"backend"`
	Decision     Decision `json:"decision"`
	RulesApplied []string `json:"rules_applied"`
}

type Summary struct {
	Grounded     int `json:"grounded"`
	Contradicted int `json:"contradicted"`
	Unverifiable int `json:"unverifiable"`
}

type Report struct {
	SchemaVersion string  `json:"schema_version"`
	Run           Run     `json:"run"`
	Claims        []Claim `json:"claims"`
	Policy        Policy  `json:"policy"`
	Summary       Summary `json:"summary"`
}

// ExecutionContext is threaded through all pipeline calls; fields (identity,
// tenant, trace IDs) will be added here without requiring re-plumbing of signatures.
type ExecutionContext struct{}
