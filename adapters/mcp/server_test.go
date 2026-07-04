package mcp_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/santhosh-tekuri/jsonschema/v6"

	mcpadapter "github.com/Ayush10212/receipts/adapters/mcp"
)

func pythonExe(t *testing.T) string {
	t.Helper()
	for _, c := range []string{"python", "python3"} {
		if p, err := exec.LookPath(c); err == nil {
			return p
		}
	}
	t.Skip("no python interpreter found")
	return ""
}

func loadSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	schemaPath := filepath.Join(filepath.Dir(file), "..", "..", "protocol", "report.v0.json")
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var doc any
	json.Unmarshal(data, &doc)
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	c.AddResource("report.v0.json", doc)
	sch, err := c.Compile("report.v0.json")
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}
	return sch
}

func callHandler(t *testing.T, args map[string]any) *mcpgo.CallToolResult {
	t.Helper()
	req := mcpgo.CallToolRequest{}
	req.Params.Name = "receipts_check_code"
	req.Params.Arguments = args
	result, err := mcpadapter.HandleCheckCode(context.Background(), req)
	if err != nil {
		t.Fatalf("HandleCheckCode: %v", err)
	}
	return result
}

func TestMCPServer_ReturnsSchemaValidReport(t *testing.T) {
	py := pythonExe(t)
	sch := loadSchema(t)

	result := callHandler(t, map[string]any{
		"code":    "import json\njson.dumps\n",
		"python":  py,
		"workdir": ".",
	})

	if result.IsError {
		t.Fatalf("unexpected tool error: %v", result.Content)
	}

	// Block 0 is the plain-English review for the human; block 1 is the JSON Report.
	if got := len(result.Content); got != 2 {
		t.Fatalf("expected 2 content blocks (review + JSON), got %d", got)
	}
	review := result.Content[0].(mcpgo.TextContent)
	if !strings.Contains(review.Text, "plain-English review") {
		t.Errorf("first block should be the plain-English review, got:\n%s", review.Text)
	}

	jsonText := reportJSON(t, result)
	var v any
	if err := json.Unmarshal([]byte(jsonText), &v); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, jsonText)
	}
	if err := sch.Validate(v); err != nil {
		t.Fatalf("report failed schema validation: %v\n%s", err, jsonText)
	}
}

// reportJSON returns the machine-readable Report block (the last content block).
func reportJSON(t *testing.T, result *mcpgo.CallToolResult) string {
	t.Helper()
	last := result.Content[len(result.Content)-1].(mcpgo.TextContent)
	return last.Text
}

func TestMCPServer_ContradictedSymbol_ReportedCorrectly(t *testing.T) {
	py := pythonExe(t)

	result := callHandler(t, map[string]any{
		"code":    "import json\njson.contradicted_symbol_does_not_exist\n",
		"python":  py,
		"workdir": ".",
	})

	if result.IsError {
		t.Fatalf("unexpected tool error: %v", result.Content)
	}
	var r map[string]any
	if err := json.Unmarshal([]byte(reportJSON(t, result)), &r); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	summary := r["summary"].(map[string]any)
	if summary["contradicted"].(float64) < 1 {
		t.Errorf("expected >=1 contradicted in summary, got: %v", summary)
	}
}

func TestMCPServer_EmptyCode_ReturnsError(t *testing.T) {
	result := callHandler(t, map[string]any{"code": ""})
	if !result.IsError {
		t.Error("expected IsError=true for empty code")
	}
}
