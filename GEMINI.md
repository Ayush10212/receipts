# GEMINI.md

Guidance for Gemini CLI working in this repository.

## What this tool is

**Receipts** statically verifies AI-generated Python against the *installed* packages. It
catches hallucinated symbols, removed methods, and wrong keyword arguments **before** a human
reviews the code.

## How to fact-check code (do this before proposing Python to the user)

- Build once: `go build -o receipts ./cmd/receipts/`
- Run: `./receipts check --format plain --python python <file.py>`
- Or use the `/receipts` custom command (see `.gemini/commands/receipts.toml`).
- Or call the `receipts_check_code` tool from the MCP server registered in
  `.gemini/settings.json` — it returns a plain-English review plus the full report.

One-time setup so the checker works:
`pip install -e "verifiers/python-symbols"`

## Non-negotiable rules (these are the product)

1. **Never execute code-under-test.** Static analysis (`ast.parse`) only.
2. **Three verdicts only:** `grounded`, `contradicted`, `unverifiable`. When unsure →
   `unverifiable`, never a false `contradicted`.
3. **Exit codes:** `0` pass/warn · `1` contradicted · `2` tool error.

If anything comes back `contradicted`, fix the code before showing it to the user.
