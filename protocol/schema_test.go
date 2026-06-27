package protocol_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func loadSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Dir(file)
	schemaPath := filepath.Join(dir, "report.v0.json")

	data, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schemaDoc any
	if err := json.Unmarshal(data, &schemaDoc); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}

	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)

	if err := c.AddResource("report.v0.json", schemaDoc); err != nil {
		t.Fatalf("add schema resource: %v", err)
	}
	sch, err := c.Compile("report.v0.json")
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}
	return sch
}

func validateFile(t *testing.T, sch *jsonschema.Schema, path string) error {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return sch.Validate(v)
}

func testdataPath(name string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata", name)
}

func TestReportSchema_ValidReports(t *testing.T) {
	sch := loadSchema(t)

	cases := []string{
		"report_grounded.json",
		"report_contradicted.json",
		"report_unverifiable.json",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			if err := validateFile(t, sch, testdataPath(name)); err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
		})
	}
}

func TestReportSchema_MalformedReportRejected(t *testing.T) {
	sch := loadSchema(t)
	err := validateFile(t, sch, testdataPath("report_malformed.json"))
	if err == nil {
		t.Fatal("expected malformed report to be rejected, but it passed validation")
	}
}
