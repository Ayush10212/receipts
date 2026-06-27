package verifier_test

import (
	"context"
	"testing"
	"time"

	"github.com/Ayush10212/receipts/core/report"
	"github.com/Ayush10212/receipts/core/verifier"
)

// echoVerifier is a tiny Python script that speaks the JSON-RPC protocol correctly.
const echoVerifierScript = `
import sys, json

def respond(id_, result):
    print(json.dumps({"jsonrpc":"2.0","id":id_,"result":result}), flush=True)

for line in sys.stdin:
    req = json.loads(line)
    method = req["method"]
    id_ = req["id"]
    if method == "initialize":
        respond(id_, {"name":"echo","version":"0.1.0","capabilities":["python"],"determinism":"deterministic"})
    elif method == "analyze":
        respond(id_, {"claims":[{
            "id":"c-1","type":"symbol","subtype":"attribute",
            "text":"pandas.DataFrame.append",
            "locus":{"file":"x.py","line":1,"col":0},
            "verdict":"contradicted","confidence":1.0,
            "evidence":[{"kind":"introspection","detail":"removed","determinism":"deterministic"}]
        }]})
    elif method == "shutdown":
        respond(id_, {})
        break
`

func startEchoVerifier(t *testing.T) *verifier.Client {
	t.Helper()
	ctx := context.Background()
	c, err := verifier.NewClient(ctx, 5*time.Second, "python", "-c", echoVerifierScript)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestClient_Initialize(t *testing.T) {
	c := startEchoVerifier(t)
	defer c.Close()

	if c.Info.Name != "echo" {
		t.Errorf("name: got %q want echo", c.Info.Name)
	}
	if c.Info.Determinism != "deterministic" {
		t.Errorf("determinism: got %q", c.Info.Determinism)
	}
}

func TestClient_Analyze(t *testing.T) {
	c := startEchoVerifier(t)
	defer c.Shutdown(context.Background())

	claims, err := c.Analyze(context.Background(),
		verifier.Artifact{Path: "x.py", Content: "import pandas"},
		verifier.AnalyzeContext{Workdir: ".", Language: "python",
			TargetEnv: report.TargetEnv{Language: "python", Version: "3.11", Prefix: "/venv"}},
	)
	if err != nil {
		t.Fatalf("Analyze returned unexpected error (should degrade, not error): %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("expected 1 claim, got %d", len(claims))
	}
	if claims[0].Verdict != report.VerdictContradicted {
		t.Errorf("verdict: got %q want contradicted", claims[0].Verdict)
	}
}

func TestClient_TimeoutDegrades(t *testing.T) {
	// Verifier that hangs forever on analyze.
	hangScript := `
import sys, json
for line in sys.stdin:
    req = json.loads(line)
    if req["method"] == "initialize":
        print(json.dumps({"jsonrpc":"2.0","id":req["id"],"result":{"name":"hang","version":"0.1.0","capabilities":[],"determinism":"deterministic"}}), flush=True)
    # Never responds to analyze — simulates timeout.
`
	c, err := verifier.NewClient(context.Background(), 5*time.Second, "python", "-c", hangScript)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	// Use a very short timeout so the test is fast.
	shortCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	claims, err := c.Analyze(shortCtx,
		verifier.Artifact{Path: "hang.py"},
		verifier.AnalyzeContext{Workdir: ".", Language: "python",
			TargetEnv: report.TargetEnv{Language: "python", Version: "3.11", Prefix: "/venv"}},
	)
	// Analyze must NOT return an error — it degrades gracefully to unverifiable.
	if err != nil {
		t.Fatalf("expected graceful degradation, got error: %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("expected 1 degraded claim, got %d", len(claims))
	}
	if claims[0].Verdict != report.VerdictUnverifiable {
		t.Errorf("timeout should produce unverifiable, got %q", claims[0].Verdict)
	}
	if !verifier.IsToolError(claims[0]) {
		t.Error("timeout claim should be marked as tool-error type")
	}
}

func TestClient_CrashDegrades(t *testing.T) {
	// Verifier that crashes after initialize.
	crashScript := `
import sys, json
for line in sys.stdin:
    req = json.loads(line)
    if req["method"] == "initialize":
        print(json.dumps({"jsonrpc":"2.0","id":req["id"],"result":{"name":"crash","version":"0.1.0","capabilities":[],"determinism":"deterministic"}}), flush=True)
    elif req["method"] == "analyze":
        sys.exit(1)
`
	c, err := verifier.NewClient(context.Background(), 5*time.Second, "python", "-c", crashScript)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	claims, err := c.Analyze(context.Background(),
		verifier.Artifact{Path: "crash.py"},
		verifier.AnalyzeContext{Workdir: ".", Language: "python",
			TargetEnv: report.TargetEnv{Language: "python", Version: "3.11", Prefix: "/venv"}},
	)
	if err != nil {
		t.Fatalf("crash should degrade gracefully, got error: %v", err)
	}
	if len(claims) != 1 || claims[0].Verdict != report.VerdictUnverifiable {
		t.Errorf("crash should yield unverifiable degraded claim, got %+v", claims)
	}
}
