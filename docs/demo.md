# Launch demo — 90 seconds

The launch artifact is a ~90-second screen recording showing Receipts catching a
**removed method in the agent loop, before a human sees it.** This file is the shot
list and the exact commands, so the recording is reproducible.

## Setup (off camera)

Use an environment with a *modern* pandas where `DataFrame.append` no longer exists
(removed in pandas 2.0):

```bash
pip install "pandas>=2.0"
pip install -e "verifiers/python-symbols"
go build -o receipts ./cmd/receipts/
go build -o receipts-mcp ./cmd/receipts-mcp/
```

## The story (on camera)

**0:00–0:15 — The setup.** Show an agent (or a chat) generating Python that calls a
method that *looks* right but was removed:

```python
# app/etl.py — "AI-generated"
import pandas as pd

def combine(a, b):
    return a.append(b, ignore_index=True)   # removed in pandas 2.0
```

**0:15–0:45 — The catch.** Run Receipts. It flags `contradicted` with version-cited
evidence, and exits non-zero:

```bash
./receipts check --python "$(which python)" app/etl.py
```

Expected output:

```
Receipts 0.1.0  run=run-…
Target env: python 3.11.4 (/…/.venv)
Summary: grounded=1  contradicted=1  unverifiable=0
Decision: fail
  [contradicted] pandas.DataFrame.append  app/etl.py:5
    evidence: 'pandas.DataFrame.append' not found (AttributeError on 'append')
```

> Note: write the demo line as `pd.DataFrame.append(a, b)` if you want the receiver
> resolved statically with certainty. A method call on a local variable
> (`a.append(...)`) whose type can't be traced is reported `unverifiable` by design —
> that's the no-false-positives rule in action, and it's worth showing too.

**0:45–1:15 — In the agent loop (MCP).** Show the same check happening *inside* the
agent via the `receipts_check_code` MCP tool: the agent generates the snippet, calls
the tool, sees `contradicted`, and revises to `merge`/`concat` **before** surfacing
anything to the human. The human never sees the broken code.

**1:15–1:30 — Plain-English note (optional).** Re-run with `--llm` to show the same
deterministic verdict, now with a one-line human-readable note tagged `subjective`:

```bash
export MISTRAL_API_KEY=…
./receipts check --llm --python "$(which python)" app/etl.py
```

The verdict and counts are byte-identical to the non-LLM run — only an advisory
`llm-note` is added. That's the honesty guarantee, on screen.

## Done when

The recording shows a real `contradicted` catch end-to-end in the agent loop, with
version-cited evidence, before the human sees the code. Link the final video from the
top of [README.md](../README.md).
