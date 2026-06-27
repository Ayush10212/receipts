package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/Ayush10212/receipts/core/config"
	"github.com/Ayush10212/receipts/core/engine"
	"github.com/Ayush10212/receipts/core/policy"
	"github.com/Ayush10212/receipts/core/report"
	"github.com/Ayush10212/receipts/core/verifier"
)

// NewServer creates an MCP server exposing the receipts_check_code tool.
func NewServer() *server.MCPServer {
	s := server.NewMCPServer(
		"receipts",
		"0.1.0",
		server.WithToolCapabilities(false),
	)

	tool := mcpgo.NewTool("receipts_check_code",
		mcpgo.WithDescription(
			"Statically verifies Python code against installed packages. "+
				"Call this before proposing generated code to the user. Returns two parts: "+
				"(1) a plain-English review meant to be shown to the human as-is, and "+
				"(2) the full machine-readable Receipts Report with "+
				"grounded/contradicted/unverifiable verdicts. If anything is 'contradicted', "+
				"revise the code before surfacing it — the human should never see code that "+
				"calls something which does not exist.",
		),
		mcpgo.WithString("code",
			mcpgo.Required(),
			mcpgo.Description("Python source code to verify"),
		),
		mcpgo.WithString("language",
			mcpgo.Description("Programming language (currently only 'python')"),
			mcpgo.DefaultString("python"),
		),
		mcpgo.WithString("workdir",
			mcpgo.Description("Working directory for env resolution"),
			mcpgo.DefaultString("."),
		),
		mcpgo.WithString("python",
			mcpgo.Description("Path to Python interpreter (optional; uses env detection if omitted)"),
		),
	)

	s.AddTool(tool, HandleCheckCode)
	return s
}

// HandleCheckCode is the MCP tool handler for receipts_check_code.
func HandleCheckCode(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args, _ := req.Params.Arguments.(map[string]any)
	if args == nil {
		args = map[string]any{}
	}
	code, _ := args["code"].(string)
	language, _ := args["language"].(string)
	workdir, _ := args["workdir"].(string)
	pythonPath, _ := args["python"].(string)

	if language == "" {
		language = "python"
	}
	if workdir == "" {
		workdir = "."
	}
	if code == "" {
		return mcpgo.NewToolResultError("code is required"), nil
	}

	cfg, _ := config.FileProvider{}.Load(workdir)

	backends := engine.Backends{
		Policy:    policy.LocalBackend{},
		NewClient: mcpClientFactory(pythonPath),
	}

	artifacts := []engine.Artifact{
		{Path: "mcp_input.py", Content: []byte(code), Language: language},
	}

	r, err := engine.Run(ctx, report.ExecutionContext{}, artifacts, cfg, backends)
	if err != nil {
		return mcpgo.NewToolResultError(fmt.Sprintf("engine error: %v", err)), nil
	}

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return mcpgo.NewToolResultError(fmt.Sprintf("marshal error: %v", err)), nil
	}

	// Two content blocks, two audiences. Block 0 is the plain-English review the
	// agent should show the human before acting. Block 1 is the full Report JSON
	// for tooling. Both describe the same frozen verdicts — the prose adds no
	// judgement of its own.
	return &mcpgo.CallToolResult{
		Content: []mcpgo.Content{
			mcpgo.NewTextContent(report.Explain(r)),
			mcpgo.NewTextContent(string(data)),
		},
	}, nil
}

func mcpClientFactory(pythonPath string) verifier.ClientFactory {
	return func(ctx context.Context, _ report.ExecutionContext, _ string) (*verifier.Client, error) {
		exe := "python"
		if pythonPath != "" {
			exe = pythonPath
		} else if p, err := exec.LookPath("python"); err == nil {
			exe = p
		}
		return verifier.NewClient(ctx, 30*time.Second, exe, "-m", "receipts_python_symbols")
	}
}

// ServeStdio starts the MCP server on stdin/stdout.
func ServeStdio(ctx context.Context) error {
	s := NewServer()
	return server.ServeStdio(s)
}
