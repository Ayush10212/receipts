package verifier

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync/atomic"
	"time"

	"github.com/Ayush10212/receipts/core/report"
)

// Artifact is a file to be analyzed.
type Artifact struct {
	Path    string `json:"path"`
	Content string `json:"content,omitempty"`
}

// AnalyzeContext carries runtime context to the verifier plugin.
type AnalyzeContext struct {
	Workdir   string            `json:"workdir"`
	Language  string            `json:"language"`
	TargetEnv report.TargetEnv  `json:"target_env"`
}

// PluginInfo is the response from the initialize RPC call.
type PluginInfo struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities"`
	Determinism  string   `json:"determinism"`
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// errTainted is returned by call() when a previous timeout left the decoder in an
// unknown state.  The client must be discarded (Shutdown kills the subprocess directly).
var errTainted = errors.New("verifier: client tainted by previous call timeout")

// Client manages one verifier subprocess.
type Client struct {
	cmd    *exec.Cmd
	enc    *json.Encoder
	dec    *json.Decoder
	nextID atomic.Int64
	// per-call timeout
	callTimeout time.Duration
	// tainted is set after a call timeout to prevent concurrent decoder access.
	tainted atomic.Bool
	Info    PluginInfo
}

// NewClient spawns the verifier subprocess and runs initialize.
// callTimeout applies to each individual RPC call.
func NewClient(ctx context.Context, callTimeout time.Duration, cmdArgs ...string) (*Client, error) {
	if len(cmdArgs) == 0 {
		return nil, fmt.Errorf("verifier: no command specified")
	}
	cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("verifier: stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("verifier: stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("verifier: start: %w", err)
	}

	c := &Client{
		cmd:         cmd,
		enc:         json.NewEncoder(stdin),
		dec:         json.NewDecoder(bufio.NewReader(stdout)),
		callTimeout: callTimeout,
	}

	var info PluginInfo
	if err := c.call(ctx, "initialize", struct{}{}, &info); err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("verifier: initialize: %w", err)
	}
	c.Info = info
	return c, nil
}

func (c *Client) call(ctx context.Context, method string, params any, result any) error {
	if c.tainted.Load() {
		return errTainted
	}

	id := c.nextID.Add(1)

	callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
	defer cancel()

	req := rpcRequest{JSONRPC: "2.0", ID: id, Method: method, Params: params}

	// Use a channel to propagate encode/decode errors with context cancellation.
	errc := make(chan error, 1)
	go func() {
		if err := c.enc.Encode(req); err != nil {
			errc <- fmt.Errorf("encode: %w", err)
			return
		}
		var resp rpcResponse
		if err := c.dec.Decode(&resp); err != nil {
			errc <- fmt.Errorf("decode: %w", err)
			return
		}
		if resp.Error != nil {
			errc <- fmt.Errorf("rpc error %d: %s", resp.Error.Code, resp.Error.Message)
			return
		}
		if result != nil {
			errc <- json.Unmarshal(resp.Result, result)
			return
		}
		errc <- nil
	}()

	select {
	case err := <-errc:
		return err
	case <-callCtx.Done():
		// Mark the client tainted: the goroutine above is still reading from c.dec.
		// Any future call would race on the decoder.  Shutdown must kill the process directly.
		c.tainted.Store(true)
		go func() { <-errc }() // drain the goroutine so it doesn't leak
		return fmt.Errorf("verifier call %q timed out after %v", method, c.callTimeout)
	}
}

type analyzeParams struct {
	Artifact Artifact       `json:"artifact"`
	Context  AnalyzeContext `json:"context"`
}

type analyzeResult struct {
	Claims []report.Claim `json:"claims"`
}

// Analyze sends an artifact to the verifier and returns claims.
// On any RPC error the file's claims degrade to a single tool-error claim (never a false verdict).
func (c *Client) Analyze(ctx context.Context, artifact Artifact, actx AnalyzeContext) ([]report.Claim, error) {
	params := analyzeParams{Artifact: artifact, Context: actx}
	var result analyzeResult
	if err := c.call(ctx, "analyze", params, &result); err != nil {
		// Graceful degradation: surface as tool error, not a verdict.
		return []report.Claim{
			{
				ID:         "tool-error-" + artifact.Path,
				Type:       "tool-error",
				Subtype:    report.SubtypeImport,
				Text:       fmt.Sprintf("verifier error on %s: %v", artifact.Path, err),
				Locus:      report.Locus{File: artifact.Path, Line: 1, Col: 0},
				Verdict:    report.VerdictUnverifiable,
				Confidence: 0,
				Evidence:   []report.Evidence{{Kind: "tool-error", Detail: err.Error(), Determinism: report.DeterminismDeterministic}},
				Verifier:   report.VerifierInfo{Name: c.Info.Name, Version: c.Info.Version, Determinism: report.DeterminismDeterministic},
			},
		}, nil
	}

	// Stamp verifier identity onto every returned claim (plugins don't self-identify per claim).
	det := report.Determinism(c.Info.Determinism)
	for i := range result.Claims {
		result.Claims[i].Verifier = report.VerifierInfo{
			Name:        c.Info.Name,
			Version:     c.Info.Version,
			Determinism: det,
		}
	}
	return result.Claims, nil
}

// Shutdown sends the shutdown RPC and waits for the subprocess to exit.
// If the client is tainted (a previous call timed out), it kills the process directly
// to avoid racing on the decoder with the leaked goroutine.
func (c *Client) Shutdown(ctx context.Context) error {
	if c.tainted.Load() {
		if c.cmd.Process != nil {
			c.cmd.Process.Kill()
		}
	} else {
		_ = c.call(ctx, "shutdown", struct{}{}, nil)
	}
	return c.cmd.Wait()
}

// Close kills the subprocess without a graceful shutdown.
func (c *Client) Close() {
	if c.cmd.Process != nil {
		c.cmd.Process.Kill()
	}
}

// ToolError returns true if the claim is a tool-error degradation (not a real verdict).
func IsToolError(c report.Claim) bool {
	return c.Type == "tool-error"
}

// ClientFactory creates a Client for a given language.
type ClientFactory func(ctx context.Context, ectx report.ExecutionContext, language string) (*Client, error)

// StdinWriteCloser wraps an io.WriteCloser for use in tests.
type StdinWriteCloser = io.WriteCloser
