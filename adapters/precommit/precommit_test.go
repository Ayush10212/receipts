package precommit_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Ayush10212/receipts/adapters/cli"
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

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init", dir},
		{"-C", dir, "config", "user.email", "test@example.com"},
		{"-C", dir, "config", "user.name", "Test"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

func stageFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", dir, "add", name).CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
}

func TestPrecommit_ContradictedFileFails(t *testing.T) {
	py := pythonExe(t)
	dir := t.TempDir()
	initGitRepo(t, dir)

	stageFile(t, dir, "bad.py", "import json\njson.contradicted_symbol_does_not_exist\n")

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	var out, errBuf bytes.Buffer
	code := cli.Run([]string{"--python", py, "--staged", "--fail-on", "contradicted"}, &out, &errBuf)

	if code != cli.ExitPolicyFail {
		t.Errorf("expected exit 1 for staged contradicted file, got %d\nstdout: %s\nstderr: %s",
			code, out.String(), errBuf.String())
	}
}

func TestPrecommit_CleanFilePasses(t *testing.T) {
	py := pythonExe(t)
	dir := t.TempDir()
	initGitRepo(t, dir)

	stageFile(t, dir, "good.py", "import json\njson.dumps\n")

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	var out, errBuf bytes.Buffer
	code := cli.Run([]string{"--python", py, "--staged", "--fail-on", "contradicted"}, &out, &errBuf)

	if code != cli.ExitPass {
		t.Errorf("expected exit 0 for clean staged file, got %d\nstdout: %s\nstderr: %s",
			code, out.String(), errBuf.String())
	}
}

func TestPrecommit_NoStagedPythonFiles_ToolError(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	// Stage only a non-Python file.
	stageFile(t, dir, "README.md", "# hello\n")

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	var out, errBuf bytes.Buffer
	code := cli.Run([]string{"--staged", "--fail-on", "contradicted"}, &out, &errBuf)

	// No python files staged → tool error (nothing to check).
	if code != cli.ExitToolError {
		t.Errorf("expected exit 2 with no staged Python files, got %d", code)
	}
}
