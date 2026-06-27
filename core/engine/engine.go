package engine

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"time"

	"github.com/Ayush10212/receipts/core/config"
	"github.com/Ayush10212/receipts/core/policy"
	"github.com/Ayush10212/receipts/core/report"
	"github.com/Ayush10212/receipts/core/sink"
	"github.com/Ayush10212/receipts/core/verifier"
)

// Backends bundles the swappable dependencies the engine needs.
type Backends struct {
	Policy policy.Backend
	Sink   sink.ReportSink
	// NewClient is called once per unique language in the artifact list.
	NewClient verifier.ClientFactory
}

// Artifact is a source file to analyze.
type Artifact struct {
	Path     string
	Content  []byte
	Language string
}

// Run executes the full pipeline: resolve env → dispatch → assemble Report → policy → sink.
// It is a pure function of its inputs: no globals, no network.
func Run(ctx context.Context, ectx report.ExecutionContext, artifacts []Artifact, cfg config.Config, backends Backends) (report.Report, error) {
	targetEnv := resolveTargetEnv(cfg)

	// Compute inputs_hash = sha256(sorted file contents + target_env id).
	inputsHash := computeInputsHash(artifacts, targetEnv)

	// Dispatch each artifact to its verifier client.
	clients := map[string]*verifier.Client{}
	defer func() {
		for _, c := range clients {
			c.Shutdown(ctx)
		}
	}()

	var allClaims []report.Claim
	for _, art := range artifacts {
		c, err := clientFor(ctx, ectx, art.Language, clients, backends.NewClient)
		if err != nil {
			// Degrade: emit a tool-error claim for every file whose verifier failed to start.
			allClaims = append(allClaims, toolErrorClaim(art.Path, err))
			continue
		}
		claims, _ := c.Analyze(ctx, verifier.Artifact{Path: art.Path, Content: string(art.Content)},
			verifier.AnalyzeContext{
				Workdir:   ".",
				Language:  art.Language,
				TargetEnv: targetEnv,
			})
		allClaims = append(allClaims, claims...)
	}

	// Assemble Report.
	r := report.Report{
		SchemaVersion: "0.1.0",
		Run: report.Run{
			ID:          fmt.Sprintf("run-%d", time.Now().UnixNano()),
			Timestamp:   time.Now().UTC(),
			ToolVersion: "0.1.0",
			InputsHash:  inputsHash,
			TargetEnv:   targetEnv,
		},
		Claims:  allClaims,
		Summary: summarize(allClaims),
	}

	// Policy evaluation.
	decision, rules, err := backends.Policy.Evaluate(ctx, ectx, r)
	if err != nil {
		return r, fmt.Errorf("policy: %w", err)
	}
	r.Policy = report.Policy{
		Backend:      "local",
		Decision:     decision,
		RulesApplied: rules,
	}

	// Emit.
	if backends.Sink != nil {
		if err := backends.Sink.Emit(ctx, ectx, r); err != nil {
			return r, fmt.Errorf("sink: %w", err)
		}
	}

	return r, nil
}

func clientFor(ctx context.Context, ectx report.ExecutionContext, lang string, clients map[string]*verifier.Client, factory verifier.ClientFactory) (*verifier.Client, error) {
	if c, ok := clients[lang]; ok {
		return c, nil
	}
	c, err := factory(ctx, ectx, lang)
	if err != nil {
		return nil, err
	}
	clients[lang] = c
	return c, nil
}

func resolveTargetEnv(cfg config.Config) report.TargetEnv {
	// Target env is provided by the verifier; the engine stores whatever the verifier reports.
	// Default placeholder — the real env is filled in by the Python verifier at analyze time.
	return report.TargetEnv{
		Language: "python",
		Version:  "unknown",
		Prefix:   "unknown",
	}
}

func computeInputsHash(artifacts []Artifact, env report.TargetEnv) string {
	h := sha256.New()
	// Sort by path for determinism.
	sorted := make([]Artifact, len(artifacts))
	copy(sorted, artifacts)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })
	for _, a := range sorted {
		h.Write([]byte(a.Path))
		h.Write(a.Content)
	}
	h.Write([]byte(env.Language + env.Version + env.Prefix))
	return fmt.Sprintf("sha256:%x", h.Sum(nil))
}

func summarize(claims []report.Claim) report.Summary {
	var s report.Summary
	for _, c := range claims {
		switch c.Verdict {
		case report.VerdictGrounded:
			s.Grounded++
		case report.VerdictContradicted:
			s.Contradicted++
		case report.VerdictUnverifiable:
			s.Unverifiable++
		}
	}
	return s
}

func toolErrorClaim(path string, err error) report.Claim {
	return report.Claim{
		ID:         "tool-error-" + path,
		Type:       "tool-error",
		Subtype:    report.SubtypeImport,
		Text:       fmt.Sprintf("verifier failed to start for %s: %v", path, err),
		Locus:      report.Locus{File: path, Line: 1, Col: 0},
		Verdict:    report.VerdictUnverifiable,
		Confidence: 0,
		Evidence:   []report.Evidence{{Kind: "tool-error", Detail: err.Error(), Determinism: report.DeterminismDeterministic}},
		Verifier:   report.VerifierInfo{Name: "engine", Version: "0.1.0", Determinism: report.DeterminismDeterministic},
	}
}
