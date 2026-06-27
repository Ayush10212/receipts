# `examples/` — try it in 10 seconds

Two tiny files so you can see both verdicts immediately.

| File | What it does | Expected result |
|---|---|---|
| `bad.py` | Calls `pd.DataFrame.append`, which was **removed in pandas 2.0**. | ❌ `contradicted` — exit code `1` |
| `good.py` | Uses `pd.concat`, the correct modern API. | ✅ `grounded` — exit code `0` |

```bash
# from the repo root, after building (see the main README)
./receipts check --format plain --python python examples/bad.py
./receipts check --format plain --python python examples/good.py
```

(On Windows: `.\receipts.exe ... --python python`.)

`.receipts.yaml` here shows the optional config format — note `enabled: false`, and that it
contains **no API keys** (those only ever come from environment variables).
