package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Ayush10212/receipts/core/config"
	"github.com/Ayush10212/receipts/core/engine"
	"github.com/Ayush10212/receipts/core/policy"
	"github.com/Ayush10212/receipts/core/report"
	"github.com/Ayush10212/receipts/core/sink"
	"github.com/Ayush10212/receipts/core/verifier"
	"github.com/Ayush10212/receipts/llm"
)

const (
	ExitPass       = 0
	ExitPolicyFail = 1
	ExitToolError  = 2
)

type Options struct {
	Paths           []string
	Staged          bool
	PythonPath      string
	Format          string // pretty | json | sarif
	FailOn          string // contradicted | none
	CheckSignatures bool
	Explain         bool
	Workdir         string
	LLMEnabled      bool
}

// Run is the entry point for `receipts check`. Returns an exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("receipts check", flag.ContinueOnError)
	fs.SetOutput(stderr)

	staged := fs.Bool("staged", false, "check git staged files")
	python := fs.String("python", "", "path to Python interpreter")
	format := fs.String("format", "pretty", "output format: pretty|json|sarif")
	failOn := fs.String("fail-on", "contradicted", "fail policy: contradicted|none")
	_ = fs.Bool("check-signatures", true, "verify kwarg signatures (default on)")
	explain := fs.Bool("explain", false, "print resolved target env and claim detail")
	enableLLM := fs.Bool("llm", false, "enable LLM advisory annotations (requires API key)")

	if err := fs.Parse(args); err != nil {
		return ExitToolError
	}

	workdir, _ := os.Getwd()

	opts := Options{
		Paths:      fs.Args(),
		Staged:     *staged,
		PythonPath: *python,
		Format:     *format,
		FailOn:     *failOn,
		Explain:    *explain,
		Workdir:    workdir,
		LLMEnabled: *enableLLM,
	}

	return runCheck(context.Background(), opts, stdout, stderr)
}

func runCheck(ctx context.Context, opts Options, stdout, stderr io.Writer) int {
	// Collect files.
	paths, err := collectPaths(opts)
	if err != nil {
		fmt.Fprintf(stderr, "receipts: %v\n", err)
		return ExitToolError
	}
	if len(paths) == 0 {
		fmt.Fprintln(stderr, "receipts: no files to check")
		return ExitToolError
	}

	// Load config.
	cfg, err := config.FileProvider{}.Load(opts.Workdir)
	if err != nil {
		fmt.Fprintf(stderr, "receipts: config error: %v\n", err)
		return ExitToolError
	}

	// Build backends.
	var outputSink sink.ReportSink
	switch opts.Format {
	case "json":
		outputSink = sink.JSONSink{W: stdout}
	case "sarif":
		outputSink = sink.SARIFSink{W: stdout}
	case "plain":
		outputSink = sink.PlainSink{W: stdout}
	default:
		outputSink = sink.PrettySink{W: stdout}
	}

	var policyBackend policy.Backend
	if opts.FailOn == "none" {
		policyBackend = policy.NoneBackend{}
	} else {
		policyBackend = policy.LocalBackend{}
	}

	// Sink is wired manually (after optional LLM annotation).
	backends := engine.Backends{
		Policy:    policyBackend,
		NewClient: pythonClientFactory(opts.PythonPath),
	}

	// Build artifacts.
	artifacts := make([]engine.Artifact, 0, len(paths))
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			fmt.Fprintf(stderr, "receipts: read %s: %v\n", p, err)
			return ExitToolError
		}
		artifacts = append(artifacts, engine.Artifact{
			Path:     p,
			Content:  data,
			Language: "python",
		})
	}

	ectx := report.ExecutionContext{}

	// Run engine (deterministic phase — no LLM here).
	r, err := engine.Run(ctx, ectx, artifacts, cfg, backends)
	if err != nil {
		fmt.Fprintf(stderr, "receipts: engine error: %v\n", err)
		return ExitToolError
	}

	// Optional LLM annotation — runs AFTER verdicts are frozen.
	if opts.LLMEnabled || cfg.LLM.Enabled {
		router := llm.NewRouterFromEnv(cfg.LLM, nil)
		if annotateErr := tryAnnotate(ctx, &r, router, stderr); annotateErr != nil {
			// Non-fatal: print notice and continue with deterministic output.
			fmt.Fprintf(stderr, "receipts: LLM annotation skipped: %v\n", annotateErr)
		}
	}

	// In --explain mode with a non-pretty format, surface the resolved target env
	// (the pretty sink already prints it, so avoid duplicating there).
	if opts.Explain && opts.Format != "pretty" {
		fmt.Fprintf(stdout, "Target env: %s %s (%s)\n",
			r.Run.TargetEnv.Language, r.Run.TargetEnv.Version, r.Run.TargetEnv.Prefix)
	}

	// Emit report.
	if err := outputSink.Emit(ctx, ectx, r); err != nil {
		fmt.Fprintf(stderr, "receipts: sink error: %v\n", err)
		return ExitToolError
	}

	switch r.Policy.Decision {
	case report.DecisionFail:
		return ExitPolicyFail
	case report.DecisionPass, report.DecisionWarn:
		return ExitPass
	default:
		return ExitToolError
	}
}

func collectPaths(opts Options) ([]string, error) {
	if opts.Staged {
		return stagedPythonFiles()
	}
	return opts.Paths, nil
}

func stagedPythonFiles() ([]string, error) {
	out, err := exec.Command("git", "diff", "--cached", "--name-only", "--diff-filter=ACM").Output()
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}
	var paths []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && strings.HasSuffix(line, ".py") {
			paths = append(paths, line)
		}
	}
	return paths, nil
}

// tryAnnotate calls llm.Annotate. If the router has no API keys it prints a notice
// to stderr and returns without error (degrade to deterministic output).
func tryAnnotate(ctx context.Context, r *report.Report, router *llm.Router, stderr io.Writer) error {
	if router.Disabled() {
		fmt.Fprintln(stderr, "receipts: --llm requested but no API key configured (MISTRAL_API_KEY / XAI_API_KEY); running deterministic-only")
		return nil
	}
	llm.Annotate(ctx, r, router)
	return nil
}

func pythonClientFactory(pythonPath string) verifier.ClientFactory {
	return func(ctx context.Context, _ report.ExecutionContext, _ string) (*verifier.Client, error) {
		exe := "python"
		if pythonPath != "" {
			exe = pythonPath
		}
		return verifier.NewClient(ctx, 30*time.Second, exe, "-m", "receipts_python_symbols")
	}
}
