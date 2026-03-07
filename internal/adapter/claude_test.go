package adapter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// marshalInput is a test helper that marshals hook input and fails on error.
func marshalInput(t *testing.T, input ClaudeHookInput) []byte {
	t.Helper()

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal test input: %v", err)
	}

	return data
}

// captureStdout runs fn with stdout redirected to a pipe and returns what was written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	os.Stdout = w

	fn()

	if err = w.Close(); err != nil {
		t.Fatalf("failed to close pipe writer: %v", err)
	}

	os.Stdout = oldStdout

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)

	return string(buf[:n])
}

func TestHandleClaudeHook_PreToolUseEdit(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, ".git"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	contextYAML := `
context:
  - content: "Test context before edit"
    on: edit
    when: before
  - content: "Test context after edit"
    on: edit
    when: after
`
	if err := os.WriteFile(filepath.Join(tmpDir, "CONTEXT.yaml"), []byte(contextYAML), 0o600); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(tmpDir, "file.py")

	if err := os.WriteFile(target, []byte("# existing file"), 0o600); err != nil {
		t.Fatal(err)
	}

	inputBytes := marshalInput(t, ClaudeHookInput{
		SessionID:     "test-session",
		HookEventName: "PreToolUse",
		ToolName:      "Edit",
		ToolInput:     json.RawMessage(`{"file_path":"` + target + `"}`),
		CWD:           tmpDir,
	})

	output := captureStdout(t, func() {
		if err := HandleClaudeHook(inputBytes); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	var hookOutput ClaudeHookOutput
	if err := json.Unmarshal([]byte(output), &hookOutput); err != nil {
		t.Fatalf("failed to parse output JSON: %v (output was: %s)", err, output)
	}

	if hookOutput.HookSpecificOutput == nil {
		t.Fatal("expected hookSpecificOutput to be present")
	}

	if hookOutput.HookSpecificOutput.HookEventName != "PreToolUse" {
		t.Errorf("expected hookEventName PreToolUse, got %s", hookOutput.HookSpecificOutput.HookEventName)
	}

	if hookOutput.HookSpecificOutput.AdditionalContext == "" {
		t.Error("expected non-empty additionalContext")
	}
}

func TestHandleClaudeHook_NoFilePath(t *testing.T) {
	inputBytes := marshalInput(t, ClaudeHookInput{
		SessionID:     "test-session",
		HookEventName: "PreToolUse",
		ToolName:      "Bash",
		ToolInput:     json.RawMessage(`{"command":"ls"}`),
	})

	err := HandleClaudeHook(inputBytes)
	if err != nil {
		t.Fatalf("expected no error for tool without file_path, got: %v", err)
	}
}

func TestHandleClaudeHook_WriteNewFile(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, ".git"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	contextYAML := `
context:
  - content: "Context for new files"
    on: create
    when: before
  - content: "Context for edits only"
    on: edit
    when: before
`
	if err := os.WriteFile(filepath.Join(tmpDir, "CONTEXT.yaml"), []byte(contextYAML), 0o600); err != nil {
		t.Fatal(err)
	}

	// File does not exist — should be treated as create.
	target := filepath.Join(tmpDir, "newfile.py")

	inputBytes := marshalInput(t, ClaudeHookInput{
		SessionID:     "test-session",
		HookEventName: "PreToolUse",
		ToolName:      "Write",
		ToolInput:     json.RawMessage(`{"file_path":"` + target + `"}`),
		CWD:           tmpDir,
	})

	output := captureStdout(t, func() {
		if err := HandleClaudeHook(inputBytes); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	var hookOutput ClaudeHookOutput
	if err := json.Unmarshal([]byte(output), &hookOutput); err != nil {
		t.Fatalf("failed to parse output: %v (output: %s)", err, output)
	}

	ctx := hookOutput.HookSpecificOutput.AdditionalContext
	if ctx == "" {
		t.Fatal("expected context for create action")
	}

	if !strings.Contains(ctx, "Context for new files") {
		t.Errorf("expected create context, got: %s", ctx)
	}

	if strings.Contains(ctx, "Context for edits only") {
		t.Errorf("should not contain edit-only context, got: %s", ctx)
	}
}
