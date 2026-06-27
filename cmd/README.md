# `cmd/` — the binaries

Entry points. Each subfolder builds one executable. They're deliberately tiny — they just wire
up an adapter from [`../adapters/`](../adapters/) and hand off.

| Binary | Build | What it is |
|---|---|---|
| `receipts` | `go build -o receipts ./cmd/receipts/` | The command-line tool (`receipts check ...`). |
| `receipts-mcp` | `go build -o receipts-mcp ./cmd/receipts-mcp/` | The MCP server an AI client connects to. |

If you just want to use the tool, build `receipts` and see the [top-level README](../README.md).
