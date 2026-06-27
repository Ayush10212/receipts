# Use Receipts in your AI client

Receipts is designed to sit between the AI and you, fact-checking generated Python before
you ever see it. **Two layers work in every client**, and most popular clients also get a
native shortcut that's already committed to this repo — so a fresh `git clone` just works.

## One-time setup (any client)

```bash
# install the Python checker into the environment whose packages you want to verify
pip install -e "verifiers/python-symbols"

# (optional) pre-build the binaries; the MCP configs below use `go run`, so this is optional
go build -o receipts ./cmd/receipts/
go build -o receipts-mcp ./cmd/receipts-mcp/
```

You need **Go 1.26+** and **Python 3.10+** on your PATH.

## The two universal layers

| Layer | Works in | How |
|---|---|---|
| **CLI** | *Everything* that can run a terminal command | `./receipts check --format plain --python python file.py` |
| **MCP server** | Any MCP-capable client | tool `receipts_check_code` → returns a plain-English review + full report |

## Per-client integration (already in this repo)

| Client | What's committed | What you get |
|---|---|---|
| **Claude Code** | [`.claude/commands/receipts.md`](../.claude/commands/receipts.md), [`.mcp.json`](../.mcp.json), [`CLAUDE.md`](../CLAUDE.md) | `/receipts` command + auto-MCP |
| **Gemini CLI** | [`.gemini/commands/receipts.toml`](../.gemini/commands/receipts.toml), [`.gemini/settings.json`](../.gemini/settings.json), [`GEMINI.md`](../GEMINI.md) | `/receipts` command + MCP |
| **Cursor** | [`.cursor/rules/receipts.mdc`](../.cursor/rules/receipts.mdc), [`.cursor/mcp.json`](../.cursor/mcp.json) | always-on rule + MCP |
| **VS Code + Copilot** | [`.github/copilot-instructions.md`](../.github/copilot-instructions.md), [`.vscode/mcp.json`](../.vscode/mcp.json) | auto guidance + MCP |
| **Codex** | [`AGENTS.md`](../AGENTS.md) | run-it instructions (Codex reads `AGENTS.md`) |
| **Any other agent** | [`AGENTS.md`](../AGENTS.md) | the emerging cross-tool standard — most agents read it |

### Notes per client

- **Claude Code** — opening the cloned folder offers the project MCP server (`.mcp.json`); approve it once. `/receipts file.py` runs the check and returns the plain-English review.
- **Gemini CLI** — custom commands load from `.gemini/commands/`, so `/receipts` is available after clone. MCP servers load from `.gemini/settings.json`.
- **Cursor** — the rule in `.cursor/rules/` applies automatically; the MCP server in `.cursor/mcp.json` appears under Settings → MCP.
- **VS Code (Copilot)** — `.github/copilot-instructions.md` is picked up automatically; `.vscode/mcp.json` registers the server for Agent mode.
- **Codex** — reads `AGENTS.md` for how to run the check. Use the CLI or wire the MCP server via your Codex config.

## If a client isn't listed

Use the **CLI** (works literally everywhere) or point the client's MCP config at:

```json
{ "command": "go", "args": ["run", "./cmd/receipts-mcp"] }
```

run from the repo root. That's all any MCP client needs.
