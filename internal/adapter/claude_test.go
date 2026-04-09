package adapter

import (
	"bytes"
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
	if err := os.WriteFile(filepath.Join(tmpDir, "AGENTS.yaml"), []byte(contextYAML), 0o600); err != nil {
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

	var out, errOut bytes.Buffer
	if err := HandleClaudeHook(inputBytes, &out, &errOut); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var hookOutput ClaudeHookOutput
	if err := json.Unmarshal(out.Bytes(), &hookOutput); err != nil {
		t.Fatalf("failed to parse output JSON: %v (output was: %s)", err, out.String())
	}

	if hookOutput.HookSpecificOutput == nil {
		t.Fatal("expected hookSpecificOutput to be present")
	}

	if hookOutput.HookSpecificOutput.HookEventName != "PreToolUse" {
		t.Errorf("expected hookEventName PreToolUse, got %s", hookOutput.HookSpecificOutput.HookEventName)
	}

	ctx := hookOutput.HookSpecificOutput.AdditionalContext
	if !strings.Contains(ctx, "Test context before edit") {
		t.Errorf("expected before-edit context, got: %s", ctx)
	}
	if strings.Contains(ctx, "Test context after edit") {
		t.Errorf("should not contain after-edit context, got: %s", ctx)
	}

	if hookOutput.HookSpecificOutput.PermissionDecision != "allow" {
		t.Errorf("expected permissionDecision 'allow', got %q", hookOutput.HookSpecificOutput.PermissionDecision)
	}

	if hookOutput.HookSpecificOutput.PermissionDecisionReason != "sctx: structured context injected" {
		t.Errorf("expected permissionDecisionReason 'sctx: structured context injected', got %q", hookOutput.HookSpecificOutput.PermissionDecisionReason)
	}
}

func TestHandleClaudeHook_PreToolUseMultiEdit(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, ".git"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	contextYAML := `
context:
  - content: "Edit guidance"
    on: edit
    when: before
  - content: "Read guidance"
    on: read
    when: before
`
	if err := os.WriteFile(filepath.Join(tmpDir, "AGENTS.yaml"), []byte(contextYAML), 0o600); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(tmpDir, "file.py")

	if err := os.WriteFile(target, []byte("# existing file"), 0o600); err != nil {
		t.Fatal(err)
	}

	inputBytes := marshalInput(t, ClaudeHookInput{
		SessionID:     "test-session",
		HookEventName: "PreToolUse",
		ToolName:      "MultiEdit",
		ToolInput:     json.RawMessage(`{"file_path":"` + target + `"}`),
		CWD:           tmpDir,
	})

	var out, errOut bytes.Buffer
	if err := HandleClaudeHook(inputBytes, &out, &errOut); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var hookOutput ClaudeHookOutput
	if err := json.Unmarshal(out.Bytes(), &hookOutput); err != nil {
		t.Fatalf("failed to parse output JSON: %v (output was: %s)", err, out.String())
	}

	if hookOutput.HookSpecificOutput == nil {
		t.Fatal("expected hookSpecificOutput to be present")
	}

	if hookOutput.HookSpecificOutput.HookEventName != "PreToolUse" {
		t.Errorf("expected hookEventName PreToolUse, got %s", hookOutput.HookSpecificOutput.HookEventName)
	}

	if hookOutput.HookSpecificOutput.PermissionDecision != "allow" {
		t.Errorf("expected permissionDecision 'allow', got %q", hookOutput.HookSpecificOutput.PermissionDecision)
	}

	ctx := hookOutput.HookSpecificOutput.AdditionalContext

	if !strings.Contains(ctx, "Edit guidance") {
		t.Errorf("expected edit context for MultiEdit, got: %s", ctx)
	}

	if strings.Contains(ctx, "Read guidance") {
		t.Errorf("MultiEdit should not include read-only context, got: %s", ctx)
	}
}

func TestHandleClaudeHook_NoFilePath(t *testing.T) {
	inputBytes := marshalInput(t, ClaudeHookInput{
		SessionID:     "test-session",
		HookEventName: "PreToolUse",
		ToolName:      "Bash",
		ToolInput:     json.RawMessage(`{"command":"ls"}`),
	})

	var out, errOut bytes.Buffer

	err := HandleClaudeHook(inputBytes, &out, &errOut)
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
	if err := os.WriteFile(filepath.Join(tmpDir, "AGENTS.yaml"), []byte(contextYAML), 0o600); err != nil {
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

	var out, errOut bytes.Buffer
	if err := HandleClaudeHook(inputBytes, &out, &errOut); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var hookOutput ClaudeHookOutput
	if err := json.Unmarshal(out.Bytes(), &hookOutput); err != nil {
		t.Fatalf("failed to parse output: %v (output: %s)", err, out.String())
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

func TestHandleClaudeHook_WriteExistingFile(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(tmpDir, "AGENTS.yaml"), []byte(contextYAML), 0o600); err != nil {
		t.Fatal(err)
	}

	// File exists — Write should be treated as edit.
	target := filepath.Join(tmpDir, "existing.py")

	if err := os.WriteFile(target, []byte("# already here"), 0o600); err != nil {
		t.Fatal(err)
	}

	inputBytes := marshalInput(t, ClaudeHookInput{
		SessionID:     "test-session",
		HookEventName: "PreToolUse",
		ToolName:      "Write",
		ToolInput:     json.RawMessage(`{"file_path":"` + target + `"}`),
		CWD:           tmpDir,
	})

	var out, errOut bytes.Buffer
	if err := HandleClaudeHook(inputBytes, &out, &errOut); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var hookOutput ClaudeHookOutput
	if err := json.Unmarshal(out.Bytes(), &hookOutput); err != nil {
		t.Fatalf("failed to parse output: %v (output: %s)", err, out.String())
	}

	ctx := hookOutput.HookSpecificOutput.AdditionalContext
	if ctx == "" {
		t.Fatal("expected context for edit action")
	}

	if !strings.Contains(ctx, "Context for edits only") {
		t.Errorf("expected edit context, got: %s", ctx)
	}

	if strings.Contains(ctx, "Context for new files") {
		t.Errorf("should not contain create-only context, got: %s", ctx)
	}
}

func TestHandleClaudeHook_UnknownToolName(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, ".git"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	contextYAML := `
context:
  - content: "Context for all actions"
    on: all
    when: before
  - content: "Context for edits only"
    on: edit
    when: before
`
	if err := os.WriteFile(filepath.Join(tmpDir, "AGENTS.yaml"), []byte(contextYAML), 0o600); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(tmpDir, "somefile.py")

	if err := os.WriteFile(target, []byte("# content"), 0o600); err != nil {
		t.Fatal(err)
	}

	inputBytes := marshalInput(t, ClaudeHookInput{
		SessionID:     "test-session",
		HookEventName: "PreToolUse",
		ToolName:      "UnknownTool",
		ToolInput:     json.RawMessage(`{"file_path":"` + target + `"}`),
		CWD:           tmpDir,
	})

	var out, errOut bytes.Buffer
	if err := HandleClaudeHook(inputBytes, &out, &errOut); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var hookOutput ClaudeHookOutput
	if err := json.Unmarshal(out.Bytes(), &hookOutput); err != nil {
		t.Fatalf("failed to parse output: %v (output: %s)", err, out.String())
	}

	if hookOutput.HookSpecificOutput == nil {
		t.Fatal("expected hookSpecificOutput to be present")
	}

	ctx := hookOutput.HookSpecificOutput.AdditionalContext

	if !strings.Contains(ctx, "Context for all actions") {
		t.Errorf("expected all-action context for unknown tool, got: %s", ctx)
	}

	if !strings.Contains(ctx, "Context for edits only") {
		t.Errorf("unknown tool should see all context including edit-only, got: %s", ctx)
	}
}

func TestHandleClaudeHook_PostToolUse_NoPermissionDecision(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, ".git"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	contextYAML := `
context:
  - content: "Test context after edit"
    on: edit
    when: after
`
	if err := os.WriteFile(filepath.Join(tmpDir, "AGENTS.yaml"), []byte(contextYAML), 0o600); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(tmpDir, "file.py")

	if err := os.WriteFile(target, []byte("# existing file"), 0o600); err != nil {
		t.Fatal(err)
	}

	inputBytes := marshalInput(t, ClaudeHookInput{
		SessionID:     "test-session",
		HookEventName: "PostToolUse",
		ToolName:      "Edit",
		ToolInput:     json.RawMessage(`{"file_path":"` + target + `"}`),
		CWD:           tmpDir,
	})

	var out, errOut bytes.Buffer
	if err := HandleClaudeHook(inputBytes, &out, &errOut); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var hookOutput ClaudeHookOutput
	if err := json.Unmarshal(out.Bytes(), &hookOutput); err != nil {
		t.Fatalf("failed to parse output JSON: %v (output was: %s)", err, out.String())
	}

	if hookOutput.HookSpecificOutput == nil {
		t.Fatal("expected hookSpecificOutput to be present")
	}

	if hookOutput.HookSpecificOutput.PermissionDecision != "" {
		t.Errorf("expected empty permissionDecision for PostToolUse, got %q", hookOutput.HookSpecificOutput.PermissionDecision)
	}

	if hookOutput.HookSpecificOutput.PermissionDecisionReason != "" {
		t.Errorf("expected empty permissionDecisionReason for PostToolUse, got %q", hookOutput.HookSpecificOutput.PermissionDecisionReason)
	}
}

func TestHandleClaudeHook_PreToolUse_NoAutoAllow(t *testing.T) {
	tests := []struct {
		name        string
		contextYAML string
	}{
		{
			name: "no matching context",
			contextYAML: `
context:
  - content: "Python-only context"
    on: edit
    when: before
    match:
      - "**/*.py"
`,
		},
		{
			name:        "empty context list",
			contextYAML: "context: []\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			if err := os.WriteFile(filepath.Join(tmpDir, ".git"), []byte(""), 0o600); err != nil {
				t.Fatal(err)
			}

			if err := os.WriteFile(filepath.Join(tmpDir, "AGENTS.yaml"), []byte(tt.contextYAML), 0o600); err != nil {
				t.Fatal(err)
			}

			target := filepath.Join(tmpDir, "file.go")

			if err := os.WriteFile(target, []byte("package main"), 0o600); err != nil {
				t.Fatal(err)
			}

			inputBytes := marshalInput(t, ClaudeHookInput{
				SessionID:     "test-session",
				HookEventName: "PreToolUse",
				ToolName:      "Edit",
				ToolInput:     json.RawMessage(`{"file_path":"` + target + `"}`),
				CWD:           tmpDir,
			})

			var out, errOut bytes.Buffer
			if err := HandleClaudeHook(inputBytes, &out, &errOut); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if out.Len() != 0 {
				t.Errorf("expected no output (no auto-allow), got: %s", out.String())
			}
		})
	}
}

// runPlanModeTest sets up a temp dir with the given AGENTS.yaml, runs
// HandleClaudeHook, and returns the raw stdout output.
func runPlanModeTest(t *testing.T, permissionMode, agentsYAML, toolName string) bytes.Buffer {
	t.Helper()

	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, ".git"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "AGENTS.yaml"), []byte(agentsYAML), 0o600); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(tmpDir, "file.go")
	if err := os.WriteFile(target, []byte("package main"), 0o600); err != nil {
		t.Fatal(err)
	}

	inputBytes := marshalInput(t, ClaudeHookInput{
		SessionID:      "test-session",
		HookEventName:  "PreToolUse",
		ToolName:       toolName,
		ToolInput:      json.RawMessage(`{"file_path":"` + target + `"}`),
		CWD:            tmpDir,
		PermissionMode: permissionMode,
	})

	var out, errOut bytes.Buffer
	if err := HandleClaudeHook(inputBytes, &out, &errOut); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return out
}

// parseHookContext parses stdout into AdditionalContext, failing if output is missing.
func parseHookContext(t *testing.T, out bytes.Buffer) string {
	t.Helper()

	var hookOutput ClaudeHookOutput
	if err := json.Unmarshal(out.Bytes(), &hookOutput); err != nil {
		t.Fatalf("failed to parse output: %v (output: %s)", err, out.String())
	}

	if hookOutput.HookSpecificOutput == nil {
		t.Fatal("expected hookSpecificOutput to be present")
	}

	return hookOutput.HookSpecificOutput.AdditionalContext
}

func TestHandleClaudeHook_PlanModeDecisions(t *testing.T) {
	agentsWithDecisions := `
context:
  - content: "Edit guidance"
    on: edit
    when: before
decisions:
  - decision: "Use Postgres over MySQL"
    rationale: "Better JSON support"
    alternatives:
      - option: "MySQL"
        reason_rejected: "Weaker JSON querying"
    revisit_when: "MySQL adds comparable JSON support"
`
	agentsDecisionsOnly := `
decisions:
  - decision: "REST over GraphQL"
    rationale: "Simpler client integration"
`
	agentsContextOnly := `
context:
  - content: "Style guide"
    on: read
    when: before
`

	t.Run("plan mode with matching decisions", func(t *testing.T) {
		out := runPlanModeTest(t, "plan", agentsDecisionsOnly, "Read")
		ctx := parseHookContext(t, out)

		if !strings.Contains(ctx, "## Architectural Decisions") {
			t.Errorf("expected decisions section, got: %s", ctx)
		}
		if !strings.Contains(ctx, "REST over GraphQL") {
			t.Errorf("expected decision text, got: %s", ctx)
		}
	})

	t.Run("non-plan mode ignores decisions", func(t *testing.T) {
		out := runPlanModeTest(t, "normal", agentsDecisionsOnly, "Read")
		if out.Len() != 0 {
			t.Errorf("expected no output, got: %s", out.String())
		}
	})

	t.Run("empty permission mode ignores decisions", func(t *testing.T) {
		out := runPlanModeTest(t, "", agentsDecisionsOnly, "Read")
		if out.Len() != 0 {
			t.Errorf("expected no output, got: %s", out.String())
		}
	})

	t.Run("plan mode context only no decisions section", func(t *testing.T) {
		out := runPlanModeTest(t, "plan", agentsContextOnly, "Read")
		ctx := parseHookContext(t, out)

		if !strings.Contains(ctx, "Style guide") {
			t.Errorf("expected context, got: %s", ctx)
		}
		if strings.Contains(ctx, "Architectural Decisions") {
			t.Errorf("did not expect decisions section, got: %s", ctx)
		}
	})

	t.Run("plan mode with both context and decisions", func(t *testing.T) {
		out := runPlanModeTest(t, "plan", agentsWithDecisions, "Edit")
		ctx := parseHookContext(t, out)

		if !strings.Contains(ctx, "## Structured Context") {
			t.Errorf("expected context section, got: %s", ctx)
		}
		if !strings.Contains(ctx, "## Architectural Decisions") {
			t.Errorf("expected decisions section, got: %s", ctx)
		}
		if !strings.Contains(ctx, "Use Postgres over MySQL") {
			t.Errorf("expected decision text, got: %s", ctx)
		}
	})

	t.Run("plan mode decisions include full detail", func(t *testing.T) {
		out := runPlanModeTest(t, "plan", agentsWithDecisions, "Edit")
		ctx := parseHookContext(t, out)

		if !strings.Contains(ctx, "Better JSON support") {
			t.Errorf("expected rationale, got: %s", ctx)
		}
		if !strings.Contains(ctx, "Considered MySQL, rejected: Weaker JSON querying") {
			t.Errorf("expected alternatives, got: %s", ctx)
		}
		if !strings.Contains(ctx, "Revisit when: MySQL adds comparable JSON support") {
			t.Errorf("expected revisit_when, got: %s", ctx)
		}
	})

	t.Run("plan mode decisions only still produces output", func(t *testing.T) {
		out := runPlanModeTest(t, "plan", agentsDecisionsOnly, "Read")
		ctx := parseHookContext(t, out)

		if !strings.Contains(ctx, "Simpler client integration") {
			t.Errorf("expected rationale, got: %s", ctx)
		}
	})
}

func TestHandleClaudeHook_MalformedInput(t *testing.T) {
	var out, errOut bytes.Buffer

	err := HandleClaudeHook([]byte(`{not valid json`), &out, &errOut)
	if err == nil {
		t.Fatal("expected error for malformed JSON input, got nil")
	}
}
