package core

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"pgregory.net/rapid"
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

func TestResolve_EditBefore(t *testing.T) {
	td := testdataDir(t)
	root := filepath.Join(td, "project")
	target := filepath.Join(root, "src", "api", "handler.py")

	result, _, err := Resolve(ResolveRequest{
		FilePath: target,
		Action:   ActionEdit,
		Timing:   TimingBefore,
		Root:     root,
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
	root := filepath.Join(td, "project")
	target := filepath.Join(root, "src", "api", "handler.py")

	result, _, err := Resolve(ResolveRequest{
		FilePath: target,
		Action:   ActionEdit,
		Timing:   TimingAfter,
		Root:     root,
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
	root := filepath.Join(td, "project")
	target := filepath.Join(root, "src", "api", "handler_test.py")

	result, _, err := Resolve(ResolveRequest{
		FilePath: target,
		Action:   ActionEdit,
		Timing:   TimingAfter,
		Root:     root,
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
	root := filepath.Join(td, "project")
	target := filepath.Join(root, "src", "api", "new_handler.py")

	result, _, err := Resolve(ResolveRequest{
		FilePath: target,
		Action:   ActionCreate,
		Timing:   TimingBefore,
		Root:     root,
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
	root := filepath.Join(td, "project")
	target := filepath.Join(root, "src", "api", "handler.py")

	result, _, err := Resolve(ResolveRequest{
		FilePath: target,
		Action:   ActionRead,
		Timing:   TimingBefore,
		Root:     root,
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
	root := filepath.Join(td, "project")
	target := filepath.Join(root, "README.md")

	result, _, err := Resolve(ResolveRequest{
		FilePath: target,
		Action:   ActionEdit,
		Timing:   TimingBefore,
		Root:     root,
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
	root := filepath.Join(td, "project")
	target := filepath.Join(root, "src", "api", "handler.py")

	result, _, err := Resolve(ResolveRequest{
		FilePath: target,
		Action:   ActionEdit,
		Timing:   TimingBefore,
		Root:     root,
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
	writeTestFile(t, filepath.Join(tmpDir, "AGENTS.yaml"), `
context:
  - content: "From AGENTS.yaml"
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
		Root:     tmpDir,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	wantContents := []string{"From AGENTS.yaml", "From AGENTS.yml"}
	assertContextContents(t, result.ContextEntries, wantContents)
}

func TestResolve_DefaultWhenIsBefore(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestFile(t, filepath.Join(tmpDir, "AGENTS.yaml"), `
context:
  - content: "no explicit when"
`)

	target := filepath.Join(tmpDir, "file.txt")
	writeTestFile(t, target, "")

	// Omitting when: should default to "before", so TimingBefore resolves the entry.
	result, _, err := Resolve(ResolveRequest{
		FilePath: target,
		Action:   ActionRead,
		Timing:   TimingBefore,
		Root:     tmpDir,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	assertContextContents(t, result.ContextEntries, []string{"no explicit when"})

	// TimingAfter must NOT match the default "before".
	result, _, err = Resolve(ResolveRequest{
		FilePath: target,
		Action:   ActionRead,
		Timing:   TimingAfter,
		Root:     tmpDir,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if len(result.ContextEntries) != 0 {
		t.Errorf("expected no entries for TimingAfter with default when, got %d", len(result.ContextEntries))
	}
}

func TestResolve_NoContextFiles(t *testing.T) {
	tmpDir := t.TempDir()

	target := filepath.Join(tmpDir, "file.txt")
	writeTestFile(t, target, "")

	result, warnings, err := Resolve(ResolveRequest{
		FilePath: target,
		Action:   ActionRead,
		Timing:   TimingBefore,
		Root:     tmpDir,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if len(result.ContextEntries) != 0 {
		t.Errorf("expected no context, got %d entries", len(result.ContextEntries))
	}

	foundWarning := false

	for _, w := range warnings {
		if w == "warning: no AGENTS.yaml files found" {
			foundWarning = true
		}
	}

	if !foundWarning {
		t.Error("expected a warning about missing context files")
	}
}

func TestResolve_AllActionAllTimingReturnsEverything(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestFile(t, filepath.Join(tmpDir, "AGENTS.yaml"), `
context:
  - content: "before-all"
    on: all
    when: before
  - content: "after-edit"
    on: edit
    when: after
  - content: "before-create"
    on: create
    when: before
  - content: "after-read"
    on: read
    when: after
`)

	target := filepath.Join(tmpDir, "file.go")
	writeTestFile(t, target, "")

	result, _, err := Resolve(ResolveRequest{
		FilePath: target,
		Action:   ActionAll,
		Timing:   TimingAll,
		Root:     tmpDir,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	assertContextContents(t, result.ContextEntries, []string{
		"before-all",
		"after-edit",
		"before-create",
		"after-read",
	})
}

func TestResolve_AllActionSpecificTiming(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestFile(t, filepath.Join(tmpDir, "AGENTS.yaml"), `
context:
  - content: "before-all"
    on: all
    when: before
  - content: "after-edit"
    on: edit
    when: after
  - content: "before-create"
    on: create
    when: before
`)

	target := filepath.Join(tmpDir, "file.go")
	writeTestFile(t, target, "")

	// ActionAll + TimingBefore should return all actions but only before timing.
	result, _, err := Resolve(ResolveRequest{
		FilePath: target,
		Action:   ActionAll,
		Timing:   TimingBefore,
		Root:     tmpDir,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	assertContextContents(t, result.ContextEntries, []string{
		"before-all",
		"before-create",
	})
}

func TestResolve_SpecificActionAllTiming(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestFile(t, filepath.Join(tmpDir, "AGENTS.yaml"), `
context:
  - content: "before-all"
    on: all
    when: before
  - content: "after-edit"
    on: edit
    when: after
  - content: "before-create"
    on: create
    when: before
`)

	target := filepath.Join(tmpDir, "file.go")
	writeTestFile(t, target, "")

	// ActionEdit + TimingAll should return edit-matching entries for all timings.
	result, _, err := Resolve(ResolveRequest{
		FilePath: target,
		Action:   ActionEdit,
		Timing:   TimingAll,
		Root:     tmpDir,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	assertContextContents(t, result.ContextEntries, []string{
		"before-all",
		"after-edit",
	})
}

func TestResolve_WhenAllMatchesAnyTiming(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestFile(t, filepath.Join(tmpDir, "AGENTS.yaml"), `
context:
  - content: "always-deliver"
    on: all
    when: all
`)

	target := filepath.Join(tmpDir, "file.go")
	writeTestFile(t, target, "")

	tests := []struct {
		name   string
		timing Timing
	}{
		{"before", TimingBefore},
		{"after", TimingAfter},
		{"all", TimingAll},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := Resolve(ResolveRequest{
				FilePath: target,
				Action:   ActionEdit,
				Timing:   tt.timing,
				Root:     tmpDir,
			})
			if err != nil {
				t.Fatalf("Resolve() error: %v", err)
			}

			assertContextContents(t, result.ContextEntries, []string{"always-deliver"})
		})
	}
}

// writeTestFile is a helper that writes a file and fails the test on error.
func writeTestFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func TestResolve_ContextAboveRootIsIgnored(t *testing.T) {
	parentDir := t.TempDir()
	writeTestFile(t, filepath.Join(parentDir, "AGENTS.yaml"), `
context:
  - content: "From above root"
`)

	childDir := filepath.Join(parentDir, "child")
	if err := os.MkdirAll(childDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	writeTestFile(t, filepath.Join(childDir, "AGENTS.yaml"), `
context:
  - content: "From child root"
`)

	target := filepath.Join(childDir, "file.txt")
	writeTestFile(t, target, "")

	result, _, err := Resolve(ResolveRequest{
		FilePath: target,
		Action:   ActionAll,
		Timing:   TimingAll,
		Root:     childDir,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	assertContextContents(t, result.ContextEntries, []string{"From child root"})
}

func TestResolve_ParentBeforeChild(t *testing.T) {
	td := testdataDir(t)
	root := filepath.Join(td, "project")
	target := filepath.Join(root, "src", "api", "handler.py")

	result, _, err := Resolve(ResolveRequest{
		FilePath: target,
		Action:   ActionEdit,
		Timing:   TimingBefore,
		Root:     root,
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

// genAction generates a random valid Action.
func genAction(t *rapid.T) Action {
	return rapid.SampledFrom([]Action{ActionRead, ActionEdit, ActionCreate, ActionAll}).Draw(t, "action")
}

// genTiming generates a random valid Timing.
func genTiming(t *rapid.T) Timing {
	return rapid.SampledFrom([]Timing{TimingBefore, TimingAfter, TimingAll}).Draw(t, "timing")
}

// genDirName generates a short directory name safe for filesystem use.
func genDirName(t *rapid.T) string {
	return rapid.StringMatching(`[a-z]{1,8}`).Draw(t, "dirname")
}

// genGlob generates a random glob pattern.
func genGlob(t *rapid.T) string {
	return rapid.SampledFrom([]string{
		"**", "**/*.go", "**/*.py", "*.txt", "src/**", "**/*.js",
		"docs/**", "*.md", "**/*_test.go", "vendor/**",
	}).Draw(t, "glob")
}

// genOnValue generates a random on value.
func genOnValue(t *rapid.T) string {
	return rapid.SampledFrom([]string{"read", "edit", "create", "all"}).Draw(t, "on")
}

// genWhenValue generates a random when value.
func genWhenValue(t *rapid.T) string {
	return rapid.SampledFrom([]string{"before", "after", "all"}).Draw(t, "when")
}

// writeAgentsYAML writes an AGENTS.yaml with the given context entries to dir.
func writeAgentsYAML(t *testing.T, dir string, entries []ContextEntry) {
	t.Helper()

	var b strings.Builder
	b.WriteString("context:\n")

	for _, e := range entries {
		fmt.Fprintf(&b, "  - content: %q\n", e.Content)

		if len(e.Match) > 0 {
			b.WriteString("    match:\n")
			for _, m := range e.Match {
				fmt.Fprintf(&b, "      - %q\n", m)
			}
		}

		if len(e.Exclude) > 0 {
			b.WriteString("    exclude:\n")
			for _, ex := range e.Exclude {
				fmt.Fprintf(&b, "      - %q\n", ex)
			}
		}

		if len(e.On) > 0 {
			b.WriteString("    on:\n")
			for _, o := range e.On {
				fmt.Fprintf(&b, "      - %s\n", o)
			}
		}

		if e.When != "" {
			fmt.Fprintf(&b, "    when: %s\n", e.When)
		}
	}

	writeTestFile(t, filepath.Join(dir, "AGENTS.yaml"), b.String())
}

func TestResolve_NeverPanics(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		tmpDir := t.TempDir()

		depth := rapid.IntRange(0, 3).Draw(rt, "depth")
		dir := tmpDir

		for i := range depth {
			dir = filepath.Join(dir, genDirName(rt))
			if err := os.MkdirAll(dir, 0o750); err != nil {
				t.Fatalf("mkdir: %v", err)
			}

			numEntries := rapid.IntRange(0, 4).Draw(rt, "numEntries")
			var entries []ContextEntry

			for j := range numEntries {
				numMatch := rapid.IntRange(0, 3).Draw(rt, "numMatch")
				var match []string
				for range numMatch {
					match = append(match, genGlob(rt))
				}

				numExclude := rapid.IntRange(0, 2).Draw(rt, "numExclude")
				var exclude []string
				for range numExclude {
					exclude = append(exclude, genGlob(rt))
				}

				numOn := rapid.IntRange(1, 3).Draw(rt, "numOn")
				var on FlexList
				for range numOn {
					on = append(on, genOnValue(rt))
				}

				entries = append(entries, ContextEntry{
					Content: fmt.Sprintf("content-%d-%d", i, j),
					Match:   match,
					Exclude: exclude,
					On:      on,
					When:    genWhenValue(rt),
				})
			}

			if len(entries) > 0 {
				writeAgentsYAML(t, dir, entries)
			}
		}

		target := filepath.Join(dir, "target.go")
		writeTestFile(t, target, "")

		action := genAction(rt)
		timing := genTiming(rt)

		// Must not panic.
		_, _, _ = Resolve(ResolveRequest{
			FilePath: target,
			Action:   action,
			Timing:   timing,
			Root:     tmpDir,
		})
	})
}

func TestResolve_ChildMergesWithParent(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		tmpDir := t.TempDir()

		parentContent := rapid.StringMatching(`[a-z]{5,15}`).Draw(rt, "parentContent")
		childContent := rapid.StringMatching(`[a-z]{5,15}`).Draw(rt, "childContent")

		writeAgentsYAML(t, tmpDir, []ContextEntry{
			{Content: parentContent, Match: []string{"**"}, On: FlexList{"all"}, When: "before"},
		})

		childDir := filepath.Join(tmpDir, genDirName(rt))
		if err := os.MkdirAll(childDir, 0o750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		writeAgentsYAML(t, childDir, []ContextEntry{
			{Content: childContent, Match: []string{"**"}, On: FlexList{"all"}, When: "before"},
		})

		target := filepath.Join(childDir, "file.txt")
		writeTestFile(t, target, "")

		result, _, err := Resolve(ResolveRequest{
			FilePath: target,
			Action:   ActionRead,
			Timing:   TimingBefore,
			Root:     tmpDir,
		})
		if err != nil {
			t.Fatalf("Resolve() error: %v", err)
		}

		foundParent := false
		foundChild := false

		for _, e := range result.ContextEntries {
			if e.Content == parentContent {
				foundParent = true
			}
			if e.Content == childContent {
				foundChild = true
			}
		}

		if !foundParent {
			t.Errorf("parent entry %q not found in results", parentContent)
		}
		if !foundChild {
			t.Errorf("child entry %q not found in results", childContent)
		}
	})
}

func TestResolve_ExcludeOverridesMatch(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		tmpDir := t.TempDir()

		ext := rapid.SampledFrom([]string{".go", ".py", ".js", ".txt", ".md"}).Draw(rt, "ext")
		globPattern := "**/*" + ext
		excludeContent := "excluded-content"

		writeAgentsYAML(t, tmpDir, []ContextEntry{
			{
				Content: excludeContent,
				Match:   []string{globPattern},
				Exclude: []string{globPattern},
				On:      FlexList{"all"},
				When:    "before",
			},
		})

		target := filepath.Join(tmpDir, "somefile"+ext)
		writeTestFile(t, target, "")

		result, _, err := Resolve(ResolveRequest{
			FilePath: target,
			Action:   ActionRead,
			Timing:   TimingBefore,
			Root:     tmpDir,
		})
		if err != nil {
			t.Fatalf("Resolve() error: %v", err)
		}

		for _, e := range result.ContextEntries {
			if e.Content == excludeContent {
				t.Errorf("excluded content %q should not appear in results", excludeContent)
			}
		}
	})
}

func TestResolve_EditEntriesFilteredForActionRead(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		tmpDir := t.TempDir()

		editContent := rapid.StringMatching(`editonly-[a-z]{5,10}`).Draw(rt, "editContent")
		when := genWhenValue(rt)

		entries := []ContextEntry{
			{Content: editContent, Match: []string{"**"}, On: FlexList{"edit"}, When: when},
		}

		numExtra := rapid.IntRange(0, 3).Draw(rt, "numExtra")
		for i := range numExtra {
			entries = append(entries, ContextEntry{
				Content: fmt.Sprintf("extra-%d", i),
				Match:   []string{"**"},
				On:      FlexList{genOnValue(rt)},
				When:    genWhenValue(rt),
			})
		}

		writeAgentsYAML(t, tmpDir, entries)

		target := filepath.Join(tmpDir, "file.go")
		writeTestFile(t, target, "")

		result, _, err := Resolve(ResolveRequest{
			FilePath: target,
			Action:   ActionRead,
			Timing:   Timing(when),
			Root:     tmpDir,
		})
		if err != nil {
			t.Fatalf("Resolve() error: %v", err)
		}

		for _, e := range result.ContextEntries {
			if e.Content == editContent {
				t.Errorf("edit-only entry %q appeared for ActionRead request", editContent)
			}
		}
	})
}

func TestResolve_ParentBeforeChildOrdering(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		tmpDir := t.TempDir()

		depth := rapid.IntRange(1, 4).Draw(rt, "depth")
		dirs := []string{tmpDir}
		dir := tmpDir

		for range depth {
			dir = filepath.Join(dir, genDirName(rt))
			if err := os.MkdirAll(dir, 0o750); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			dirs = append(dirs, dir)
		}

		for i, d := range dirs {
			writeAgentsYAML(t, d, []ContextEntry{
				{Content: fmt.Sprintf("level-%d", i), Match: []string{"**"}, On: FlexList{"all"}, When: "before"},
			})
		}

		target := filepath.Join(dir, "file.txt")
		writeTestFile(t, target, "")

		result, _, err := Resolve(ResolveRequest{
			FilePath: target,
			Action:   ActionRead,
			Timing:   TimingBefore,
			Root:     tmpDir,
		})
		if err != nil {
			t.Fatalf("Resolve() error: %v", err)
		}

		if len(result.ContextEntries) != len(dirs) {
			t.Fatalf("expected %d entries, got %d", len(dirs), len(result.ContextEntries))
		}

		for i, e := range result.ContextEntries {
			want := fmt.Sprintf("level-%d", i)
			if e.Content != want {
				t.Errorf("entry[%d]: got %q, want %q", i, e.Content, want)
			}
		}
	})
}

func TestResolve_MalformedYAMLGracefulDegradation(t *testing.T) {
	const (
		validYAML   = "context:\n  - content: \"Valid context\"\n"
		invalidYAML = `": invalid: [yaml`
	)

	tests := []struct {
		name      string
		rootYAML  string
		childYAML string
		wantCtx   []string
		wantBadIn string // "root" or "child" — which dir should appear in the warning
	}{
		{
			name:      "malformed parent still resolves child",
			rootYAML:  invalidYAML,
			childYAML: validYAML,
			wantCtx:   []string{"Valid context"},
			wantBadIn: "root",
		},
		{
			name:      "malformed child still resolves parent",
			rootYAML:  validYAML,
			childYAML: invalidYAML,
			wantCtx:   []string{"Valid context"},
			wantBadIn: "child",
		},
		{
			name:      "map typed on field still resolves sibling",
			rootYAML:  validYAML,
			childYAML: "context:\n  - content: \"Bad on\"\n    on: {read: true}\n",
			wantCtx:   []string{"Valid context"},
			wantBadIn: "child",
		},
		{
			name:      "nested sequence on field still resolves sibling",
			rootYAML:  "context:\n  - content: \"Bad on\"\n    on: [[nested]]\n",
			childYAML: validYAML,
			wantCtx:   []string{"Valid context"},
			wantBadIn: "root",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			writeTestFile(t, filepath.Join(tmpDir, "AGENTS.yaml"), tt.rootYAML)

			childDir := filepath.Join(tmpDir, "child")
			if err := os.MkdirAll(childDir, 0o750); err != nil {
				t.Fatalf("mkdir: %v", err)
			}

			writeTestFile(t, filepath.Join(childDir, "AGENTS.yaml"), tt.childYAML)

			target := filepath.Join(childDir, "file.txt")
			writeTestFile(t, target, "")

			result, warnings, err := Resolve(ResolveRequest{
				FilePath: target,
				Action:   ActionRead,
				Timing:   TimingBefore,
				Root:     tmpDir,
			})
			if err != nil {
				t.Fatalf("Resolve() error: %v", err)
			}

			assertContextContents(t, result.ContextEntries, tt.wantCtx)

			badDir := tmpDir
			if tt.wantBadIn == "child" {
				badDir = childDir
			}

			badFile := filepath.Join(badDir, "AGENTS.yaml")
			foundWarning := false

			for _, w := range warnings {
				if strings.Contains(w, "failed to parse") && strings.Contains(w, badFile) {
					foundWarning = true
				}
			}

			if !foundWarning {
				t.Errorf("expected warning about malformed %s, got warnings: %v", badFile, warnings)
			}
		})
	}
}

func TestResolve_DecisionsNotMatchedForWrongFileType(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestFile(t, filepath.Join(tmpDir, "AGENTS.yaml"), `
decisions:
  - decision: "Use ruff for linting"
    rationale: "Fast, replaces flake8+isort+pycodestyle"
    match: ["**/*.py"]
    date: 2026-03-06
`)

	target := filepath.Join(tmpDir, "main.go")
	writeTestFile(t, target, "")

	result, _, err := Resolve(ResolveRequest{
		FilePath: target,
		Action:   ActionEdit,
		Timing:   TimingBefore,
		Root:     tmpDir,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if len(result.DecisionEntries) != 0 {
		t.Errorf("expected 0 decisions for .go file, got %d: %v",
			len(result.DecisionEntries), result.DecisionEntries)
	}
}

func TestResolve_DecisionsDefaultMatch(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestFile(t, filepath.Join(tmpDir, "AGENTS.yaml"), `
decisions:
  - decision: "YAML for config format"
    rationale: "Human readable"
    date: 2026-03-06
`)

	target := filepath.Join(tmpDir, "anything.rs")
	writeTestFile(t, target, "")

	result, _, err := Resolve(ResolveRequest{
		FilePath: target,
		Action:   ActionRead,
		Timing:   TimingBefore,
		Root:     tmpDir,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if len(result.DecisionEntries) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(result.DecisionEntries))
	}
	if result.DecisionEntries[0].Decision != "YAML for config format" {
		t.Errorf("got decision %q", result.DecisionEntries[0].Decision)
	}
}

func TestResolve_DecisionsMergeParentAndChild(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestFile(t, filepath.Join(tmpDir, "AGENTS.yaml"), `
decisions:
  - decision: "Go for implementation"
    rationale: "Fast compile"
    date: 2026-03-06
`)

	childDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(childDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	writeTestFile(t, filepath.Join(childDir, "AGENTS.yaml"), `
decisions:
  - decision: "Table-driven tests"
    rationale: "Idiomatic Go"
    date: 2026-03-06
`)

	target := filepath.Join(childDir, "main.go")
	writeTestFile(t, target, "")

	result, _, err := Resolve(ResolveRequest{
		FilePath: target,
		Action:   ActionEdit,
		Timing:   TimingBefore,
		Root:     tmpDir,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if len(result.DecisionEntries) != 2 {
		t.Fatalf("expected 2 decisions, got %d", len(result.DecisionEntries))
	}
	if result.DecisionEntries[0].Decision != "Go for implementation" {
		t.Errorf("first decision should be from parent, got %q", result.DecisionEntries[0].Decision)
	}
	if result.DecisionEntries[1].Decision != "Table-driven tests" {
		t.Errorf("second decision should be from child, got %q", result.DecisionEntries[1].Decision)
	}
}

func TestResolve_ScopedDecisionExcludesNonMatchingFiles(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		tmpDir := t.TempDir()

		ext := rapid.SampledFrom([]string{".go", ".py", ".js", ".txt", ".md"}).Draw(rt, "ext")
		otherExt := rapid.SampledFrom([]string{".rs", ".rb", ".java", ".cpp", ".zig"}).Draw(rt, "otherExt")
		glob := "**/*" + ext
		decisionText := "scoped-decision"

		writeTestFile(t, filepath.Join(tmpDir, "AGENTS.yaml"), fmt.Sprintf(`
decisions:
  - decision: %q
    rationale: "test"
    match: [%q]
    date: 2026-03-06
`, decisionText, glob))

		target := filepath.Join(tmpDir, "somefile"+otherExt)
		writeTestFile(t, target, "")

		result, _, err := Resolve(ResolveRequest{
			FilePath: target,
			Action:   ActionRead,
			Timing:   TimingBefore,
			Root:     tmpDir,
		})
		if err != nil {
			t.Fatalf("Resolve() error: %v", err)
		}

		for _, d := range result.DecisionEntries {
			if d.Decision == decisionText {
				t.Errorf("decision %q with match %q should not appear for file %q",
					decisionText, glob, target)
			}
		}
	})
}

// --- Directory query tests ---

func TestResolve_DirQuery_TrailingSlashPattern(t *testing.T) {
	tmpDir := t.TempDir()
	testsDir := filepath.Join(tmpDir, "tests")
	if err := os.MkdirAll(testsDir, 0o750); err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, filepath.Join(tmpDir, "AGENTS.yaml"), `
context:
  - content: "dir-scoped to tests"
    match: ["tests/"]
    on: all
    when: before
  - content: "file-scoped to tests"
    match: ["tests/**"]
    on: all
    when: before
`)

	// Directory query for tests/ should match the trailing-slash pattern.
	result, _, err := Resolve(ResolveRequest{
		DirPath: testsDir,
		Action:  ActionAll,
		Timing:  TimingAll,
		Root:    tmpDir,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	assertContextContents(t, result.ContextEntries, []string{
		"dir-scoped to tests",
		"file-scoped to tests",
	})
}

func TestResolve_DirQuery_TrailingSlashDoesNotMatchSubdir(t *testing.T) {
	tmpDir := t.TempDir()
	testsDir := filepath.Join(tmpDir, "tests")
	unitDir := filepath.Join(testsDir, "unit")
	if err := os.MkdirAll(unitDir, 0o750); err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, filepath.Join(tmpDir, "AGENTS.yaml"), `
context:
  - content: "dir-scoped to tests"
    match: ["tests/"]
    on: all
    when: before
`)

	// Directory query for tests/unit/ should NOT match "tests/".
	result, _, err := Resolve(ResolveRequest{
		DirPath: unitDir,
		Action:  ActionAll,
		Timing:  TimingAll,
		Root:    tmpDir,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if len(result.ContextEntries) != 0 {
		t.Errorf("expected 0 entries for subdir query, got %d: %v",
			len(result.ContextEntries), result.ContextEntries)
	}
}

func TestResolve_DirQuery_TrailingSlashDoesNotMatchFileQuery(t *testing.T) {
	tmpDir := t.TempDir()
	testsDir := filepath.Join(tmpDir, "tests")
	if err := os.MkdirAll(testsDir, 0o750); err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, filepath.Join(tmpDir, "AGENTS.yaml"), `
context:
  - content: "dir-scoped to tests"
    match: ["tests/"]
    on: all
    when: before
`)

	target := filepath.Join(testsDir, "conftest.py")
	writeTestFile(t, target, "")

	// File query should NOT match "tests/" pattern.
	result, _, err := Resolve(ResolveRequest{
		FilePath: target,
		Action:   ActionAll,
		Timing:   TimingAll,
		Root:     tmpDir,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if len(result.ContextEntries) != 0 {
		t.Errorf("expected 0 entries for file query against dir pattern, got %d", len(result.ContextEntries))
	}
}

func TestResolve_DirQuery_GlobWithDoubleStarSlash(t *testing.T) {
	tmpDir := t.TempDir()
	fooTests := filepath.Join(tmpDir, "foo", "bar", "tests")
	bazTests := filepath.Join(tmpDir, "foo", "baz", "tests")
	fooBar := filepath.Join(tmpDir, "foo", "bar")
	for _, d := range []string{fooTests, bazTests} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatal(err)
		}
	}

	writeTestFile(t, filepath.Join(tmpDir, "AGENTS.yaml"), `
context:
  - content: "any tests dir under foo"
    match: ["foo/**/tests/"]
    on: all
    when: before
`)

	tests := []struct {
		name  string
		dir   string
		wantN int
	}{
		{"foo/bar/tests matches", fooTests, 1},
		{"foo/baz/tests matches", bazTests, 1},
		{"foo/bar does not match", fooBar, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := Resolve(ResolveRequest{
				DirPath: tt.dir,
				Action:  ActionAll,
				Timing:  TimingAll,
				Root:    tmpDir,
			})
			if err != nil {
				t.Fatalf("Resolve() error: %v", err)
			}

			if len(result.ContextEntries) != tt.wantN {
				t.Errorf("expected %d entries, got %d", tt.wantN, len(result.ContextEntries))
			}
		})
	}
}

func TestResolve_DirQuery_FileGlobMatchesContainingDir(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	testsDir := filepath.Join(tmpDir, "tests")
	for _, d := range []string{srcDir, testsDir} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatal(err)
		}
	}

	writeTestFile(t, filepath.Join(tmpDir, "AGENTS.yaml"), `
context:
  - content: "all python files"
    match: ["**/*.py"]
    on: all
    when: before
  - content: "src only"
    match: ["src/**"]
    on: all
    when: before
`)

	tests := []struct {
		name string
		dir  string
		want []string
	}{
		{"src gets both", srcDir, []string{"all python files", "src only"}},
		{"tests gets wildcard only", testsDir, []string{"all python files"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := Resolve(ResolveRequest{
				DirPath: tt.dir,
				Action:  ActionAll,
				Timing:  TimingAll,
				Root:    tmpDir,
			})
			if err != nil {
				t.Fatalf("Resolve() error: %v", err)
			}

			assertContextContents(t, result.ContextEntries, tt.want)
		})
	}
}

func TestResolve_DirQuery_DecisionsWithTrailingSlash(t *testing.T) {
	tmpDir := t.TempDir()
	apiDir := filepath.Join(tmpDir, "src", "api")
	modelsDir := filepath.Join(tmpDir, "src", "models")
	for _, d := range []string{apiDir, modelsDir} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatal(err)
		}
	}

	writeTestFile(t, filepath.Join(tmpDir, "AGENTS.yaml"), `
decisions:
  - decision: "REST over GraphQL"
    rationale: "Team expertise"
    match: ["src/api/"]
  - decision: "PostgreSQL over DynamoDB"
    rationale: "JSONB support"
`)

	// api dir gets both (dir-scoped + default **)
	result, _, err := Resolve(ResolveRequest{
		DirPath: apiDir,
		Action:  ActionAll,
		Timing:  TimingAll,
		Root:    tmpDir,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if len(result.DecisionEntries) != 2 {
		t.Fatalf("expected 2 decisions for api dir, got %d", len(result.DecisionEntries))
	}

	// models dir only gets the default ** decision
	result, _, err = Resolve(ResolveRequest{
		DirPath: modelsDir,
		Action:  ActionAll,
		Timing:  TimingAll,
		Root:    tmpDir,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if len(result.DecisionEntries) != 1 {
		t.Fatalf("expected 1 decision for models dir, got %d", len(result.DecisionEntries))
	}
	if result.DecisionEntries[0].Decision != "PostgreSQL over DynamoDB" {
		t.Errorf("wrong decision: %q", result.DecisionEntries[0].Decision)
	}
}

func TestResolve_DirQuery_MutuallyExclusive(t *testing.T) {
	_, _, err := Resolve(ResolveRequest{
		FilePath: "/some/file.go",
		DirPath:  "/some/dir",
		Root:     "/some",
	})
	if err == nil {
		t.Fatal("expected error when both FilePath and DirPath are set")
	}
}

func TestResolve_DirQuery_SelfDirectory(t *testing.T) {
	// Query the directory that contains the AGENTS.yaml itself.
	tmpDir := t.TempDir()
	writeTestFile(t, filepath.Join(tmpDir, "AGENTS.yaml"), `
context:
  - content: "applies to everything"
    on: all
    when: before
decisions:
  - decision: "some decision"
    rationale: "some reason"
`)

	result, _, err := Resolve(ResolveRequest{
		DirPath: tmpDir,
		Action:  ActionAll,
		Timing:  TimingAll,
		Root:    tmpDir,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	assertContextContents(t, result.ContextEntries, []string{"applies to everything"})
	if len(result.DecisionEntries) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(result.DecisionEntries))
	}
}

func TestResolve_DirQuery_ParentMergesWithChild(t *testing.T) {
	tmpDir := t.TempDir()
	childDir := filepath.Join(tmpDir, "child")
	if err := os.MkdirAll(childDir, 0o750); err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, filepath.Join(tmpDir, "AGENTS.yaml"), `
context:
  - content: "from parent"
`)
	writeTestFile(t, filepath.Join(childDir, "AGENTS.yaml"), `
context:
  - content: "from child"
`)

	result, _, err := Resolve(ResolveRequest{
		DirPath: childDir,
		Action:  ActionAll,
		Timing:  TimingAll,
		Root:    tmpDir,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	assertContextContents(t, result.ContextEntries, []string{"from parent", "from child"})
}

func TestResolve_DirQuery_ActionAndTimingFiltering(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestFile(t, filepath.Join(tmpDir, "AGENTS.yaml"), `
context:
  - content: "edit-before"
    on: edit
    when: before
  - content: "read-after"
    on: read
    when: after
  - content: "all-all"
    on: all
    when: all
`)

	result, _, err := Resolve(ResolveRequest{
		DirPath: tmpDir,
		Action:  ActionEdit,
		Timing:  TimingBefore,
		Root:    tmpDir,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	assertContextContents(t, result.ContextEntries, []string{"edit-before", "all-all"})
}

func TestResolve_DirQuery_ExcludeWithTrailingSlash(t *testing.T) {
	tmpDir := t.TempDir()
	vendorDir := filepath.Join(tmpDir, "vendor")
	srcDir := filepath.Join(tmpDir, "src")
	for _, d := range []string{vendorDir, srcDir} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatal(err)
		}
	}

	writeTestFile(t, filepath.Join(tmpDir, "AGENTS.yaml"), `
context:
  - content: "everywhere except vendor"
    exclude: ["vendor/"]
`)

	tests := []struct {
		name  string
		dir   string
		wantN int
	}{
		{"src/ not excluded", srcDir, 1},
		{"vendor/ excluded", vendorDir, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := Resolve(ResolveRequest{
				DirPath: tt.dir,
				Action:  ActionAll,
				Timing:  TimingAll,
				Root:    tmpDir,
			})
			if err != nil {
				t.Fatalf("Resolve() error: %v", err)
			}
			if len(result.ContextEntries) != tt.wantN {
				t.Errorf("expected %d entries, got %d", tt.wantN, len(result.ContextEntries))
			}
		})
	}
}

func TestResolve_DirQuery_SingleStarPattern(t *testing.T) {
	tmpDir := t.TempDir()
	fooDir := filepath.Join(tmpDir, "foo")
	fooBarDir := filepath.Join(tmpDir, "foo", "bar")
	for _, d := range []string{fooDir, fooBarDir} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatal(err)
		}
	}

	writeTestFile(t, filepath.Join(tmpDir, "AGENTS.yaml"), `
context:
  - content: "direct children only"
    match: ["foo/*"]
`)

	// foo/ should match (foo/* can match files in foo/)
	result, _, err := Resolve(ResolveRequest{
		DirPath: fooDir,
		Action:  ActionAll,
		Timing:  TimingAll,
		Root:    tmpDir,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	assertContextContents(t, result.ContextEntries, []string{"direct children only"})

	// foo/bar/ should NOT match (foo/* only matches direct children)
	result, _, err = Resolve(ResolveRequest{
		DirPath: fooBarDir,
		Action:  ActionAll,
		Timing:  TimingAll,
		Root:    tmpDir,
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	if len(result.ContextEntries) != 0 {
		t.Errorf("expected 0 entries for foo/bar/, got %d", len(result.ContextEntries))
	}
}

func TestResolve_DirQuery_NeverPanics(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		tmpDir := t.TempDir()

		depth := rapid.IntRange(0, 3).Draw(rt, "depth")
		dir := tmpDir

		for i := range depth {
			dir = filepath.Join(dir, genDirName(rt))
			if err := os.MkdirAll(dir, 0o750); err != nil {
				t.Fatalf("mkdir: %v", err)
			}

			numEntries := rapid.IntRange(0, 4).Draw(rt, "numEntries")
			var entries []ContextEntry

			for j := range numEntries {
				numMatch := rapid.IntRange(0, 2).Draw(rt, "numMatch")
				var match []string
				for range numMatch {
					// Include directory patterns in the mix.
					if rapid.Bool().Draw(rt, "isDirPattern") {
						match = append(match, genGlob(rt)+"/")
					} else {
						match = append(match, genGlob(rt))
					}
				}

				entries = append(entries, ContextEntry{
					Content: fmt.Sprintf("content-%d-%d", i, j),
					Match:   match,
					On:      FlexList{genOnValue(rt)},
					When:    genWhenValue(rt),
				})
			}

			if len(entries) > 0 {
				writeAgentsYAML(t, dir, entries)
			}
		}

		action := genAction(rt)
		timing := genTiming(rt)

		// Must not panic.
		_, _, _ = Resolve(ResolveRequest{
			DirPath: dir,
			Action:  action,
			Timing:  timing,
			Root:    tmpDir,
		})
	})
}

func TestResolve_DirQuery_FileGlobDepthMatching(t *testing.T) {
	// Tests that file-glob patterns correctly match directories at various depths,
	// including the trailing-** regression (src/** must match src/api/).
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	apiDir := filepath.Join(tmpDir, "src", "api")
	handlersDir := filepath.Join(tmpDir, "src", "api", "handlers")
	testsDir := filepath.Join(tmpDir, "tests")
	otherDir := filepath.Join(tmpDir, "other")
	for _, d := range []string{handlersDir, testsDir, otherDir} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatal(err)
		}
	}

	writeTestFile(t, filepath.Join(tmpDir, "AGENTS.yaml"), `
context:
  - content: "src everything"
    match: ["src/**"]
  - content: "handler context"
    match: ["src/api/handlers/*.py"]
`)

	tests := []struct {
		name      string
		dir       string
		wantNames []string
	}{
		{"src/ gets both", srcDir, []string{"src everything", "handler context"}},
		{"src/api/ gets both", apiDir, []string{"src everything", "handler context"}},
		{"src/api/handlers/ gets both", handlersDir, []string{"src everything", "handler context"}},
		{"tests/ gets neither", testsDir, nil},
		{"other/ gets neither", otherDir, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := Resolve(ResolveRequest{
				DirPath: tt.dir,
				Action:  ActionAll,
				Timing:  TimingAll,
				Root:    tmpDir,
			})
			if err != nil {
				t.Fatalf("Resolve() error: %v", err)
			}
			if len(result.ContextEntries) != len(tt.wantNames) {
				got := make([]string, len(result.ContextEntries))
				for i, e := range result.ContextEntries {
					got[i] = e.Content
				}
				t.Errorf("expected %d entries %v, got %d %v",
					len(tt.wantNames), tt.wantNames, len(result.ContextEntries), got)
			}
		})
	}
}

func TestResolve_DirQuery_ExcludeFileGlobPattern(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	vendorDir := filepath.Join(tmpDir, "vendor")
	vendorDeepDir := filepath.Join(tmpDir, "vendor", "github.com", "pkg")
	srcVendorDir := filepath.Join(tmpDir, "src", "vendor")
	for _, d := range []string{srcDir, vendorDeepDir, srcVendorDir} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatal(err)
		}
	}

	writeTestFile(t, filepath.Join(tmpDir, "AGENTS.yaml"), `
context:
  - content: "everywhere except vendor"
    exclude: ["**/vendor/**"]
`)

	tests := []struct {
		name  string
		dir   string
		wantN int
	}{
		{"src/ not excluded", srcDir, 1},
		{"vendor/ excluded", vendorDir, 0},
		{"vendor/deep excluded", vendorDeepDir, 0},
		{"src/vendor/ excluded", srcVendorDir, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := Resolve(ResolveRequest{
				DirPath: tt.dir,
				Action:  ActionAll,
				Timing:  TimingAll,
				Root:    tmpDir,
			})
			if err != nil {
				t.Fatalf("Resolve() error: %v", err)
			}
			if len(result.ContextEntries) != tt.wantN {
				t.Errorf("expected %d entries, got %d", tt.wantN, len(result.ContextEntries))
			}
		})
	}
}

func TestResolve_DirQuery_DoubleStarMiddlePattern(t *testing.T) {
	tmpDir := t.TempDir()
	srcTests := filepath.Join(tmpDir, "src", "api", "tests")
	srcApi := filepath.Join(tmpDir, "src", "api")
	srcDir := filepath.Join(tmpDir, "src")
	otherTests := filepath.Join(tmpDir, "other", "tests")
	for _, d := range []string{srcTests, otherTests} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatal(err)
		}
	}

	writeTestFile(t, filepath.Join(tmpDir, "AGENTS.yaml"), `
context:
  - content: "src test files"
    match: ["src/**/tests/*.py"]
`)

	tests := []struct {
		name  string
		dir   string
		wantN int
	}{
		{"src/api/tests/ matches", srcTests, 1},
		{"src/api/ matches (pattern goes through)", srcApi, 1},
		{"src/ matches (pattern starts here)", srcDir, 1},
		{"other/tests/ does not match (wrong prefix)", otherTests, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := Resolve(ResolveRequest{
				DirPath: tt.dir,
				Action:  ActionAll,
				Timing:  TimingAll,
				Root:    tmpDir,
			})
			if err != nil {
				t.Fatalf("Resolve() error: %v", err)
			}
			if len(result.ContextEntries) != tt.wantN {
				t.Errorf("expected %d entries, got %d", tt.wantN, len(result.ContextEntries))
			}
		})
	}
}

func TestResolve_DirQuery_DoubleStarPrefixMatchIsSelective(t *testing.T) {
	// Patterns like **/api/*.py should match directories containing api/ but not all directories.
	tmpDir := t.TempDir()
	apiDir := filepath.Join(tmpDir, "src", "api")
	modelsDir := filepath.Join(tmpDir, "src", "models")
	for _, d := range []string{apiDir, modelsDir} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatal(err)
		}
	}

	writeTestFile(t, filepath.Join(tmpDir, "AGENTS.yaml"), `
context:
  - content: "api handlers"
    match: ["**/api/*.py"]
`)

	tests := []struct {
		name  string
		dir   string
		wantN int
	}{
		{"src/api/ matches", apiDir, 1},
		{"src/models/ does not match", modelsDir, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := Resolve(ResolveRequest{
				DirPath: tt.dir,
				Action:  ActionAll,
				Timing:  TimingAll,
				Root:    tmpDir,
			})
			if err != nil {
				t.Fatalf("Resolve() error: %v", err)
			}
			if len(result.ContextEntries) != tt.wantN {
				t.Errorf("expected %d entries, got %d", tt.wantN, len(result.ContextEntries))
			}
		})
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
