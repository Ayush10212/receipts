//go:build benchmark

// Package benchmark runs the quality-gate corpus against the real engine.
// Gates: catch rate >= 80%, false-positive rate < 10%.
//
// This file is gated behind the `benchmark` build tag so it does NOT run as part
// of the everyday `go test ./...` (each case spawns a Python subprocess, ~4min total).
// Run it explicitly with `make benchmark` (a required CI gate).
package benchmark_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/Ayush10212/receipts/core/config"
	"github.com/Ayush10212/receipts/core/engine"
	"github.com/Ayush10212/receipts/core/policy"
	"github.com/Ayush10212/receipts/core/report"
	"github.com/Ayush10212/receipts/core/verifier"
)

type caseExpected struct {
	Verdict       string `json:"verdict"`
	ClaimContains string `json:"claim_contains"`
}

type benchCase struct {
	ID          string       `json:"id"`
	Description string       `json:"description"`
	Code        string       `json:"code"`
	Expected    caseExpected `json:"expected"`
	Category    string       `json:"category"`
}

func pythonExe(t *testing.T) string {
	t.Helper()
	for _, c := range []string{"python", "python3"} {
		if p, err := exec.LookPath(c); err == nil {
			return p
		}
	}
	t.Skip("no python interpreter found; skipping benchmark")
	return ""
}

func casesDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "cases")
}

func loadSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	schemaPath := filepath.Join(casesDir(), "case.schema.json")
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read case schema: %v", err)
	}
	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse case schema: %v", err)
	}
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	c.AddResource("case.schema.json", doc)
	sch, err := c.Compile("case.schema.json")
	if err != nil {
		t.Fatalf("compile case schema: %v", err)
	}
	return sch
}

func loadCases(t *testing.T) []benchCase {
	t.Helper()
	sch := loadSchema(t)
	dir := casesDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read cases dir: %v", err)
	}

	var cases []benchCase
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".json") || name == "case.schema.json" {
			continue
		}
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		var doc any
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		if err := sch.Validate(doc); err != nil {
			t.Fatalf("case %s failed schema validation: %v", name, err)
		}
		var bc benchCase
		if err := json.Unmarshal(data, &bc); err != nil {
			t.Fatalf("unmarshal %s: %v", name, err)
		}
		cases = append(cases, bc)
	}
	return cases
}

func runCase(t *testing.T, bc benchCase, py string) report.Report {
	t.Helper()
	cfg := config.Config{}
	backends := engine.Backends{
		Policy: policy.NoneBackend{},
		NewClient: func(ctx context.Context, _ report.ExecutionContext, _ string) (*verifier.Client, error) {
			// 90s per call: generous for slow CI machines; avoids goroutine leaks from spurious timeouts.
			return verifier.NewClient(ctx, 90*time.Second, py, "-m", "receipts_python_symbols")
		},
	}
	artifacts := []engine.Artifact{
		{Path: "test.py", Content: []byte(bc.Code), Language: "python"},
	}
	r, err := engine.Run(context.Background(), report.ExecutionContext{}, artifacts, cfg, backends)
	if err != nil {
		t.Fatalf("case %s: engine error: %v", bc.ID, err)
	}
	return r
}

// TestCaseSchema validates every case file against the case schema.
func TestCaseSchema(t *testing.T) {
	cases := loadCases(t)
	if len(cases) == 0 {
		t.Fatal("no benchmark cases found")
	}
	t.Logf("validated %d cases against case schema", len(cases))
}

// TestBenchmark_QualityGates runs the corpus against the real engine and
// enforces the two hard CI gates: catch rate >= 80%, false-positive rate < 10%.
func TestBenchmark_QualityGates(t *testing.T) {
	py := pythonExe(t)
	cases := loadCases(t)
	if len(cases) == 0 {
		t.Fatal("no benchmark cases found")
	}

	var (
		contradictedTotal  int
		contradictedCaught int
		groundedTotal      int
		groundedFP         int
	)

	for _, bc := range cases {
		r := runCase(t, bc, py)

		hasContradicted := false
		for _, cl := range r.Claims {
			if cl.Verdict == report.VerdictContradicted {
				hasContradicted = true
				break
			}
		}

		switch bc.Expected.Verdict {
		case "contradicted":
			contradictedTotal++
			if hasContradicted {
				contradictedCaught++
			} else {
				t.Logf("MISS  %s: %s", bc.ID, bc.Description)
			}
		case "grounded":
			groundedTotal++
			if hasContradicted {
				groundedFP++
				t.Logf("FP    %s: %s", bc.ID, bc.Description)
			}
		case "unverifiable":
			// Unverifiable cases must NOT produce any contradicted claim (false-positive risk).
			if hasContradicted {
				groundedFP++ // count as FP against the grounded pool
				t.Logf("FP-UV %s: %s (unverifiable case got contradicted)", bc.ID, bc.Description)
			}
		}
	}

	if contradictedTotal == 0 {
		t.Fatal("no contradicted cases in benchmark corpus")
	}

	catchRate := float64(contradictedCaught) / float64(contradictedTotal)
	fpDenom := groundedTotal // unverifiable FPs already counted; denominator is grounded-only for the rate
	var fpRate float64
	if fpDenom > 0 {
		fpRate = float64(groundedFP) / float64(fpDenom)
	}

	summary := fmt.Sprintf(
		"catch rate: %.1f%% (%d/%d caught)  |  false-positive rate: %.1f%% (%d/%d grounded flagged)",
		catchRate*100, contradictedCaught, contradictedTotal,
		fpRate*100, groundedFP, fpDenom,
	)
	t.Log(summary)

	if catchRate < 0.80 {
		t.Errorf("GATE FAIL: catch rate %.1f%% < required 80%%", catchRate*100)
	}
	if fpRate >= 0.10 {
		t.Errorf("GATE FAIL: false-positive rate %.1f%% >= allowed 10%%", fpRate*100)
	}
}
