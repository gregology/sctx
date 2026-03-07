package core

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// testdataDir returns the absolute path to the testdata directory.
func testdataDir(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	return filepath.Join(filepath.Dir(filename), "testdata")
}

func TestMatchesGlobs(t *testing.T) {
	tests := []struct {
		name      string
		sourceDir string
		absPath   string
		match     []string
		exclude   []string
		want      bool
	}{
		{
			name:      "double star matches nested file",
			sourceDir: "/project",
			absPath:   "/project/src/api/handler.py",
			match:     []string{"**"},
			want:      true,
		},
		{
			name:      "extension glob matches",
			sourceDir: "/project",
			absPath:   "/project/src/api/handler.py",
			match:     []string{"**/*.py"},
			want:      true,
		},
		{
			name:      "extension mismatch",
			sourceDir: "/project",
			absPath:   "/project/src/handler.go",
			match:     []string{"**/*.py"},
			want:      false,
		},
		{
			name:      "exclude overrides match",
			sourceDir: "/project",
			absPath:   "/project/vendor/lib.py",
			match:     []string{"**/*.py"},
			exclude:   []string{"vendor/**"},
			want:      false,
		},
		{
			name:      "file outside source dir is ignored",
			sourceDir: "/project/src",
			absPath:   "/project/other/file.py",
			match:     []string{"**/*.py"},
			want:      false,
		},
		{
			name:      "directory-scoped match",
			sourceDir: "/project",
			absPath:   "/project/src/api/handler.py",
			match:     []string{"src/api/**"},
			want:      true,
		},
		{
			name:      "single star only matches one level",
			sourceDir: "/project",
			absPath:   "/project/src/deep/file.py",
			match:     []string{"*.py"},
			want:      false,
		},
		{
			name:      "single star matches direct child",
			sourceDir: "/project",
			absPath:   "/project/file.py",
			match:     []string{"*.py"},
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesGlobs(tt.sourceDir, tt.absPath, tt.match, tt.exclude)
			if got != tt.want {
				t.Errorf("matchesGlobs(%q, %q, %v, %v) = %v, want %v",
					tt.sourceDir, tt.absPath, tt.match, tt.exclude, got, tt.want)
			}
		})
	}
}

func TestMatchesAction(t *testing.T) {
	tests := []struct {
		name   string
		on     FlexList
		action Action
		want   bool
	}{
		{"all matches read", FlexList{"all"}, ActionRead, true},
		{"all matches edit", FlexList{"all"}, ActionEdit, true},
		{"all matches create", FlexList{"all"}, ActionCreate, true},
		{"exact match edit", FlexList{"edit"}, ActionEdit, true},
		{"edit does not match read", FlexList{"edit"}, ActionRead, false},
		{"multi-value list matches", FlexList{"edit", "create"}, ActionCreate, true},
		{"multi-value list rejects", FlexList{"edit", "create"}, ActionRead, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesAction(tt.on, tt.action)
			if got != tt.want {
				t.Errorf("matchesAction(%v, %q) = %v, want %v", tt.on, tt.action, got, tt.want)
			}
		})
	}
}

func TestResolve_EditBefore(t *testing.T) {
	td := testdataDir(t)
	target := filepath.Join(td, "project", "src", "api", "handler.py")

	result, _, err := Resolve(ResolveRequest{
		FilePath: target,
		Action:   ActionEdit,
		Timing:   TimingBefore,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	// Root "Use strict type annotations everywhere" (on:all, when:before)
	// + api "Validate input with pydantic" (on:[edit,create], when:before)
	wantContents := []string{
		"Use strict type annotations everywhere",
		"Validate input with pydantic",
	}
	assertContextContents(t, result.ContextEntries, wantContents)
}

func TestResolve_EditAfter(t *testing.T) {
	td := testdataDir(t)
	target := filepath.Join(td, "project", "src", "api", "handler.py")

	result, _, err := Resolve(ResolveRequest{
		FilePath: target,
		Action:   ActionEdit,
		Timing:   TimingAfter,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	// Root "Run tests after editing" (on:edit, when:after)
	// + api "API handlers must return typed response models" (on:edit, when:after, matches *.py, excludes *_test.py)
	wantContents := []string{
		"Run tests after editing",
		"API handlers must return typed response models",
	}
	assertContextContents(t, result.ContextEntries, wantContents)
}

func TestResolve_ExcludePattern(t *testing.T) {
	td := testdataDir(t)
	target := filepath.Join(td, "project", "src", "api", "handler_test.py")

	result, _, err := Resolve(ResolveRequest{
		FilePath: target,
		Action:   ActionEdit,
		Timing:   TimingAfter,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	// "API handlers must return typed response models" excludes *_test.py,
	// so only "Run tests after editing" should come through.
	wantContents := []string{"Run tests after editing"}
	assertContextContents(t, result.ContextEntries, wantContents)
}

func TestResolve_CreateAction(t *testing.T) {
	td := testdataDir(t)
	target := filepath.Join(td, "project", "src", "api", "new_handler.py")

	result, _, err := Resolve(ResolveRequest{
		FilePath: target,
		Action:   ActionCreate,
		Timing:   TimingBefore,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	// Root "Use strict type annotations everywhere" (on:all)
	// + Root "New files must have a module docstring" (on:create)
	// + api "Validate input with pydantic" (on:[edit,create])
	wantContents := []string{
		"Use strict type annotations everywhere",
		"New files must have a module docstring",
		"Validate input with pydantic",
	}
	assertContextContents(t, result.ContextEntries, wantContents)
}

func TestResolve_ReadDoesNotMatchEditOnly(t *testing.T) {
	td := testdataDir(t)
	target := filepath.Join(td, "project", "src", "api", "handler.py")

	result, _, err := Resolve(ResolveRequest{
		FilePath: target,
		Action:   ActionRead,
		Timing:   TimingBefore,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	// Only "Use strict type annotations everywhere" (on:all).
	// "Validate input with pydantic" is on:[edit,create] so it shouldn't appear.
	wantContents := []string{"Use strict type annotations everywhere"}
	assertContextContents(t, result.ContextEntries, wantContents)
}

func TestResolve_NonPythonFile(t *testing.T) {
	td := testdataDir(t)
	target := filepath.Join(td, "project", "README.md")

	result, _, err := Resolve(ResolveRequest{
		FilePath: target,
		Action:   ActionEdit,
		Timing:   TimingBefore,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	// Only root context (on:all, when:before). The api-level context
	// requires **/*.py so it won't match a markdown file.
	wantContents := []string{"Use strict type annotations everywhere"}
	assertContextContents(t, result.ContextEntries, wantContents)
}

func TestResolve_Decisions(t *testing.T) {
	td := testdataDir(t)
	target := filepath.Join(td, "project", "src", "api", "handler.py")

	result, _, err := Resolve(ResolveRequest{
		FilePath: target,
		Action:   ActionEdit,
		Timing:   TimingBefore,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if len(result.DecisionEntries) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(result.DecisionEntries))
	}
	if result.DecisionEntries[0].Decision != "Use ruff for linting" {
		t.Errorf("got decision %q", result.DecisionEntries[0].Decision)
	}
}

func TestResolve_MergesMultipleFileNames(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestFile(t, filepath.Join(tmpDir, ".git"), "")
	writeTestFile(t, filepath.Join(tmpDir, "CONTEXT.yaml"), `
context:
  - content: "From CONTEXT.yaml"
`)
	writeTestFile(t, filepath.Join(tmpDir, "AGENTS.yml"), `
context:
  - content: "From AGENTS.yml"
`)

	target := filepath.Join(tmpDir, "file.txt")
	writeTestFile(t, target, "")

	result, _, err := Resolve(ResolveRequest{
		FilePath: target,
		Action:   ActionRead,
		Timing:   TimingBefore,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	wantContents := []string{"From CONTEXT.yaml", "From AGENTS.yml"}
	assertContextContents(t, result.ContextEntries, wantContents)
}

func TestResolve_NoContextFiles(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestFile(t, filepath.Join(tmpDir, ".git"), "")

	target := filepath.Join(tmpDir, "file.txt")
	writeTestFile(t, target, "")

	result, warnings, err := Resolve(ResolveRequest{
		FilePath: target,
		Action:   ActionRead,
		Timing:   TimingBefore,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if len(result.ContextEntries) != 0 {
		t.Errorf("expected no context, got %d entries", len(result.ContextEntries))
	}

	foundWarning := false

	for _, w := range warnings {
		if w == "warning: no CONTEXT.yaml or AGENTS.yaml files found" {
			foundWarning = true
		}
	}

	if !foundWarning {
		t.Error("expected a warning about missing context files")
	}
}

// writeTestFile is a helper that writes a file and fails the test on error.
func writeTestFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func TestResolve_ParentBeforeChild(t *testing.T) {
	td := testdataDir(t)
	target := filepath.Join(td, "project", "src", "api", "handler.py")

	result, _, err := Resolve(ResolveRequest{
		FilePath: target,
		Action:   ActionEdit,
		Timing:   TimingBefore,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if len(result.ContextEntries) < 2 {
		t.Fatalf("need at least 2 entries to test ordering, got %d", len(result.ContextEntries))
	}

	// Root context should come before subdirectory context.
	if result.ContextEntries[0].Content != "Use strict type annotations everywhere" {
		t.Errorf("first entry should be from root, got %q", result.ContextEntries[0].Content)
	}
	if result.ContextEntries[1].Content != "Validate input with pydantic" {
		t.Errorf("second entry should be from subdirectory, got %q", result.ContextEntries[1].Content)
	}
}

// assertContextContents checks that the matched context entries have exactly
// the expected content strings, in order.
func assertContextContents(t *testing.T, got []MatchedContext, want []string) {
	t.Helper()
	if len(got) != len(want) {
		contents := make([]string, len(got))
		for i, g := range got {
			contents[i] = g.Content
		}
		t.Fatalf("expected %d context entries %v, got %d %v", len(want), want, len(got), contents)
	}
	for i := range want {
		if got[i].Content != want[i] {
			t.Errorf("entry[%d]: got %q, want %q", i, got[i].Content, want[i])
		}
	}
}
