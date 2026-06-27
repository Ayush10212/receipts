"""JSON-RPC 2.0 stdio loop for the python-symbols verifier plugin."""

import json
import sys
from typing import Any


VERSION = "0.1.0"


def _respond(out, id_: Any, result: Any = None, error: dict = None) -> None:
    msg: dict = {"jsonrpc": "2.0", "id": id_}
    if error is not None:
        msg["error"] = error
    else:
        msg["result"] = result
    print(json.dumps(msg), file=out, flush=True)


def _error(code: int, message: str) -> dict:
    return {"code": code, "message": message}


def run_loop(stdin=None, stdout=None) -> None:
    """Block reading JSON-RPC requests from stdin, writing responses to stdout."""
    if stdin is None:
        stdin = sys.stdin
    if stdout is None:
        stdout = sys.stdout

    # Import here so the module loads fast even if analyze deps are missing.
    from receipts_python_symbols.analyze import analyze

    for raw in stdin:
        raw = raw.strip()
        if not raw:
            continue

        try:
            req = json.loads(raw)
        except json.JSONDecodeError as exc:
            _respond(stdout, None, error=_error(-32700, f"Parse error: {exc}"))
            continue

        id_ = req.get("id")
        method = req.get("method", "")
        params = req.get("params") or {}

        if method == "initialize":
            _respond(stdout, id_, {
                "name": "python-symbols",
                "version": VERSION,
                "capabilities": ["python"],
                "determinism": "deterministic",
            })

        elif method == "analyze":
            try:
                result = analyze(params)
                _respond(stdout, id_, result)
            except Exception as exc:  # noqa: BLE001 — degrade, never crash the loop
                _respond(stdout, id_, error=_error(-32603, f"analyze error: {exc}"))

        elif method == "shutdown":
            _respond(stdout, id_, {})
            break

        else:
            _respond(stdout, id_, error=_error(-32601, f"Method not found: {method}"))
