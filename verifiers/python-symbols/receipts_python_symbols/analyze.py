"""Wire extract → introspect → claims with verdicts + evidence."""

import sys
import uuid
from typing import Any

from receipts_python_symbols.extract import extract, RawClaim
from receipts_python_symbols.introspect import introspect


_VERSION = "0.1.0"


def analyze(params: dict) -> dict:
    """
    Entry point called from the RPC loop.
    params follows verifier-protocol.v0.json#/$defs/analyze_params.
    Returns {"claims": [...]}.
    """
    artifact = params.get("artifact", {})
    context = params.get("context", {})

    path = artifact.get("path", "<unknown>")
    content = artifact.get("content", "")

    # Resolve Python interpreter from context or environment.
    python_exe = _resolve_python(context)

    # Gate: if no env, all claims are unverifiable (env error is logged to stderr).
    env_error: str | None = None
    if python_exe is None:
        env_error = "no Python environment found; set --python, activate a venv, or use poetry/uv/pdm"

    # AST extraction — static only, never executes the file.
    raw_claims = extract(content, path)

    claims = []
    seen_imports: set[str] = set()   # deduplicate import claims by symbol

    for rc in raw_claims:
        if rc.subtype == "import":
            # Emit one import claim per unique symbol.
            if rc.symbol in seen_imports:
                continue
            seen_imports.add(rc.symbol)
            claim = _ground_claim(rc, python_exe, env_error, check_kwarg=False)
        elif rc.subtype == "attribute":
            claim = _ground_claim(rc, python_exe, env_error, check_kwarg=False)
        elif rc.subtype == "kwarg":
            claim = _ground_claim(rc, python_exe, env_error, check_kwarg=True)
        else:
            continue
        claims.append(claim)

    return {"claims": claims}


def _resolve_python(context: dict) -> str | None:
    """
    Try to find the python exe from the analyze context, then from the environment.
    Returns None if nothing is found (caller degrades to unverifiable).
    """
    # If the context carries a resolved prefix, use that interpreter.
    target_env = context.get("target_env") or {}
    prefix = target_env.get("prefix", "")

    if prefix and prefix not in ("unknown", ""):
        import os
        from pathlib import Path
        for rel in ("python.exe", "python", "bin/python", "bin/python3", "Scripts/python.exe", "Scripts/python"):
            p = Path(prefix) / rel
            if p.exists():
                return str(p)

    # Try to resolve from the workdir using env.py logic.
    workdir = context.get("workdir", ".")
    try:
        from receipts_python_symbols.env import resolve
        env = resolve(workdir)
        return env.python
    except Exception as exc:
        # Last resort: use the interpreter running this verifier plugin.
        # This is NOT silent — we print a notice to stderr so the operator knows.
        print(
            f"receipts: env resolution failed ({exc}); "
            f"using verifier interpreter {sys.executable!r}. "
            "Set --python, activate a venv, or use poetry/uv/pdm for explicit control.",
            file=sys.stderr,
        )
        return sys.executable


def _ground_claim(rc: RawClaim, python_exe: str | None, env_error: str | None, *, check_kwarg: bool) -> dict:
    """Run introspection for one raw claim and return a fully formed claim dict."""
    locus = {
        "file": rc.locus.file,
        "line": rc.locus.line,
        "col": rc.locus.col,
    }
    if rc.locus.end_line:
        locus["end_line"] = rc.locus.end_line
    if rc.locus.end_col is not None:
        locus["end_col"] = rc.locus.end_col

    if env_error or python_exe is None:
        detail = env_error or "no Python interpreter available"
        return _claim_dict(rc, "unverifiable", 0.0, "env-error", detail, locus)

    kwarg = rc.kwarg if check_kwarg else None
    result = introspect(rc.symbol, python_exe=python_exe, kwarg=kwarg)

    verdict = result["verdict"]
    detail = result["detail"]
    confidence = 1.0 if verdict != "unverifiable" else 0.0
    kind = "introspection"

    return _claim_dict(rc, verdict, confidence, kind, detail, locus)


def _claim_dict(rc: RawClaim, verdict: str, confidence: float, kind: str, detail: str, locus: dict) -> dict:
    return {
        "id": rc.id,
        "type": "symbol",
        "subtype": rc.subtype,
        "text": rc.text,
        "locus": locus,
        "verdict": verdict,
        "confidence": confidence,
        "evidence": [{"kind": kind, "detail": detail, "determinism": "deterministic"}],
    }
