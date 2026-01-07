package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Define minimal JSON-RPC types for the test
type jsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      int         `json:"id"`
}

type jsonRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	ID      int         `json:"id"`
}

// TestE2E_ObsidianMCP runs the compiled binary against a fake server.
func TestE2E_ObsidianMCP(t *testing.T) {
	// 1. Start Fake Obsidian Server
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/active/" && r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// Return a minimal valid Note JSON
			_, _ = w.Write([]byte(`{
				"content": "Hello from fake obsidian!",
				"frontmatter": {},
				"path": "fake.md",
				"stat": {"ctime": 0, "mtime": 0, "size": 0},
				"tags": []
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	// 2. Create Config
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.json")

	configContent := map[string]interface{}{
		"obsidian": map[string]string{
			"url":    ts.URL,
			"apikey": "dummy-key",
			"cert":   "", // Empty to trigger InsecureSkipVerify
		},
		"mcp": map[string]interface{}{
			"tools": map[string]bool{
				"get_active_file": true,
			},
		},
	}

	configBytes, err := json.Marshal(configContent)
	require.NoError(t, err)
	err = os.WriteFile(configPath, configBytes, 0644)
	require.NoError(t, err)

	// 3. Build/Run obsidianmcp
	// Use "go run main.go" (plus other necessary files if split)
	// Since we are in the same package (main), we can't easily "run ourselves" without infinite recursion if we called TestMain.
	// But exec.Command("go", "run", ...) starts a separate process.
	// We need to point to the current directory ".".
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// We use "go run ." to run the main package in the current directory.
	// NOTE: This assumes we are running the test from the cmd/obsidianmcp directory or we adjust the path.
	// If we run `go test ./cmd/obsidianmcp`, the CWD is that directory? No, usually it's the package directory.
	// Let's verify CWD or use absolute path to main.go.
	// Safer: Build a binary first.
	// 3. Build/Run obsidianmcp with coverage instrumentation
	// To get coverage from the subprocess, we need Go 1.20+ and build with -cover.
	// We also need to set GOCOVERDIR env var.
	coverDir := filepath.Join(configDir, "covdata")
	err = os.MkdirAll(coverDir, 0755)
	require.NoError(t, err)

	binPath := filepath.Join(configDir, "obsidianmcp")
	// Add -cover flag
	buildCmd := exec.Command("go", "build", "-cover", "-o", binPath, ".")
	// If the test is running in the package directory, "." works.
	// If not, we might fail. Let's assume standard `go test ./cmd/obsidianmcp` execution where CWD is the package dir.

	out, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Build failed: %s", string(out))

	cmd := exec.CommandContext(ctx, binPath, "-config", configPath)

	// Set GOCOVERDIR for the subprocess
	cmd.Env = append(os.Environ(), "GOCOVERDIR="+coverDir)

	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)

	cmd.Stderr = os.Stderr // Pass through logs

	err = cmd.Start()
	require.NoError(t, err)
	defer func() {
		// Note: Coverage data is written on exit.
		// Killing forcefully (Kill) might prevent writing coverage data depending on implementation,
		// but typical nice termination is better.
		// Since we are using a fake interaction, we might want to close stdin
		// or send a signal to let it exit gracefully if possible.
		// However, for this test, let's try to ensure we minimally wait or just rely on the fact
		// that we can't easily graceful exit without a "shutdown" RPC method or SIGTERM.
		// server.ServeStdio handles SIGTERM.
		// Let's replace Kill with Signal.
		_ = cmd.Process.Signal(os.Interrupt)
		_ = cmd.Wait() // Wait for file flush

		t.Logf("Coverage data written to: %s", coverDir)

		// Print coverage percentage
		covCmd := exec.Command("go", "tool", "covdata", "percent", "-i="+coverDir)
		covOut, err := covCmd.CombinedOutput()
		if err != nil {
			t.Logf("Failed to get coverage: %v", err)
		} else {
			t.Logf("Per-package coverage:\n%s", string(covOut))
		}

		// Calculate overall total
		profilePath := filepath.Join(coverDir, "profile.txt")
		txtFmtCmd := exec.Command("go", "tool", "covdata", "textfmt", "-i="+coverDir, "-o="+profilePath)
		if err := txtFmtCmd.Run(); err != nil {
			t.Logf("Failed to convert coverage to text format: %v", err)
		} else {
			funcCmd := exec.Command("go", "tool", "cover", "-func="+profilePath)
			funcOut, err := funcCmd.CombinedOutput()
			if err != nil {
				t.Logf("Failed to calculate total coverage: %v", err)
			} else {
				t.Logf("Overall Coverage Stats:\n%s", string(funcOut))
			}
		}
	}()

	// Helper to send/receive messages
	encoder := json.NewEncoder(stdin)
	decoder := json.NewDecoder(stdout)

	// 4. Send Initialize
	initReq := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0",
			},
		},
		ID: 1,
	}
	err = encoder.Encode(initReq)
	require.NoError(t, err)

	var initResp jsonRPCResponse
	err = decoder.Decode(&initResp)
	require.NoError(t, err)
	assert.Nil(t, initResp.Error)

	// 6. Call get_active_file
	callReq := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      "obsidian_get_active_file",
			"arguments": map[string]interface{}{},
		},
		ID: 2,
	}
	err = encoder.Encode(callReq)
	require.NoError(t, err)

	var callResp jsonRPCResponse
	err = decoder.Decode(&callResp)
	require.NoError(t, err)
	require.Nil(t, callResp.Error, "Tool call returned error")

	// Validate Result
	resultMap, ok := callResp.Result.(map[string]interface{})
	require.True(t, ok, "Result should be a map")

	content, ok := resultMap["content"].([]interface{})
	require.True(t, ok, "Content should be a list")
	require.NotEmpty(t, content)

	textObj, ok := content[0].(map[string]interface{})
	require.True(t, ok, "Item should be a map")
	assert.Equal(t, "text", textObj["type"])

	// The content of the tool result is the JSON representation of the Note struct
	resultJSONStr, ok := textObj["text"].(string)
	require.True(t, ok, "Text should be a string")

	var noteMap map[string]interface{}
	err = json.Unmarshal([]byte(resultJSONStr), &noteMap)
	require.NoError(t, err, "Failed to parse result JSON: %s", resultJSONStr)

	assert.Equal(t, "Hello from fake obsidian!", noteMap["content"])
}
