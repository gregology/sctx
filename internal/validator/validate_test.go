package validator

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func testdataDir(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	return filepath.Join(filepath.Dir(filename), "testdata")
}

func TestValidateFile_Valid(t *testing.T) {
	path := filepath.Join(testdataDir(t), "valid.yaml")
	errors := ValidateFile(path)

	if len(errors) != 0 {
		for _, e := range errors {
			t.Errorf("unexpected: %s", e)
		}
	}
}

func TestValidateFile_Invalid(t *testing.T) {
	path := filepath.Join(testdataDir(t), "invalid.yaml")
	errors := ValidateFile(path)

	// Expecting: missing content, invalid action "nope", invalid when "yesterday",
	// missing decision, missing rationale.
	if len(errors) < 4 {
		t.Errorf("expected at least 4 errors, got %d", len(errors))
		for _, e := range errors {
			t.Logf("  %s", e)
		}
	}

	wantMessages := map[string]bool{
		"content is required":   false,
		"invalid action":        false,
		"invalid when":          false,
		"decision is required":  false,
		"rationale is required": false,
	}

	for _, e := range errors {
		for substr := range wantMessages {
			if containsStr(e.Message, substr) {
				wantMessages[substr] = true
			}
		}
	}

	for substr, found := range wantMessages {
		if !found {
			t.Errorf("expected an error containing %q", substr)
		}
	}
}

func TestValidateFile_BadYAML(t *testing.T) {
	path := filepath.Join(testdataDir(t), "bad_yaml.yaml")
	errors := ValidateFile(path)

	if len(errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errors))
	}
	if !containsStr(errors[0].Message, "invalid YAML") {
		t.Errorf("expected YAML parse error, got: %s", errors[0].Message)
	}
}

func TestValidateFile_NonexistentFile(t *testing.T) {
	errors := ValidateFile("/nonexistent/AGENTS.yaml")

	if len(errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errors))
	}
	if !containsStr(errors[0].Message, "cannot read file") {
		t.Errorf("expected read error, got: %s", errors[0].Message)
	}
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestValidateTree(t *testing.T) {
	validYAML := []byte("context:\n  - content: \"Use strict types\"\n")
	invalidYAML := []byte("context:\n  - content: \"\"\n")

	t.Run("multi-level tree with valid and invalid files", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "AGENTS.yaml"), validYAML)
		writeFile(t, filepath.Join(root, "sub", "dir", "AGENTS.yaml"), invalidYAML)

		errs, err := ValidateTree(root)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(errs) != 1 {
			t.Errorf("expected 1 validation error, got %d", len(errs))
			for _, e := range errs {
				t.Logf("  %s", e)
			}
		}
	})

	t.Run("no AGENTS files", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "README.md"), []byte("hello"))

		errs, err := ValidateTree(root)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(errs) != 0 {
			t.Errorf("expected 0 validation errors, got %d", len(errs))
		}
	})

	t.Run("both AGENTS.yaml and AGENTS.yml in same directory", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "AGENTS.yaml"), validYAML)
		writeFile(t, filepath.Join(root, "AGENTS.yml"), invalidYAML)

		errs, err := ValidateTree(root)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(errs) != 1 {
			t.Errorf("expected 1 validation error, got %d", len(errs))
			for _, e := range errs {
				t.Logf("  %s", e)
			}
		}
	})

	t.Run("unreadable directory returns error", func(t *testing.T) {
		root := t.TempDir()
		sub := filepath.Join(root, "noperm")
		writeFile(t, filepath.Join(sub, "AGENTS.yaml"), validYAML)

		if err := os.Chmod(sub, 0o000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(sub, 0o700) }) //nolint:gosec // restore perms for cleanup

		_, err := ValidateTree(root)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestValidateFile_WhenAll(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "AGENTS.yaml")
	writeFile(t, path, []byte("context:\n  - content: \"Always deliver\"\n    when: all\n"))

	errs := ValidateFile(path)
	if len(errs) != 0 {
		for _, e := range errs {
			t.Errorf("unexpected: %s", e)
		}
	}
}

func TestValidateFile_UnknownFields(t *testing.T) {
	path := filepath.Join(testdataDir(t), "unknown_fields.yaml")
	errs := ValidateFile(path)

	// Expect warnings for: "foo" (top level), "mach" (context[0]), "exlude" (context[1]),
	// "importanc" (decisions[0]), "scor" (decisions[0].alternatives[0])
	wantWarnings := []string{
		`top level: unknown field "foo"`,
		`context[0]: unknown field "mach"`,
		`context[1]: unknown field "exlude"`,
		`decisions[0]: unknown field "importanc"`,
		`decisions[0].alternatives[0]: unknown field "scor"`,
	}

	for _, w := range wantWarnings {
		found := false
		for _, e := range errs {
			if containsStr(e.Message, w) {
				if !e.IsWarn {
					t.Errorf("expected IsWarn=true for %q", w)
				}
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected warning containing %q", w)
		}
	}
}

func TestValidateFile_NoUnknownFieldWarnings(t *testing.T) {
	path := filepath.Join(testdataDir(t), "valid.yaml")
	errs := ValidateFile(path)

	for _, e := range errs {
		if e.IsWarn {
			t.Errorf("unexpected warning: %s", e)
		}
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
