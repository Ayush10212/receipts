# `adapters/` — the surfaces

These are the thin shells that let you *use* Receipts. They contain **no verdict logic** —
they collect input, call the core engine, and present the result. All the real work happens
in [`../core/`](../core/).

| Adapter | What it is |
|---|---|
| `cli/` | The `receipts check` command line tool. The main way humans run it. |
| `mcp/` | An [MCP](https://modelcontextprotocol.io) server exposing `receipts_check_code`, so an AI can fact-check its own code before showing it to you. Returns a plain-English review **and** the full report. |
| `precommit/` | A pre-commit hook so hallucinated code is blocked before it lands in git. |

Because all three share the same core, they always agree on the verdict — they just differ
in how you invoke them and how the answer is displayed.
