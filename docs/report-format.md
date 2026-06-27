# Report format (`report.v0.json`)

The Report is the single output of a Receipts run — the public, machine-readable contract every surface (CLI `--format json`, MCP tool, SARIF exporter) is built on. The authoritative definition is [`protocol/report.v0.json`](../protocol/report.v0.json), a JSON Schema 2020-12 document. This page is a human-readable companion.

## Stability

`report.v0.json` is **public API**. Changes are **additive-only** from `schema_version` `0.1.0`: new optional fields may be added, but existing fields are never removed, renamed, or have their meaning changed. A breaking change would ship as `report.v1.json`. Schema validation runs as a required CI gate (`go test ./protocol/...`).

## Top-level shape

```jsonc
{
  "schema_version": "0.1.0",   // const — pin your parser to this
  "run":     { ... },          // metadata about this invocation
  "claims":  [ ... ],          // one entry per verified symbol reference
  "policy":  { ... },          // the pass/warn/fail decision
  "summary": { ... }           // verdict counts
}
```

All five top-level keys are required, and `additionalProperties` is `false` everywhere — an unknown key is a schema violation, which keeps producers honest.

### `run`

| Field | Type | Meaning |
|---|---|---|
| `id` | string | Unique id for this run (`run-<nanos>`). |
| `timestamp` | date-time | UTC start time. |
| `tool_version` | string | Receipts version that produced the report. |
| `inputs_hash` | string | `sha256` over the sorted file contents + target-env identity. Same inputs → same hash; the report is reproducible. |
| `target_env` | object | `{ language, version, prefix }` — the **installed** environment claims were checked against. |

### `claims[]`

One claim per symbol reference Receipts extracted and judged.

| Field | Type | Meaning |
|---|---|---|
| `id` | string | Stable id within the report. |
| `type` | string | Claim family, e.g. `symbol` (or `tool-error` for a degradation). |
| `subtype` | enum | `import` \| `attribute` \| `kwarg`. |
| `text` | string | The dotted symbol, e.g. `pandas.DataFrame.append`. |
| `locus` | object | `{ file, line, col, end_line?, end_col? }` — where it appears in the source. |
| `verdict` | enum | `grounded` \| `contradicted` \| `unverifiable`. |
| `confidence` | number | `0..1`. `1.0` for a definitive deterministic verdict; `0.0` for `unverifiable`. |
| `evidence[]` | array | Why this verdict was reached (see below). |
| `verifier` | object | `{ name, version, determinism }` — which plugin produced the claim. |

### `evidence[]`

| Field | Type | Meaning |
|---|---|---|
| `kind` | string | e.g. `introspection`, `tool-error`, `env-error`, `llm-note`. |
| `detail` | string | Human-readable explanation, e.g. `'pandas.DataFrame.append' not found (AttributeError on 'append')`. |
| `determinism` | enum | `deterministic` \| `subjective`. |

**The honesty rule, encoded here:** an `llm-note` evidence entry always carries `determinism: "subjective"` and only ever appears on a `contradicted` or `unverifiable` claim — never on `grounded`, and it never changes the claim's `verdict`. Everything that decides a verdict is `deterministic`.

### `policy`

| Field | Type | Meaning |
|---|---|---|
| `backend` | string | Policy backend name (e.g. `local`). |
| `decision` | enum | `pass` \| `warn` \| `fail`. |
| `rules_applied` | string[] | The rules that fired. |

The local backend: `fail` if any claim is `contradicted`; `warn` if any is `unverifiable` (and none contradicted); else `pass`. The CLI maps `fail` → exit 1.

### `summary`

Counts across all claims: `{ grounded, contradicted, unverifiable }`. These are computed before any LLM step and are immutable thereafter — toggling `--llm` never changes them.

## Example

```json
{
  "schema_version": "0.1.0",
  "run": {
    "id": "run-1782563727887885600",
    "timestamp": "2026-06-27T12:00:00Z",
    "tool_version": "0.1.0",
    "inputs_hash": "sha256:…",
    "target_env": { "language": "python", "version": "3.11.4", "prefix": "/home/u/.venv" }
  },
  "claims": [
    {
      "id": "c-001", "type": "symbol", "subtype": "attribute",
      "text": "pandas.DataFrame.append",
      "locus": { "file": "app/etl.py", "line": 42, "col": 0 },
      "verdict": "contradicted", "confidence": 1.0,
      "evidence": [
        { "kind": "introspection", "detail": "'pandas.DataFrame.append' not found (AttributeError on 'append')", "determinism": "deterministic" }
      ],
      "verifier": { "name": "python-symbols", "version": "0.1.0", "determinism": "deterministic" }
    }
  ],
  "policy": { "backend": "local", "decision": "fail", "rules_applied": ["fail-on-contradicted"] },
  "summary": { "grounded": 0, "contradicted": 1, "unverifiable": 0 }
}
```
