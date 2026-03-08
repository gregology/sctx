package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

	buf := make([]byte, 64*1024)
	n, _ := r.Read(buf)

	return string(buf[:n])
}

// captureStderr runs fn with stderr redirected to a pipe and returns what was written.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	oldStderr := os.Stderr

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	os.Stderr = w

	fn()

	if err = w.Close(); err != nil {
		t.Fatalf("failed to close pipe writer: %v", err)
	}

	os.Stderr = oldStderr

	buf := make([]byte, 64*1024)
	n, _ := r.Read(buf)

	return string(buf[:n])
}

// writeTestFile is a helper that writes a file and fails the test on error.
func writeTestFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

// setupProject creates a temp dir with a .git marker and an AGENTS.yaml file.
func setupProject(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, ".git"), "")
	writeTestFile(t, filepath.Join(dir, "AGENTS.yaml"), `
context:
  - content: "Test context entry"
    on: all
    when: before
decisions:
  - decision: "Use Go"
    rationale: "Fast compilation"
`)

	return dir
}

func TestRun_NoArgs(t *testing.T) {
	stderr := captureStderr(t, func() {
		code := run([]string{"sctx"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
	})

	if !strings.Contains(stderr, "Usage:") {
		t.Errorf("expected usage in stderr, got: %s", stderr)
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	stderr := captureStderr(t, func() {
		code := run([]string{"sctx", "bogus"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
	})

	if !strings.Contains(stderr, "unknown command: bogus") {
		t.Errorf("expected unknown command error, got: %s", stderr)
	}
}

func TestRun_Help(t *testing.T) {
	for _, flag := range []string{"help", "--help", "-h"} {
		t.Run(flag, func(t *testing.T) {
			output := captureStdout(t, func() {
				code := run([]string{"sctx", flag})
				if code != 0 {
					t.Errorf("expected exit code 0, got %d", code)
				}
			})

			if !strings.Contains(output, "Usage:") {
				t.Errorf("expected usage output, got: %s", output)
			}
		})
	}
}

func TestRun_Version(t *testing.T) {
	output := captureStdout(t, func() {
		code := run([]string{"sctx", "version"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})

	if !strings.Contains(output, "sctx") {
		t.Errorf("expected version output containing 'sctx', got: %s", output)
	}
}

func TestCmdContext_MissingPath(t *testing.T) {
	stderr := captureStderr(t, func() {
		code := run([]string{"sctx", "context"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
	})

	if !strings.Contains(stderr, "missing required <path> argument") {
		t.Errorf("expected missing path error, got: %s", stderr)
	}
}

func TestCmdContext_OnNeedsValue(t *testing.T) {
	stderr := captureStderr(t, func() {
		code := run([]string{"sctx", "context", "/some/file", "--on"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
	})

	if !strings.Contains(stderr, "--on requires a value") {
		t.Errorf("expected --on error, got: %s", stderr)
	}
}

func TestCmdContext_WhenNeedsValue(t *testing.T) {
	stderr := captureStderr(t, func() {
		code := run([]string{"sctx", "context", "/some/file", "--when"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
	})

	if !strings.Contains(stderr, "--when requires a value") {
		t.Errorf("expected --when error, got: %s", stderr)
	}
}

func TestCmdContext_TextOutput(t *testing.T) {
	dir := setupProject(t)
	target := filepath.Join(dir, "file.go")
	writeTestFile(t, target, "")

	output := captureStdout(t, func() {
		code := run([]string{"sctx", "context", target})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})

	if !strings.Contains(output, "Test context entry") {
		t.Errorf("expected context in output, got: %s", output)
	}
}

func TestCmdContext_JSONOutput(t *testing.T) {
	dir := setupProject(t)
	target := filepath.Join(dir, "file.go")
	writeTestFile(t, target, "")

	output := captureStdout(t, func() {
		code := run([]string{"sctx", "context", target, "--json"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})

	if !strings.Contains(output, `"Content"`) {
		t.Errorf("expected JSON output with Content field, got: %s", output)
	}

	if !strings.Contains(output, "Test context entry") {
		t.Errorf("expected context content in JSON output, got: %s", output)
	}
}

func TestCmdContext_OnFilter(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, ".git"), "")
	writeTestFile(t, filepath.Join(dir, "AGENTS.yaml"), `
context:
  - content: "Edit only context"
    on: edit
    when: before
  - content: "Read only context"
    on: read
    when: before
`)

	target := filepath.Join(dir, "file.go")
	writeTestFile(t, target, "")

	output := captureStdout(t, func() {
		code := run([]string{"sctx", "context", target, "--on", "read"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})

	if !strings.Contains(output, "Read only context") {
		t.Errorf("expected read context, got: %s", output)
	}

	if strings.Contains(output, "Edit only context") {
		t.Errorf("should not contain edit context, got: %s", output)
	}
}

func TestCmdContext_WhenFilter(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, ".git"), "")
	writeTestFile(t, filepath.Join(dir, "AGENTS.yaml"), `
context:
  - content: "Before context"
    on: all
    when: before
  - content: "After context"
    on: all
    when: after
`)

	target := filepath.Join(dir, "file.go")
	writeTestFile(t, target, "")

	output := captureStdout(t, func() {
		code := run([]string{"sctx", "context", target, "--when", "after"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})

	if !strings.Contains(output, "After context") {
		t.Errorf("expected after context, got: %s", output)
	}

	if strings.Contains(output, "Before context") {
		t.Errorf("should not contain before context, got: %s", output)
	}
}

func TestCmdContext_CombinedFlags(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, ".git"), "")
	writeTestFile(t, filepath.Join(dir, "AGENTS.yaml"), `
context:
  - content: "Edit before"
    on: edit
    when: before
  - content: "Read after"
    on: read
    when: after
`)

	target := filepath.Join(dir, "file.go")
	writeTestFile(t, target, "")

	output := captureStdout(t, func() {
		code := run([]string{"sctx", "context", target, "--on", "edit", "--when", "before", "--json"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})

	if !strings.Contains(output, "Edit before") {
		t.Errorf("expected edit before context in JSON, got: %s", output)
	}

	if strings.Contains(output, "Read after") {
		t.Errorf("should not contain read after context, got: %s", output)
	}
}

func TestCmdContext_NoMatchingContext(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, ".git"), "")
	writeTestFile(t, filepath.Join(dir, "AGENTS.yaml"), `
context:
  - content: "Only after"
    on: all
    when: after
`)

	target := filepath.Join(dir, "file.go")
	writeTestFile(t, target, "")

	output := captureStdout(t, func() {
		code := run([]string{"sctx", "context", target, "--when", "before"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})

	if !strings.Contains(output, "No matching context found.") {
		t.Errorf("expected no-match message, got: %s", output)
	}
}

func TestCmdDecisions_MissingPath(t *testing.T) {
	stderr := captureStderr(t, func() {
		code := run([]string{"sctx", "decisions"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
	})

	if !strings.Contains(stderr, "missing required <path> argument") {
		t.Errorf("expected missing path error, got: %s", stderr)
	}
}

func TestCmdDecisions_TextOutput(t *testing.T) {
	dir := setupProject(t)
	target := filepath.Join(dir, "file.go")
	writeTestFile(t, target, "")

	output := captureStdout(t, func() {
		code := run([]string{"sctx", "decisions", target})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})

	if !strings.Contains(output, "Use Go") {
		t.Errorf("expected decision in output, got: %s", output)
	}

	if !strings.Contains(output, "Fast compilation") {
		t.Errorf("expected rationale in output, got: %s", output)
	}
}

func TestCmdDecisions_JSONOutput(t *testing.T) {
	dir := setupProject(t)
	target := filepath.Join(dir, "file.go")
	writeTestFile(t, target, "")

	output := captureStdout(t, func() {
		code := run([]string{"sctx", "decisions", target, "--json"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})

	if !strings.Contains(output, `"Decision"`) {
		t.Errorf("expected JSON with Decision field, got: %s", output)
	}

	if !strings.Contains(output, "Use Go") {
		t.Errorf("expected decision content in JSON, got: %s", output)
	}
}

func TestCmdDecisions_NoDecisions(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, ".git"), "")
	writeTestFile(t, filepath.Join(dir, "AGENTS.yaml"), `
context:
  - content: "Just context, no decisions"
`)

	target := filepath.Join(dir, "file.go")
	writeTestFile(t, target, "")

	output := captureStdout(t, func() {
		code := run([]string{"sctx", "decisions", target})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})

	if !strings.Contains(output, "No matching decisions found.") {
		t.Errorf("expected no-decisions message, got: %s", output)
	}
}

func TestCmdValidate_ValidTree(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "AGENTS.yaml"), `
context:
  - content: "Valid entry"
`)

	output := captureStdout(t, func() {
		code := run([]string{"sctx", "validate", dir})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})

	if !strings.Contains(output, "All context files are valid.") {
		t.Errorf("expected valid message, got: %s", output)
	}
}

func TestCmdValidate_InvalidTree(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "AGENTS.yaml"), `
context:
  - content: ""
`)

	code := run([]string{"sctx", "validate", dir})
	if code != 1 {
		t.Errorf("expected exit code 1 for invalid tree, got %d", code)
	}
}

func TestCmdValidate_DefaultsToCurrentDir(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "AGENTS.yaml"), `
context:
  - content: "Valid"
`)
	t.Chdir(dir)

	output := captureStdout(t, func() {
		code := run([]string{"sctx", "validate"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})

	if !strings.Contains(output, "All context files are valid.") {
		t.Errorf("expected valid message, got: %s", output)
	}
}

func TestCmdValidate_NonexistentDir(t *testing.T) {
	stderr := captureStderr(t, func() {
		code := run([]string{"sctx", "validate", "/nonexistent/path"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
	})

	if stderr == "" {
		t.Error("expected error output for nonexistent directory")
	}
}

func TestCmdInit_CreatesFile(t *testing.T) {
	t.Chdir(t.TempDir())

	output := captureStdout(t, func() {
		code := run([]string{"sctx", "init"})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})

	if !strings.Contains(output, "Created AGENTS.yaml") {
		t.Errorf("expected created message, got: %s", output)
	}

	data, err := os.ReadFile("AGENTS.yaml")
	if err != nil {
		t.Fatalf("failed to read created file: %v", err)
	}

	if !strings.Contains(string(data), "Structured Context") {
		t.Errorf("expected template content, got: %s", string(data))
	}
}

func TestCmdInit_FileAlreadyExists(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "AGENTS.yaml"), "existing")
	t.Chdir(dir)

	stderr := captureStderr(t, func() {
		code := run([]string{"sctx", "init"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
	})

	if !strings.Contains(stderr, "file already exists") {
		t.Errorf("expected file exists error, got: %s", stderr)
	}
}

func TestCmdClaude_MissingSubcommand(t *testing.T) {
	stderr := captureStderr(t, func() {
		code := run([]string{"sctx", "claude"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
	})

	if !strings.Contains(stderr, "sctx claude <enable|disable>") {
		t.Errorf("expected usage hint, got: %s", stderr)
	}
}

func TestCmdClaude_UnknownSubcommand(t *testing.T) {
	stderr := captureStderr(t, func() {
		code := run([]string{"sctx", "claude", "bogus"})
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
	})

	if !strings.Contains(stderr, "sctx claude <enable|disable>") {
		t.Errorf("expected usage hint, got: %s", stderr)
	}
}

func TestCmdClaude_EnableWithoutClaudeDir(t *testing.T) {
	t.Chdir(t.TempDir())

	code := run([]string{"sctx", "claude", "enable"})
	if code != 1 {
		t.Errorf("expected exit code 1 without .claude dir, got %d", code)
	}
}

func TestCmdClaude_DisableWithoutClaudeDir(t *testing.T) {
	t.Chdir(t.TempDir())

	code := run([]string{"sctx", "claude", "disable"})
	if code != 1 {
		t.Errorf("expected exit code 1 without .claude dir, got %d", code)
	}
}
