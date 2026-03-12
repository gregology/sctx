package adapter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func marshalPiInput(t *testing.T, input PiHookInput) []byte {
	t.Helper()

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal test input: %v", err)
	}

	return data
}

const piCreateEditYAML = `
context:
  - content: "Context for new files"
    on: create
    when: before
  - content: "Context for edits only"
    on: edit
    when: before
`

// setupPiTestDir creates a temp directory with .git marker and AGENTS.yaml.
func setupPiTestDir(t *testing.T, contextYAML string) string {
	t.Helper()

	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, ".git"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "AGENTS.yaml"), []byte(contextYAML), 0o600); err != nil {
		t.Fatal(err)
	}

	return tmpDir
}

// runPiHook runs HandlePiHook with captured stdout and returns the parsed output.
func runPiHook(t *testing.T, input PiHookInput) (string, *PiHookOutput) {
	t.Helper()

	inputBytes := marshalPiInput(t, input)

	output := captureStdout(t, func() {
		if err := HandlePiHook(inputBytes); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if output == "" {
		return "", nil
	}

	var hookOutput PiHookOutput
	if err := json.Unmarshal([]byte(output), &hookOutput); err != nil {
		t.Fatalf("failed to parse output JSON: %v (output was: %s)", err, output)
	}

	return output, &hookOutput
}

func TestHandlePiHook_ToolCallEdit(t *testing.T) {
	contextYAML := `
context:
  - content: "Test context before edit"
    on: edit
    when: before
  - content: "Test context after edit"
    on: edit
    when: after
`
	tmpDir := setupPiTestDir(t, contextYAML)
	target := filepath.Join(tmpDir, "file.py")

	if err := os.WriteFile(target, []byte("# existing file"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, out := runPiHook(t, PiHookInput{
		Source:   "pi",
		Event:    "tool_call",
		ToolName: "edit",
		Input:    json.RawMessage(`{"path":"` + target + `"}`),
		CWD:      tmpDir,
	})

	if out == nil || out.AdditionalContext == "" {
		t.Fatal("expected non-empty additionalContext")
	}

	if !strings.Contains(out.AdditionalContext, "Test context before edit") {
		t.Errorf("expected before-edit context, got: %s", out.AdditionalContext)
	}

	if strings.Contains(out.AdditionalContext, "Test context after edit") {
		t.Errorf("should not contain after-edit context in tool_call, got: %s", out.AdditionalContext)
	}
}

func TestHandlePiHook_NoPath(t *testing.T) {
	inputBytes := marshalPiInput(t, PiHookInput{
		Source:   "pi",
		Event:    "tool_call",
		ToolName: "bash",
		Input:    json.RawMessage(`{"command":"ls"}`),
		CWD:      "/tmp",
	})

	err := HandlePiHook(inputBytes)
	if err != nil {
		t.Fatalf("expected no error for tool without path, got: %v", err)
	}
}

func TestHandlePiHook_WriteNewFile(t *testing.T) {
	tmpDir := setupPiTestDir(t, piCreateEditYAML)
	target := filepath.Join(tmpDir, "newfile.py")

	_, out := runPiHook(t, PiHookInput{
		Source:   "pi",
		Event:    "tool_call",
		ToolName: "write",
		Input:    json.RawMessage(`{"path":"` + target + `","content":"hello"}`),
		CWD:      tmpDir,
	})

	if out == nil {
		t.Fatal("expected output for create action")
	}

	if !strings.Contains(out.AdditionalContext, "Context for new files") {
		t.Errorf("expected create context, got: %s", out.AdditionalContext)
	}

	if strings.Contains(out.AdditionalContext, "Context for edits only") {
		t.Errorf("should not contain edit-only context, got: %s", out.AdditionalContext)
	}
}

func TestHandlePiHook_WriteExistingFile(t *testing.T) {
	tmpDir := setupPiTestDir(t, piCreateEditYAML)
	target := filepath.Join(tmpDir, "existing.py")

	if err := os.WriteFile(target, []byte("# already here"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, out := runPiHook(t, PiHookInput{
		Source:   "pi",
		Event:    "tool_call",
		ToolName: "write",
		Input:    json.RawMessage(`{"path":"` + target + `","content":"updated"}`),
		CWD:      tmpDir,
	})

	if out == nil {
		t.Fatal("expected output for edit action")
	}

	if !strings.Contains(out.AdditionalContext, "Context for edits only") {
		t.Errorf("expected edit context, got: %s", out.AdditionalContext)
	}

	if strings.Contains(out.AdditionalContext, "Context for new files") {
		t.Errorf("should not contain create-only context, got: %s", out.AdditionalContext)
	}
}

func TestHandlePiHook_UnknownToolName(t *testing.T) {
	contextYAML := `
context:
  - content: "Context for all actions"
    on: all
    when: before
  - content: "Context for edits only"
    on: edit
    when: before
`
	tmpDir := setupPiTestDir(t, contextYAML)
	target := filepath.Join(tmpDir, "somefile.py")

	if err := os.WriteFile(target, []byte("# content"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, out := runPiHook(t, PiHookInput{
		Source:   "pi",
		Event:    "tool_call",
		ToolName: "unknown_tool",
		Input:    json.RawMessage(`{"path":"` + target + `"}`),
		CWD:      tmpDir,
	})

	if out == nil {
		t.Fatal("expected output for unknown tool")
	}

	if !strings.Contains(out.AdditionalContext, "Context for all actions") {
		t.Errorf("expected all-action context for unknown tool, got: %s", out.AdditionalContext)
	}

	if !strings.Contains(out.AdditionalContext, "Context for edits only") {
		t.Errorf("unknown tool should see all context including edit-only, got: %s", out.AdditionalContext)
	}
}

func TestHandlePiHook_ToolResult(t *testing.T) {
	contextYAML := `
context:
  - content: "Test context after edit"
    on: edit
    when: after
  - content: "Test context before edit"
    on: edit
    when: before
`
	tmpDir := setupPiTestDir(t, contextYAML)
	target := filepath.Join(tmpDir, "file.py")

	if err := os.WriteFile(target, []byte("# existing file"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, out := runPiHook(t, PiHookInput{
		Source:   "pi",
		Event:    "tool_result",
		ToolName: "edit",
		Input:    json.RawMessage(`{"path":"` + target + `"}`),
		CWD:      tmpDir,
	})

	if out == nil {
		t.Fatal("expected output for tool_result")
	}

	if !strings.Contains(out.AdditionalContext, "Test context after edit") {
		t.Errorf("expected after-edit context, got: %s", out.AdditionalContext)
	}

	if strings.Contains(out.AdditionalContext, "Test context before edit") {
		t.Errorf("should not contain before-edit context in tool_result, got: %s", out.AdditionalContext)
	}
}

func TestHandlePiHook_NoMatchingContext(t *testing.T) {
	contextYAML := `
context:
  - content: "Python-only context"
    on: edit
    when: before
    match:
      - "**/*.py"
`
	tmpDir := setupPiTestDir(t, contextYAML)
	target := filepath.Join(tmpDir, "file.go")

	if err := os.WriteFile(target, []byte("package main"), 0o600); err != nil {
		t.Fatal(err)
	}

	raw, _ := runPiHook(t, PiHookInput{
		Source:   "pi",
		Event:    "tool_call",
		ToolName: "edit",
		Input:    json.RawMessage(`{"path":"` + target + `"}`),
		CWD:      tmpDir,
	})

	if raw != "" {
		t.Errorf("expected no output for non-matching context, got: %s", raw)
	}
}

func TestBashReadPath(t *testing.T) {
	tests := []struct {
		command string
		want    string
	}{
		{"cat go.mod", "go.mod"},
		{"head -20 internal/core/schema.go", "internal/core/schema.go"},
		{"tail -n 50 go.sum", "go.sum"},
		{"cat -n go.mod", "go.mod"},
		{"cat go.mod | grep require", "go.mod"},
		{"head -n 5 go.mod | wc -l", "go.mod"},
		{"cat", ""},
		{"cat -n", ""},
		{"ls -la", ""},
		{"grep foo bar.go", ""},
		{"echo hello | cat", ""},
		{"cat file1.go file2.go", "file1.go"},
		{`cat "my file.txt"`, ""},
		{`cat 'my file.txt'`, ""},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			got := bashReadPath(tt.command)
			if got != tt.want {
				t.Errorf("bashReadPath(%q) = %q, want %q", tt.command, got, tt.want)
			}
		})
	}
}

func TestHandlePiHook_BashCat(t *testing.T) {
	contextYAML := `
context:
  - content: "Read context for go.mod"
    on: read
    when: before
`
	tmpDir := setupPiTestDir(t, contextYAML)
	target := filepath.Join(tmpDir, "go.mod")

	if err := os.WriteFile(target, []byte("module example"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, out := runPiHook(t, PiHookInput{
		Source:   "pi",
		Event:    "tool_call",
		ToolName: "bash",
		Input:    json.RawMessage(`{"command":"cat ` + target + `"}`),
		CWD:      tmpDir,
	})

	if out == nil || out.AdditionalContext == "" {
		t.Fatal("expected context for bash cat command")
	}

	if !strings.Contains(out.AdditionalContext, "Read context for go.mod") {
		t.Errorf("expected read context, got: %s", out.AdditionalContext)
	}
}

func TestHandlePiHook_BashCatNoReadContext(t *testing.T) {
	contextYAML := `
context:
  - content: "Edit-only context"
    on: edit
    when: before
`
	tmpDir := setupPiTestDir(t, contextYAML)
	target := filepath.Join(tmpDir, "file.go")

	if err := os.WriteFile(target, []byte("package main"), 0o600); err != nil {
		t.Fatal(err)
	}

	raw, _ := runPiHook(t, PiHookInput{
		Source:   "pi",
		Event:    "tool_call",
		ToolName: "bash",
		Input:    json.RawMessage(`{"command":"cat ` + target + `"}`),
		CWD:      tmpDir,
	})

	if raw != "" {
		t.Errorf("expected no output for edit-only context on bash read, got: %s", raw)
	}
}

func TestHandlePiHook_BashNonReadCommand(t *testing.T) {
	contextYAML := `
context:
  - content: "Should not appear"
    on: read
    when: before
`
	tmpDir := setupPiTestDir(t, contextYAML)

	raw, _ := runPiHook(t, PiHookInput{
		Source:   "pi",
		Event:    "tool_call",
		ToolName: "bash",
		Input:    json.RawMessage(`{"command":"ls -la"}`),
		CWD:      tmpDir,
	})

	if raw != "" {
		t.Errorf("expected no output for non-read bash command, got: %s", raw)
	}
}

func TestHandlePiHook_MalformedInput(t *testing.T) {
	err := HandlePiHook([]byte(`{not valid json`))
	if err == nil {
		t.Fatal("expected error for malformed JSON input, got nil")
	}
}

func TestIsPiHook(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "pi source",
			input: `{"source":"pi","event":"tool_call"}`,
			want:  true,
		},
		{
			name:  "claude input",
			input: `{"session_id":"abc","hook_event_name":"PreToolUse"}`,
			want:  false,
		},
		{
			name:  "malformed json",
			input: `{not valid`,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPiHook([]byte(tt.input))
			if got != tt.want {
				t.Errorf("IsPiHook(%s) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
