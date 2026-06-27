"""
Sandboxed introspection: run in a worker subprocess, never in the main process.
Imports INSTALLED packages only — never the code-under-test.
"""

import json
import subprocess
import sys
from typing import Any

# ── Worker code (runs inside the sandboxed subprocess) ───────────────────────

_WORKER = """
import json, sys, inspect

def run(req):
    symbol   = req["symbol"]
    kwarg    = req.get("kwarg")
    parts    = symbol.split(".")
    mod_name = parts[0]

    # Import the top-level module.
    try:
        import importlib
        obj = importlib.import_module(mod_name)
    except ImportError as e:
        # A missing or uninstallable module is always unverifiable, never contradicted.
        return {"verdict": "unverifiable", "detail": f"ImportError: {e}"}

    # Walk attribute chain.
    for attr in parts[1:]:
        try:
            obj = getattr(obj, attr)
        except AttributeError:
            return {"verdict": "contradicted", "detail": f"{symbol!r} not found (AttributeError on {attr!r})"}

    # kwarg check.
    if kwarg:
        try:
            sig = inspect.signature(obj)
        except (ValueError, TypeError):
            # C-extension or decorated signature — cannot introspect.
            return {"verdict": "unverifiable", "detail": f"cannot introspect signature of {symbol!r} (C-ext or opaque)"}
        params = sig.parameters
        if any(p.kind == inspect.Parameter.VAR_KEYWORD for p in params.values()):
            return {"verdict": "unverifiable", "detail": f"{symbol!r} accepts **kwargs — cannot verify kwarg {kwarg!r}"}
        if kwarg not in params:
            return {"verdict": "contradicted", "detail": f"{symbol!r} has no kwarg {kwarg!r}"}
        return {"verdict": "grounded", "detail": f"{symbol!r}({kwarg}=...) exists"}

    # Attribute-only check: verify the final object is not a C-extension type that
    # we cannot inspect further (we only flag if getattr succeeded → grounded).
    try:
        if callable(obj):
            inspect.signature(obj)   # probe — if it raises, still grounded (attr exists)
    except (ValueError, TypeError):
        pass   # C-ext callable — attr exists but sig opaque → still grounded

    return {"verdict": "grounded", "detail": f"{symbol!r} exists"}

req = json.loads(sys.stdin.read())
try:
    result = run(req)
except Exception as e:
    result = {"verdict": "unverifiable", "detail": f"worker exception: {e}"}
print(json.dumps(result))
"""

# ── Cache ─────────────────────────────────────────────────────────────────────

_CACHE: dict[str, dict] = {}   # in-process cache; disk cache wired via analyze.py


def _cache_key(symbol: str, kwarg: str | None, python_exe: str) -> str:
    return f"{python_exe}\x00{symbol}\x00{kwarg or ''}"


# ── Public API ────────────────────────────────────────────────────────────────

def introspect(
    symbol: str,
    *,
    python_exe: str,
    kwarg: str | None = None,
    timeout: float = 10.0,
) -> dict[str, str]:
    """
    Introspect `symbol` (dotted path, e.g. "pandas.DataFrame.append") in a
    sandboxed subprocess running `python_exe`.

    Returns {"verdict": "grounded"|"contradicted"|"unverifiable", "detail": str}.
    A timeout or crash on one symbol → unverifiable for that symbol only.
    """
    key = _cache_key(symbol, kwarg, python_exe)
    if key in _CACHE:
        return _CACHE[key]

    req = json.dumps({"symbol": symbol, "kwarg": kwarg})

    try:
        proc = subprocess.run(
            [python_exe, "-c", _WORKER],
            input=req,
            capture_output=True,
            text=True,
            timeout=timeout,
        )
    except subprocess.TimeoutExpired:
        result = {"verdict": "unverifiable", "detail": f"introspection timed out after {timeout}s for {symbol!r}"}
        _CACHE[key] = result
        return result
    except Exception as exc:
        result = {"verdict": "unverifiable", "detail": f"subprocess error: {exc}"}
        _CACHE[key] = result
        return result

    if proc.returncode != 0:
        result = {"verdict": "unverifiable", "detail": f"worker crashed (exit {proc.returncode}): {proc.stderr.strip()}"}
        _CACHE[key] = result
        return result

    try:
        result = json.loads(proc.stdout.strip())
    except (json.JSONDecodeError, ValueError) as exc:
        result = {"verdict": "unverifiable", "detail": f"worker produced invalid JSON: {exc}"}

    _CACHE[key] = result
    return result
