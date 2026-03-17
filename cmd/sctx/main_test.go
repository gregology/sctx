package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gregology/sctx/internal/adapter"
)

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal test input: %v", err)
	}

	return data
}

const testAgentsYAML = `context:
  - content: "test guidance"
    match: ["**"]
    on: read
    when: before

  - content: "edit guidance"
    match: ["**/*.go"]
    on: edit
    when: after

decisions:
  - decision: "use Go"
    rationale: "fast startup"
    alternatives:
      - option: "Python"
        reason_rejected: "slow"
    revisit_when: "never"
    date: 2025-01-01
    match: ["**"]
`

func TestCmdContext(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, filepath.Join(tmp, "AGENTS.yaml"), testAgentsYAML)

	target := filepath.Join(tmp, "foo.go")
	writeTestFile(t, target, "package foo\n")

	tests := []struct {
		name    string
		args    []string
		wantOut string
		wantErr error
	}{
		{
			name:    "missing path",
			args:    []string{},
			wantErr: errMissingPath,
		},
		{
			name:    "text output with match",
			args:    []string{target},
			wantOut: "test guidance",
		},
		{
			name:    "on read filter",
			args:    []string{target, "--on", "read"},
			wantOut: "test guidance",
		},
		{
			name:    "on edit filter",
			args:    []string{target, "--on", "edit", "--when", "after"},
			wantOut: "edit guidance",
		},
		{
			name:    "json output",
			args:    []string{target, "--on", "read", "--json"},
			wantOut: `"Content": "test guidance"`,
		},
		{
			name:    "no match",
			args:    []string{target, "--on", "create"},
			wantOut: "No matching context found.",
		},
		{
			name:    "on missing value",
			args:    []string{target, "--on"},
			wantErr: errOnNeedsValue,
		},
		{
			name:    "when missing value",
			args:    []string{target, "--when"},
			wantErr: errWhenNeedsValue,
		},
		{
			name:    "invalid action",
			args:    []string{target, "--on", "nope"},
			wantErr: errInvalidAction,
		},
		{
			name:    "invalid timing",
			args:    []string{target, "--when", "nope"},
			wantErr: errInvalidTiming,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errOut bytes.Buffer

			err := cmdContext(tt.args, &out, &errOut)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("got error %v, want %v", err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !strings.Contains(out.String(), tt.wantOut) {
				t.Errorf("output %q does not contain %q", out.String(), tt.wantOut)
			}
		})
	}
}

func TestCmdContext_TextFormat(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, filepath.Join(tmp, "AGENTS.yaml"), testAgentsYAML)

	target := filepath.Join(tmp, "foo.go")
	writeTestFile(t, target, "package foo\n")

	var out, errOut bytes.Buffer

	if err := cmdContext([]string{target, "--on", "read"}, &out, &errOut); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	if !strings.Contains(got, "  - test guidance") {
		t.Errorf("expected indented content, got %q", got)
	}

	if !strings.Contains(got, "(from "+tmp+")") {
		t.Errorf("expected source dir, got %q", got)
	}
}

func TestCmdContext_FlagOrder(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, filepath.Join(tmp, "AGENTS.yaml"), testAgentsYAML)

	target := filepath.Join(tmp, "foo.go")
	writeTestFile(t, target, "package foo\n")

	var out, errOut bytes.Buffer

	err := cmdContext([]string{target, "--json", "--on", "read", "--when", "before"}, &out, &errOut)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out.String(), `"Content"`) {
		t.Errorf("expected JSON output, got %q", out.String())
	}
}

func TestCmdDecisions(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, filepath.Join(tmp, "AGENTS.yaml"), testAgentsYAML)

	target := filepath.Join(tmp, "foo.go")
	writeTestFile(t, target, "package foo\n")

	tests := []struct {
		name    string
		args    []string
		wantOut string
		wantErr error
	}{
		{
			name:    "missing path",
			args:    []string{},
			wantErr: errMissingPath,
		},
		{
			name:    "text output",
			args:    []string{target},
			wantOut: "use Go",
		},
		{
			name:    "text shows rationale",
			args:    []string{target},
			wantOut: "Rationale: fast startup",
		},
		{
			name:    "text shows alternatives",
			args:    []string{target},
			wantOut: "Considered Python, rejected: slow",
		},
		{
			name:    "text shows revisit_when",
			args:    []string{target},
			wantOut: "Revisit when: never",
		},
		{
			name:    "json output",
			args:    []string{target, "--json"},
			wantOut: `"Decision": "use Go"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errOut bytes.Buffer

			err := cmdDecisions(tt.args, &out, &errOut)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("got error %v, want %v", err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !strings.Contains(out.String(), tt.wantOut) {
				t.Errorf("output %q does not contain %q", out.String(), tt.wantOut)
			}
		})
	}
}

func TestCmdDecisions_NoMatch(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, filepath.Join(tmp, "AGENTS.yaml"), `context:
  - content: "only context, no decisions"
    match: ["**"]
`)

	target := filepath.Join(tmp, "foo.go")
	writeTestFile(t, target, "package foo\n")

	var out, errOut bytes.Buffer

	if err := cmdDecisions([]string{target}, &out, &errOut); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out.String(), "No matching decisions found.") {
		t.Errorf("expected no-match message, got %q", out.String())
	}
}

func TestCmdValidate(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantOut string
		wantErr error
	}{
		{
			name:    "valid file",
			yaml:    testAgentsYAML,
			wantOut: "All context files are valid.",
		},
		{
			name: "invalid file returns error",
			yaml: `context:
  - content: ""
    on: nope
`,
			wantErr: errValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			writeTestFile(t, filepath.Join(tmp, "AGENTS.yaml"), tt.yaml)

			var out bytes.Buffer

			err := cmdValidate([]string{tmp}, &out)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("got error %v, want %v", err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !strings.Contains(out.String(), tt.wantOut) {
				t.Errorf("output %q does not contain %q", out.String(), tt.wantOut)
			}
		})
	}
}

func TestCmdValidate_DefaultDir(t *testing.T) {
	var out bytes.Buffer

	err := cmdValidate([]string{}, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCmdInit(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	var out bytes.Buffer

	if err := cmdInit(&out); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out.String(), "Created AGENTS.yaml") {
		t.Errorf("expected creation message, got %q", out.String())
	}

	data, err := os.ReadFile(filepath.Join(tmp, "AGENTS.yaml")) //nolint:gosec // test reads known temp file
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(data), "Structured Context") {
		t.Errorf("file content missing header, got %q", string(data))
	}
}

func TestCmdInit_AlreadyExists(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	writeTestFile(t, filepath.Join(tmp, "AGENTS.yaml"), "existing\n")

	var out bytes.Buffer

	err := cmdInit(&out)
	if !errors.Is(err, errFileExists) {
		t.Errorf("got error %v, want %v", err, errFileExists)
	}
}

func TestCmdClaude(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr error
	}{
		{
			name:    "missing subcommand",
			args:    []string{},
			wantErr: errClaudeSubcommand,
		},
		{
			name:    "invalid subcommand",
			args:    []string{"bogus"},
			wantErr: errClaudeSubcommand,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cmdClaude(tt.args)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got error %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCmdPi(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr error
	}{
		{
			name:    "missing subcommand",
			args:    []string{},
			wantErr: errPiSubcommand,
		},
		{
			name:    "invalid subcommand",
			args:    []string{"bogus"},
			wantErr: errPiSubcommand,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cmdPi(tt.args)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got error %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCmdHook(t *testing.T) {
	tmp := t.TempDir()
	writeTestFile(t, filepath.Join(tmp, ".git"), "")
	writeTestFile(t, filepath.Join(tmp, "AGENTS.yaml"), `context:
  - content: "hook test context"
    on: edit
    when: before
`)
	target := filepath.Join(tmp, "file.go")
	writeTestFile(t, target, "package main\n")

	claudeInput := mustMarshal(t, adapter.ClaudeHookInput{
		SessionID:     "s1",
		HookEventName: "PreToolUse",
		ToolName:      "Edit",
		ToolInput:     json.RawMessage(`{"file_path":"` + target + `"}`),
		CWD:           tmp,
	})

	piInput := mustMarshal(t, adapter.PiHookInput{
		Source:   "pi",
		Event:    "tool_call",
		ToolName: "edit",
		Input:    json.RawMessage(`{"path":"` + target + `"}`),
		CWD:      tmp,
	})

	piWithClaudeShape := mustMarshal(t, adapter.PiHookInput{
		Source:   "pi",
		Event:    "tool_call",
		ToolName: "edit",
		Input:    json.RawMessage(`{"path":"` + target + `","file_path":"` + target + `"}`),
		CWD:      tmp,
	})

	tests := []struct {
		name       string
		input      []byte
		wantErr    bool
		wantOut    string
		notWantOut string
	}{
		{
			name:    "claude hook dispatches to claude handler",
			input:   claudeInput,
			wantOut: "hookSpecificOutput",
		},
		{
			name:    "pi hook dispatches to pi handler",
			input:   piInput,
			wantOut: "additionalContext",
		},
		{
			name:    "empty input returns error",
			input:   []byte{},
			wantErr: true,
		},
		{
			name:    "malformed JSON returns error",
			input:   []byte(`{not valid`),
			wantErr: true,
		},
		{
			name:    "pi source with claude-shaped payload routes to pi handler",
			input:   piWithClaudeShape,
			wantOut: "additionalContext",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errOut bytes.Buffer

			err := cmdHook(bytes.NewReader(tt.input), &out, &errOut)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantOut != "" && !strings.Contains(out.String(), tt.wantOut) {
				t.Errorf("output %q does not contain %q", out.String(), tt.wantOut)
			}

			if tt.notWantOut != "" && strings.Contains(out.String(), tt.notWantOut) {
				t.Errorf("output %q should not contain %q", out.String(), tt.notWantOut)
			}
		})
	}
}
