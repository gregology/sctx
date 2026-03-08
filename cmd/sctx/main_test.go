package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func sctxBin(t *testing.T) string {
	t.Helper()
	bin := t.TempDir() + "/sctx"
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = "."
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return bin
}

func TestInvalidOnValue(t *testing.T) {
	bin := sctxBin(t)

	cmd := exec.Command(bin, "context", "foo.py", "--on", "banana")
	cmd.Env = append(os.Environ(), "HOME="+t.TempDir())
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit for invalid --on value")
	}

	got := string(out)
	if !strings.Contains(got, `invalid --on value "banana"`) {
		t.Errorf("expected error about invalid --on value, got: %s", got)
	}
}

func TestInvalidWhenValue(t *testing.T) {
	bin := sctxBin(t)

	cmd := exec.Command(bin, "context", "foo.py", "--when", "yesterday")
	cmd.Env = append(os.Environ(), "HOME="+t.TempDir())
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit for invalid --when value")
	}

	got := string(out)
	if !strings.Contains(got, `invalid --when value "yesterday"`) {
		t.Errorf("expected error about invalid --when value, got: %s", got)
	}
}

func TestValidOnAndWhenValues(t *testing.T) {
	bin := sctxBin(t)

	// Valid --on and --when should not produce a validation error.
	cmd := exec.Command(bin, "context", "foo.py", "--on", "read", "--when", "before")
	cmd.Env = append(os.Environ(), "HOME="+t.TempDir())
	out, err := cmd.CombinedOutput()
	got := string(out)

	// The command may fail for other reasons (e.g., no AGENTS.yaml), but it
	// should NOT fail with an "invalid --on" or "invalid --when" error.
	if err != nil && (strings.Contains(got, "invalid --on") || strings.Contains(got, "invalid --when")) {
		t.Errorf("valid values rejected: %s", got)
	}
}
