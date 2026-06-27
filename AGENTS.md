# AGENTS.md

Guidance for AI agents (Codex, Claude Code, Copilot, etc.) working in this repository.

## What this tool does

Receipts statically verifies AI-generated Python code against installed packages. It catches hallucinated symbols before a human reviews the code. The MCP surface (`adapters/mcp/`) is the primary integration point for agents — call `receipts_check_code` to self-check code before proposing it.

## Non-negotiable constraints

These are not style preferences. Violating them breaks the product's credibility:

1. **Never execute code-under-test.** Use `ast.parse` only. No `exec`, no `importlib`, no `subprocess` on user files.
2. **Three verdicts, no others:** `grounded`, `contradicted`, `unverifiable`. When in doubt → `unverifiable`. Never guess `contradicted`.
3. **LLM output never touches verdicts.** It adds `kind:"llm-note", determinism:"subjective"` evidence entries only, and only on `contradicted`/`unverifiable` claims. Never on `grounded`.
4. **Schemas are additive-only.** `protocol/report.v0.json` and `protocol/verifier-protocol.v0.json` are public API from commit 1. Never remove or rename fields.

## Before writing any code

Run `go test ./...` and `python -m pytest` (from `verifiers/python-symbols/`) to confirm the baseline is green. If either is red, fix it before adding new code.

## Language boundary — respect it

- **Go** (`core/`, `adapters/`, `llm/`, `cmd/`) knows nothing about Python syntax, AST, or packages.
- **Python** (`verifiers/python-symbols/`) knows nothing about Go types or the Report schema beyond what `verifier-protocol.v0.json` defines.
- Communication crosses this boundary **only** via JSON-RPC 2.0 over stdio, per `protocol/verifier-protocol.v0.json`.

If you are tempted to parse Python in Go or call Go from Python directly, stop — add a protocol method instead.

## Verifier subprocess safety

`introspect.py` imports installed packages in a sandboxed worker subprocess. Rules:
- Use `rlimits` and a wall-clock timeout on every introspection call.
- A timeout or crash on one symbol → `unverifiable` for that symbol only. Never propagate the failure to other symbols or the whole file.
- Never import the user's own code (the code-under-test). Only import from the installed environment.

## Env detection — loud failures only

`env.py` must resolve the Python interpreter in this order and **fail loudly** if none is found:
1. `--python` flag
2. `$VIRTUAL_ENV`
3. `./.venv` or `./venv`
4. `poetry`/`uv`/`pdm` managed venv
5. **FAIL** — never fall back to system Python silently.

The resolved env must always be printed in `--explain` and pretty output.

## Adding a new language verifier

1. Implement `initialize` / `analyze` / `shutdown` per `protocol/verifier-protocol.v0.json`.
2. Return `claims[]` — the Go core assembles the Report.
3. No changes to `core/` are needed.
4. Add the verifier under `verifiers/<language>/`.

## Schema changes

Any change to `protocol/report.v0.json` or `protocol/verifier-protocol.v0.json`:
- Must be additive only (new optional fields).
- Must bump the minor version in `schema_version`.
- Must update the sample reports in `protocol/schema_test.go` to exercise the new field.
- CI will reject a PR if `go test ./protocol/...` fails.

## CI gates that must stay green

| Gate | Command | Threshold |
|---|---|---|
| Go tests | `go test ./...` | all pass |
| Python tests | `python -m pytest` | all pass |
| Schema validation | `go test ./protocol/...` | all pass |
| Honesty invariant | part of `go test ./...` | verdicts identical with/without `--llm` |
| Benchmark | `go run testdata/benchmark/run_benchmark.go` | catch ≥ 80%, FP < 10% |

Never mark work complete if any gate is red.

## LLM layer rules

- `llm.enabled` defaults to `false`. The `--llm` flag overrides per-run.
- With no API key set, `--llm` must degrade gracefully: print a notice, produce deterministic output, never crash.
- Read model IDs from config (`.receipts.yaml`). Never hardcode model names.
- The Router tries `llm.primary`, retries once, then falls back to `llm.fallback`.
