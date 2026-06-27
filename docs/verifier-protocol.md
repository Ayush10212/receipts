# Verifier protocol (`verifier-protocol.v0.json`)

A **verifier plugin** is a subprocess that knows one language. The Go core knows none. They talk over **JSON-RPC 2.0 on stdio**. This is the seam that lets you add a language without touching the core. The authoritative definition is [`protocol/verifier-protocol.v0.json`](../protocol/verifier-protocol.v0.json); this page explains it.

## Stability

Like the report, this is **public API with additive-only changes** from `v0`. A plugin written against `v0` keeps working as the protocol grows. Validated in CI.

## Transport

- One plugin process per language, spawned by the core.
- The core writes a JSON-RPC **request** as a single line to the plugin's **stdin**; the plugin writes a JSON-RPC **response** to **stdout**.
- The plugin must keep **stdout** clean — only JSON-RPC responses. Logs, warnings, and notices go to **stderr**.
- Each call has a timeout enforced by the core. A timed-out or crashed call degrades the affected file to a single `unverifiable` tool-error claim — it never blocks the run or fabricates a verdict.

## Methods

The lifecycle is `initialize` → one or more `analyze` → `shutdown`.

### `initialize`

The core's handshake. The plugin announces who it is and what it can do.

**Request:** `params: {}`

**Result:**

| Field | Type | Meaning |
|---|---|---|
| `name` | string | Plugin name, e.g. `python-symbols`. |
| `version` | string | Plugin version. |
| `capabilities` | string[] | Languages/features handled, e.g. `["python"]`. |
| `determinism` | enum | `deterministic` \| `subjective`. A symbol verifier is `deterministic`. |

The core stamps `{ name, version, determinism }` onto every claim the plugin returns, so claims don't have to repeat their own provenance.

### `analyze`

The work. One artifact in, claims out.

**Request `params`:**

```jsonc
{
  "artifact": {
    "path": "main.py",
    "content": "import pandas as pd\npd.DataFrame.append()"
  },
  "context": {
    "workdir": "/project",
    "language": "python",
    "target_env": { "language": "python", "version": "3.11.4", "prefix": "/home/u/.venv" }
  }
}
```

**Result:** `{ "claims": [ <claim>, ... ] }`, where each claim matches `report.v0.json`'s claim shape **minus** the `verifier` field (the core fills that in from `initialize`):

```jsonc
{
  "id": "c-001", "type": "symbol", "subtype": "attribute",
  "text": "pandas.DataFrame.append",
  "locus": { "file": "main.py", "line": 2, "col": 0 },
  "verdict": "contradicted", "confidence": 1.0,
  "evidence": [{ "kind": "introspection", "detail": "removed in pandas 2.0", "determinism": "deterministic" }]
}
```

### `shutdown`

**Request:** `params: {}` → **Result:** `{}`. The plugin should exit its read loop and terminate.

## Non-negotiable rules for any plugin

These mirror the core's guarantees. A plugin that breaks them breaks Receipts' credibility:

1. **Never execute the artifact.** Parse it statically (for Python, `ast.parse`). The `content` you receive is untrusted source, not something to import or `exec`.
2. **Introspect only the *installed* dependency**, and do it in a **sandboxed subprocess** with a timeout — never in the plugin's main process.
3. **Ambiguity → `unverifiable`, never `contradicted`.** Import errors, C-extension opacity, erased/decorated signatures, `**kwargs`, untraceable receivers, timeouts — all `unverifiable`. Only emit `contradicted` when a resolvable target *provably* lacks the symbol.
4. **A failure on one symbol degrades only that symbol.** It must never contaminate other claims in the same file.

## Worked example: the exchange

```text
→  {"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}
←  {"jsonrpc":"2.0","id":1,"result":{"name":"python-symbols","version":"0.1.0","capabilities":["python"],"determinism":"deterministic"}}

→  {"jsonrpc":"2.0","id":2,"method":"analyze","params":{"artifact":{"path":"main.py","content":"import pandas as pd\npd.DataFrame.append()"},"context":{"workdir":"/project","language":"python","target_env":{"language":"python","version":"3.11.4","prefix":"/home/u/.venv"}}}}
←  {"jsonrpc":"2.0","id":2,"result":{"claims":[{"id":"c-001","type":"symbol","subtype":"attribute","text":"pandas.DataFrame.append","locus":{"file":"main.py","line":2,"col":0},"verdict":"contradicted","confidence":1.0,"evidence":[{"kind":"introspection","detail":"removed in pandas 2.0","determinism":"deterministic"}]}]}}

→  {"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}
←  {"jsonrpc":"2.0","id":3,"result":{}}
```

The reference implementation is [`verifiers/python-symbols`](../verifiers/python-symbols). To add a language, write a new plugin that speaks this protocol — a standalone subprocess in whatever language best introspects that ecosystem, with **zero changes** to the Go core.
