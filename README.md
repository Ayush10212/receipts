<div align="center">

# 🧾 Receipts

### Catch the code your AI made up — *before* you run it.

AI writes Python that **looks** perfect but calls functions that don't exist —
methods removed in a version bump, arguments that were never there, names it
simply hallucinated. Receipts checks every symbol against the packages you
*actually have installed*, and tells you in plain English what's real and what's a lie.

[![CI](https://github.com/Ayush10212/receipts/actions/workflows/ci.yml/badge.svg)](https://github.com/Ayush10212/receipts/actions/workflows/ci.yml)
![Go](https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go&logoColor=white)
![Python](https://img.shields.io/badge/Python-3.10%2B-3776AB?logo=python&logoColor=white)
![License](https://img.shields.io/badge/license-Apache--2.0-blue)
![Catch rate](https://img.shields.io/badge/catch%20rate-100%25-success)
![False positives](https://img.shields.io/badge/false%20positives-0%25-success)

</div>

---

## The problem in one screenshot

An AI assistant hands you this. It looks fine. It is not fine:

```python
import pandas as pd
result = pd.DataFrame.append(df1, df2, ignore_index=True)   # removed in pandas 2.0
```

Run Receipts and you get a verdict a non-developer can act on:

```text
Receipts — plain-English review
================================

File checked:    app.py
Checked against: your installed python packages

What I found:
  OK      4  - confirmed real and correct
  WRONG   1  - uses something that does NOT exist
  ?       0  - couldn't be checked (not necessarily wrong)

Details:

  [WRONG] pandas.DataFrame.append   (line 2)
         This uses a name that isn't in the installed package - it was likely
         removed or renamed. Running it would crash with an AttributeError.
         technical detail: 'pandas.DataFrame.append' not found

Bottom line: DO NOT use this code yet. It calls something that does not exist
and will crash. Fix the WRONG items above first.
```

No guessing. No running the code. Just the truth, checked against reality.

> 🆕 **New here? Read [docs/plain-english.md](docs/plain-english.md)** — the whole tool
> explained with zero jargon.

---

## Quick start

```bash
# 1. Clone
git clone https://github.com/Ayush10212/receipts.git
cd receipts

# 2. Install the Python checker into the same environment whose packages
#    you want to verify against (your project's venv, or your global python)
pip install -e "verifiers/python-symbols"

# 3. Build the tool
go build -o receipts ./cmd/receipts/

# 4. Try it on the included broken example
./receipts check --format plain --python python examples/bad.py     # ❌ catches it
./receipts check --format plain --python python examples/good.py    # ✅ all clear
```

On Windows, use `.\receipts.exe` and `--python python`.

---

## The three answers it gives

Every function, attribute, and keyword argument gets exactly one verdict:

| Verdict | Friendly name | Meaning |
|---|:---:|---|
| `grounded` | ✅ **OK** | Confirmed present in your installed package. |
| `contradicted` | ❌ **WRONG** | Provably does **not** exist — removed, renamed, or hallucinated. |
| `unverifiable` | ⚠️ **?** | Couldn't be checked (package not installed, C-extension, ambiguity). **Not** treated as wrong. |

**It is deliberately cautious.** If it isn't *certain* something is wrong, it says
`unverifiable` — never a false `contradicted`. A tool that cries wolf gets ignored, so
the false-positive rate is a hard gate (**< 10%**, enforced in CI — currently **0%**).

---

## Use it as a layer between the AI and you

This is the point. Receipts is most powerful sitting *inside* your AI workflow so every
piece of generated code is fact-checked automatically.

**Works in every AI client out of the box.** A fresh `git clone` ships native integration
for the major assistants — see **[docs/ai-clients.md](docs/ai-clients.md)** for the full
list:

| Client | What you get on clone |
|---|---|
| **Claude Code** | `/receipts` command + auto-MCP (`.claude/`, `.mcp.json`) |
| **Gemini CLI** | `/receipts` command + MCP (`.gemini/`) |
| **Cursor** | always-on rule + MCP (`.cursor/`) |
| **VS Code + Copilot** | auto instructions + MCP (`.github/`, `.vscode/`) |
| **Codex / any agent** | run-it guidance via `AGENTS.md` |

In Claude Code, for example: type `/receipts yourfile.py` (or just `/receipts` to check your
current git diff) and your assistant runs the check and hands back the plain-English review.
The **CLI** and **MCP server** work in *every* client regardless.

**MCP server** — Receipts ships an [MCP](https://modelcontextprotocol.io) server with a
`receipts_check_code` tool. An AI calls it *before* showing you code; it returns **two
parts** — a plain-English review for you and the full machine report for tooling. If
anything is `contradicted`, a well-behaved assistant fixes the code first. You only ever
see code that's real.

```bash
go build -o receipts-mcp ./cmd/receipts-mcp/
```

**Pre-commit hook** — block hallucinated code from ever landing in git:

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/Ayush10212/receipts
    rev: v0.1.0
    hooks:
      - id: receipts-check
```

---

## Output formats

```bash
./receipts check --format plain   app.py   # plain-English review (best for humans)
./receipts check --format pretty  app.py   # compact technical summary (default)
./receipts check --format json    app.py   # the full Report (machine-readable)
./receipts check --format sarif   app.py   # SARIF 2.1.0 for code-scanning UIs
```

**Exit codes:** `0` pass · `1` something is `contradicted` · `2` tool error.
Perfect for CI gates and git hooks.

---

## How it works

The Go core knows **zero Python**. All language knowledge lives behind a JSON-RPC 2.0
boundary, so the engine never has to understand any one language:

```
receipts check [files]
        │
   core/engine                pure Go: (files, config) → Report → policy → output
        │  JSON-RPC 2.0 over stdio
   verifiers/python-symbols   pure Python: AST extract + sandboxed introspection
```

The Python checker reads your file with `ast.parse` (**it is never imported or run**),
then introspects each symbol in a **separate sandboxed subprocess** that imports only the
*installed* dependency and inspects it with `getattr` / `inspect.signature`.

Three guarantees that never break:

1. **It never executes your code.** Static analysis only.
2. **Verdicts are deterministic.** Same code + same environment → same answer, every time.
3. **The optional LLM layer is advisory only.** It may *describe* a verdict in plain
   English; it can never create, change, or override one.

Want the deep dive? Each folder has its own README — start with [`core/`](core/) and
[`verifiers/`](verifiers/), or read [CLAUDE.md](CLAUDE.md) for the full architecture.

---

## Project layout

| Folder | What's in it |
|---|---|
| [`core/`](core/) | The pure-Go engine: orchestration, report assembly, policy, output, cache. |
| [`verifiers/`](verifiers/) | Language checkers. `python-symbols/` is the reference implementation. |
| [`adapters/`](adapters/) | The surfaces: CLI, MCP server, pre-commit hook. |
| [`llm/`](llm/) | Optional advisory annotations (Mistral / Grok). Off by default. |
| [`protocol/`](protocol/) | Versioned public JSON schemas (the Report and the plugin protocol). |
| [`benchmark/`](benchmark/) | 50-case corpus + quality gates (catch rate, false-positive rate). |
| [`cmd/`](cmd/) | Entry points for the `receipts` and `receipts-mcp` binaries. |
| [`docs/`](docs/) | Plain-English guide, report format, protocol docs, demo script. |
| [`examples/`](examples/) | A broken file and a correct one to try the tool on. |

---

## License & contributions

Released under the [Apache-2.0](LICENSE) license — free to use, clone, and build on.

This is a personal project and is **not accepting external pull requests**. You're very
welcome to ⭐ star it, fork it, and use it however you like.
