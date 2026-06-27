# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this project is

**Receipts** is a static analysis tool that verifies AI-generated Python code against installed dependencies. It catches hallucinated symbols, removed methods, and wrong kwargs *before* a human sees the code — operating as a gate in the agent loop via an MCP server.

Three hard invariants that must never break:
1. **Never execute the code-under-test.** Parse it statically (AST) only.
2. **Deterministic verdicts only:** `grounded | contradicted | unverifiable`. Ambiguity → `unverifiable`, never `contradicted`.
3. **LLM output is advisory prose only**, tagged `determinism:"subjective"`. It must never create or change a verdict.

## Build & test commands

All commands run from `receipts/` (the monorepo root inside `paper-trail/`).

```bash
# Full suite (Go tests + Python tests + schema validation)
make test

# Go only
go test ./...

# Single Go package
go test ./core/engine/...

# Python only (from verifiers/python-symbols/)
python -m pytest

# Single Python test file
python -m pytest tests/test_extract.py

# Schema validation only
go test ./protocol/...

# Build the CLI binary
go build ./cmd/receipts/
```

Install the Python verifier in editable mode before running Python tests:
```bash
cd verifiers/python-symbols
pip install -e ".[dev]"
```

## Architecture

### Two-language split

The Go core knows **zero Python**. All language knowledge lives behind a JSON-RPC 2.0 boundary:

```
receipts check [files]
        │
   core/engine          ← pure Go: (artifacts, config, backends) → Report
        │  JSON-RPC 2.0 over stdio
   verifiers/python-symbols   ← pure Python: AST extract + sandboxed introspect
```

Adding a new language = a new verifier plugin implementing `verifier-protocol.v0.json`. Zero core changes.

### Go packages (in `core/`)

| Package | Role |
|---|---|
| `core/engine` | Orchestrator: resolve env → dispatch files → assemble Report → policy → sink |
| `core/report` | Go structs mirroring `protocol/report.v0.json` 1:1 |
| `core/verifier` | Spawns verifier subprocess, speaks JSON-RPC, handles timeouts/crashes gracefully |
| `core/config` | Loads `.receipts.yaml` hierarchy from workdir upward |
| `core/policy` | Evaluates Report → Decision (`pass/warn/fail`); local rule: fail-on-contradicted |
| `core/sink` | Emits Reports: pretty (stdout), JSON, SARIF 2.1.0 |
| `core/cache` | Disk cache keyed by `(dist, version, dotted_symbol)` |

### Python verifier (`verifiers/python-symbols/`)

Pipeline inside `analyze()`:

1. **`env.py`** — resolve interpreter (flag → `$VIRTUAL_ENV` → `.venv` → poetry/uv/pdm → FAIL; never silent)
2. **`extract.py`** — `ast.parse` the file; emit claims for imports, attribute refs, kwargs (never imports/executes the file)
3. **`introspect.py`** — in a sandboxed worker subprocess, `import` the *installed* dependency and `getattr`/`inspect.signature` it
4. **`analyze.py`** — wire extract → introspect → claims with verdicts + evidence

### Surfaces (`adapters/`)

All three are thin shells — no verdict logic:
- `adapters/cli` + `cmd/receipts/` → `receipts check` CLI
- `adapters/precommit/` → pre-commit hook (calls `receipts check --staged`)
- `adapters/mcp/` → MCP server exposing `receipts_check_code({code, language, workdir})`

### LLM layer (`llm/`)

Off by default (`--llm` flag / `llm.enabled: false`). When on, runs **after** the deterministic Report is fully assembled. Adds `kind:"llm-note", determinism:"subjective"` evidence entries to `contradicted`/`unverifiable` claims only. Supports Mistral and Grok (xAI) via one shared OpenAI-compatible client, configured in `.receipts.yaml`.

### Versioned public contracts (`protocol/`)

`report.v0.json` and `verifier-protocol.v0.json` are public API. **Additive-only changes.** Schema validation runs in CI as a required gate (`go test ./protocol/...`).

## Verdict rules (encode these, never relax them)

- `grounded` — symbol/attr/kwarg confirmed present in the installed package
- `contradicted` — resolvable module/class provably lacks the attr, or signature provably lacks the kwarg
- `unverifiable` — import error, C-extension opacity, erased/decorated signature, timeout, crash, untraceable receiver, or any ambiguity
- A crash on one symbol degrades **only that symbol** to `unverifiable`; it never contaminates other claims

## Exit codes (CLI)

| Code | Meaning |
|---|---|
| 0 | Pass |
| 1 | Policy fail (≥1 contradicted when `--fail-on contradicted`) |
| 2 | Tool error |

## CI quality gates

Two gates are enforced as required CI checks:
- **Honesty test** — runs a fixed corpus with LLM on/off; asserts byte-identical verdicts, decisions, and summaries; asserts no `llm-note` on a `grounded` claim
- **Benchmark** — catch rate ≥ 80%, false-positive rate < 10% on `testdata/benchmark/cases/`

## Config (`.receipts.yaml`)

```yaml
llm:
  enabled: false          # --llm flag overrides per-run
  primary: mistral        # mistral | grok
  fallback: grok
  mistral_model: <model-id>
  grok_model: <model-id>
  timeout_ms: 8000
```

LLM API keys via env vars: `MISTRAL_API_KEY`, `XAI_API_KEY`.
