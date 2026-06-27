// Command receipts-mcp runs the Receipts MCP server on stdin/stdout, exposing the
// receipts_check_code tool so an agent can statically verify generated Python before
// proposing it. It is a thin shell — all logic lives in adapters/mcp.
package main

import (
	"context"
	"fmt"
	"os"

	mcpadapter "github.com/Ayush10212/receipts/adapters/mcp"
)

func main() {
	if err := mcpadapter.ServeStdio(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "receipts-mcp: %v\n", err)
		os.Exit(1)
	}
}
