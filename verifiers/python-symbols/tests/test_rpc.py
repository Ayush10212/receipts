"""Test the JSON-RPC loop over pipes (Step 3.1 done-when)."""

import io
import json

import pytest

from receipts_python_symbols.rpc import run_loop


def _exchange(requests: list[dict]) -> list[dict]:
    """Drive the RPC loop with a sequence of requests, collect responses."""
    lines = "\n".join(json.dumps(r) for r in requests) + "\n"
    stdin = io.StringIO(lines)
    stdout = io.StringIO()
    run_loop(stdin, stdout)
    stdout.seek(0)
    return [json.loads(l) for l in stdout if l.strip()]


def test_initialize():
    responses = _exchange([
        {"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}},
        {"jsonrpc": "2.0", "id": 2, "method": "shutdown", "params": {}},
    ])
    init_resp = responses[0]
    assert init_resp["result"]["name"] == "python-symbols"
    assert init_resp["result"]["determinism"] == "deterministic"
    assert "python" in init_resp["result"]["capabilities"]


def test_shutdown_stops_loop():
    responses = _exchange([
        {"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}},
        {"jsonrpc": "2.0", "id": 2, "method": "shutdown", "params": {}},
        # This should never be processed:
        {"jsonrpc": "2.0", "id": 3, "method": "initialize", "params": {}},
    ])
    ids = [r["id"] for r in responses]
    assert 3 not in ids, "request after shutdown should not be processed"


def test_unknown_method_returns_error():
    responses = _exchange([
        {"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}},
        {"jsonrpc": "2.0", "id": 2, "method": "nonexistent", "params": {}},
        {"jsonrpc": "2.0", "id": 3, "method": "shutdown", "params": {}},
    ])
    err_resp = next(r for r in responses if r["id"] == 2)
    assert "error" in err_resp
    assert err_resp["error"]["code"] == -32601


def test_analyze_returns_claims():
    responses = _exchange([
        {"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}},
        {"jsonrpc": "2.0", "id": 2, "method": "analyze", "params": {
            "artifact": {"path": "test.py", "content": "import pandas as pd\n"},
            "context": {"workdir": ".", "language": "python",
                        "target_env": {"language": "python", "version": "3.11", "prefix": ""}},
        }},
        {"jsonrpc": "2.0", "id": 3, "method": "shutdown", "params": {}},
    ])
    analyze_resp = next(r for r in responses if r["id"] == 2)
    assert "result" in analyze_resp
    assert "claims" in analyze_resp["result"]
