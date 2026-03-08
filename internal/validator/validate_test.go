package validator

import (
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

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
