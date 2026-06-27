# `core/` — the Go engine

This is the brain of Receipts, written in pure Go. **It knows zero Python** (or any other
language). It orchestrates the run, talks to language checkers over a JSON-RPC protocol,
assembles the results into a Report, applies a pass/fail policy, and prints the output.

Adding a new language never touches this folder — that's the whole design.

| Package | What it does |
|---|---|
| `engine/` | The orchestrator: take files + config → run the checker → build the Report → apply policy → emit. |
| `report/` | The Report data structures, plus `explain.go` (the plain-English translation layer). |
| `verifier/` | Spawns a language-checker subprocess and speaks JSON-RPC 2.0 to it; handles timeouts and crashes gracefully. |
| `config/` | Loads `.receipts.yaml` settings, walking up from the working directory. |
| `policy/` | Turns a Report into a decision: `pass`, `warn`, or `fail`. |
| `sink/` | Writes the Report out: `plain` (human review), `pretty`, `json`, or `sarif`. |
| `cache/` | On-disk cache so the same symbol isn't re-checked every run. |

**Key rule:** a crash or timeout while checking one symbol degrades only *that* symbol to
`unverifiable`. It never breaks the run or contaminates other results.
