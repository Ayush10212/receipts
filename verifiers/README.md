# `verifiers/` — language checkers

Each subfolder is a **standalone subprocess that knows exactly one language**. The Go core
spawns it and talks to it over JSON-RPC 2.0 (defined in
[`../protocol/verifier-protocol.v0.json`](../protocol/verifier-protocol.v0.json)).

This is how Receipts stays language-agnostic: to support Ruby or JavaScript, you'd add a
new folder here — never edit the core.

## `python-symbols/` (the reference implementation)

The Python checker. Its pipeline:

1. **`env.py`** — figure out which Python interpreter / environment to verify against.
2. **`extract.py`** — read the file with `ast.parse` and list every symbol it uses
   (imports, attributes, keyword arguments). **The file is never imported or executed.**
3. **`introspect.py`** — in a *separate sandboxed subprocess*, import only the installed
   dependency and check each symbol with `getattr` / `inspect.signature`.
4. **`analyze.py`** — wire it together into claims with verdicts and evidence.

The golden rule: **never run the code under review, and when unsure, say `unverifiable` —
never a false `contradicted`.**
