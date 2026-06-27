# `protocol/` — the public contracts

Two JSON Schemas here are **versioned public API**. Anything built against `v0` keeps working
as the project grows, because changes are **additive-only** (a breaking change would be a new
`v1`, never an edit to `v0`). CI validates fixtures against these schemas on every push.

| File | What it defines |
|---|---|
| `report.v0.json` | The shape of a Receipts **Report** — runs, claims, verdicts, evidence, policy, summary. See [`../docs/report-format.md`](../docs/report-format.md). |
| `verifier-protocol.v0.json` | The **JSON-RPC 2.0 protocol** a language checker must speak (`initialize` → `analyze` → `shutdown`). See [`../docs/verifier-protocol.md`](../docs/verifier-protocol.md). |

`testdata/` holds example reports (grounded, contradicted, unverifiable, and a deliberately
malformed one) used to prove the schema accepts what it should and rejects what it shouldn't.
