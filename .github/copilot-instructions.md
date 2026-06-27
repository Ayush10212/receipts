# Copilot instructions

This repository is **Receipts** — a static checker that verifies AI-generated Python against
the *installed* packages, catching hallucinated symbols, removed methods, and wrong keyword
arguments before a human reviews the code.

## Before proposing Python code, fact-check it

- Build once: `go build -o receipts ./cmd/receipts/`
- Run: `./receipts check --format plain --python python <file.py>`
- Or use the MCP server in `.vscode/mcp.json` (tool: `receipts_check_code`), which returns a
  plain-English review plus the full report.
- One-time setup so the checker works: `pip install -e "verifiers/python-symbols"`.

## Rules

1. **Never execute the code under review.** Static analysis only.
2. **Three verdicts:** `grounded`, `contradicted`, `unverifiable`. When unsure →
   `unverifiable`, never a false `contradicted`.
3. **Exit codes:** `0` pass/warn · `1` contradicted · `2` tool error.

If a symbol comes back `contradicted`, it does not exist in the installed package — fix the
code (suggest the correct current API) before showing it to the user.
