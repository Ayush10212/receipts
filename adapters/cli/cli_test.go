package cli_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Ayush10212/receipts/adapters/cli"
)

// pythonExe returns the Python interpreter used by the current environment.
func pythonExe(t *testing.T) string {
	t.Helper()
	for _, candidate := range []string{"python", "python3"} {
		if path, err := exec.LookPath(candidate); err == nil {
			return path
		}
	}
	t.Skip("no python interpreter found")
	return ""
}

const contradictedPy = `import pandas as pd
pd.DataFrame.append
`

const cleanPy = `import pandas as pd
pd.DataFrame.merge
`

func writeTempPy(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCLI_ExitCode1_OnContradicted(t *testing.T) {
	py := pythonExe(t)
	dir := t.TempDir()
	path := writeTempPy(t, dir, "bad.py", contradictedPy)

	var out, errBuf bytes.Buffer
	code := cli.Run([]string{"--python", py, "--format", "pretty", path}, &out, &errBuf)

	if code != cli.ExitPolicyFail {
		t.Errorf("expected exit 1 (policy-fail), got %d\nstdout: %s\nstderr: %s",
			code, out.String(), errBuf.String())
	}
}

func TestCLI_ExitCode0_OnClean(t *testing.T) {
	py := pythonExe(t)
	dir := t.TempDir()
	path := writeTempPy(t, dir, "good.py", cleanPy)

	var out, errBuf bytes.Buffer
	code := cli.Run([]string{"--python", py, "--format", "pretty", path}, &out, &errBuf)

	if code != cli.ExitPass {
		t.Errorf("expected exit 0 (pass), got %d\nstdout: %s\nstderr: %s",
			code, out.String(), errBuf.String())
	}
}

func TestCLI_FailOnNone_AlwaysPasses(t *testing.T) {
	py := pythonExe(t)
	dir := t.TempDir()
	path := writeTempPy(t, dir, "bad.py", contradictedPy)

	var out, errBuf bytes.Buffer
	code := cli.Run([]string{"--python", py, "--fail-on", "none", path}, &out, &errBuf)

	if code != cli.ExitPass {
		t.Errorf("expected exit 0 with --fail-on none, got %d", code)
	}
}

func TestCLI_PrettyAlwaysPrintsTargetEnv(t *testing.T) {
	py := pythonExe(t)
	dir := t.TempDir()
	path := writeTempPy(t, dir, "good.py", cleanPy)

	var out, errBuf bytes.Buffer
	cli.Run([]string{"--python", py, "--format", "pretty", path}, &out, &errBuf)

	if !strings.Contains(out.String(), "Target env:") {
		t.Errorf("pretty output must always print 'Target env:', got:\n%s", out.String())
	}
}

func TestCLI_ExplainAlwaysPrintsTargetEnv(t *testing.T) {
	py := pythonExe(t)
	dir := t.TempDir()
	path := writeTempPy(t, dir, "good.py", cleanPy)

	var out, errBuf bytes.Buffer
	cli.Run([]string{"--python", py, "--format", "json", "--explain", path}, &out, &errBuf)

	if !strings.Contains(out.String(), "Target env:") {
		t.Errorf("--explain must always print 'Target env:', got:\n%s", out.String())
	}
}

func TestCLI_JSONFormat_ValidOutput(t *testing.T) {
	py := pythonExe(t)
	dir := t.TempDir()
	path := writeTempPy(t, dir, "good.py", cleanPy)

	var out, errBuf bytes.Buffer
	cli.Run([]string{"--python", py, "--format", "json", path}, &out, &errBuf)

	if !strings.Contains(out.String(), `"schema_version"`) {
		t.Errorf("json output should contain schema_version, got:\n%s", out.String())
	}
}

func TestCLI_NoFiles_ExitToolError(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := cli.Run([]string{}, &out, &errBuf)
	if code != cli.ExitToolError {
		t.Errorf("expected exit 2 with no files, got %d", code)
	}
}
