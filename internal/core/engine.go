package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"gopkg.in/yaml.v3"
)

// AgentsFileNames are the recognized filenames, in priority order.
var AgentsFileNames = []string{
	"AGENTS.yaml",
	"AGENTS.yml",
}

// Resolve finds all context and decisions that apply to a file for a given
// action and timing. This is the primary entry point for the core engine.
func Resolve(req ResolveRequest) (*ResolveResult, []string, error) {
	absPath, err := filepath.Abs(req.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving absolute path: %w", err)
	}

	root := req.Root
	if root == "" {
		root, err = os.Getwd()
		if err != nil {
			return nil, nil, fmt.Errorf("getting working directory: %w", err)
		}
	}

	root, err = filepath.Abs(root)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving root path: %w", err)
	}

	// Reject file paths outside the project root.
	if !isDescendant(root, absPath) {
		return &ResolveResult{}, nil, nil
	}

	files, warnings := discoverAndParse(filepath.Dir(absPath), root)

	result := &ResolveResult{}

	for _, cf := range files {
		matchedCtx := filterContext(cf, absPath, req.Action, req.Timing)
		result.ContextEntries = append(result.ContextEntries, matchedCtx...)

		matchedDec := filterDecisions(cf, absPath)
		result.DecisionEntries = append(result.DecisionEntries, matchedDec...)
	}

	return result, warnings, nil
}

// discoverAndParse walks from startDir up to root, collecting and parsing all
// context files. Files from parent directories come first (lower specificity).
func discoverAndParse(startDir, root string) (files []ContextFile, warnings []string) {
	// Guard: if startDir is not a descendant of root, don't walk.
	if !isDescendant(root, startDir) {
		return nil, nil
	}

	var dirs []string
	current := startDir

	for {
		dirs = append(dirs, current)

		if current == root {
			break
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}

		current = parent
	}

	// Reverse so parent directories come first.
	for i, j := 0, len(dirs)-1; i < j; i, j = i+1, j-1 {
		dirs[i], dirs[j] = dirs[j], dirs[i]
	}

	for _, dir := range dirs {
		for _, name := range AgentsFileNames {
			path := filepath.Join(dir, name)

			data, err := os.ReadFile(path) //nolint:gosec // paths come from directory walk, not user input
			if err != nil {
				continue // File doesn't exist, just skip.
			}

			var cf ContextFile
			if err := yaml.Unmarshal(data, &cf); err != nil {
				warnings = append(warnings, fmt.Sprintf("warning: failed to parse %s: %v", path, err))
				continue
			}

			cf.sourceDir = dir
			applyDefaults(&cf)
			files = append(files, cf)
		}
	}

	if len(files) == 0 {
		warnings = append(warnings, "warning: no AGENTS.yaml files found")
	}

	return files, warnings
}

// isDescendant reports whether child is under parent (or equal to it).
func isDescendant(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..")
}

// applyDefaults fills in default values for context and decision entries.
func applyDefaults(cf *ContextFile) {
	for i := range cf.Context {
		if len(cf.Context[i].Match) == 0 {
			cf.Context[i].Match = []string{"**"}
		}

		if len(cf.Context[i].On) == 0 {
			cf.Context[i].On = []string{"all"}
		}

		if cf.Context[i].When == "" {
			cf.Context[i].When = "before"
		}
	}

	for i := range cf.Decisions {
		if len(cf.Decisions[i].Match) == 0 {
			cf.Decisions[i].Match = []string{"**"}
		}
	}
}

// filterContext returns context entries from cf that match the given file, action, and timing.
func filterContext(cf ContextFile, absPath string, action Action, timing Timing) []MatchedContext {
	var matched []MatchedContext

	for _, entry := range cf.Context {
		if !matchesGlobs(cf.sourceDir, absPath, entry.Match, entry.Exclude) {
			continue
		}

		if !matchesAction(entry.On, action) {
			continue
		}

		if timing != TimingAll && Timing(entry.When) != timing {
			continue
		}

		matched = append(matched, MatchedContext{
			Content:   entry.Content,
			SourceDir: cf.sourceDir,
		})
	}

	return matched
}

// filterDecisions returns decision entries from cf that match the given file.
func filterDecisions(cf ContextFile, absPath string) []DecisionEntry {
	var matched []DecisionEntry

	for _, entry := range cf.Decisions {
		if !matchesGlobs(cf.sourceDir, absPath, entry.Match, nil) {
			continue
		}

		matched = append(matched, entry)
	}

	return matched
}

// matchesGlobs checks if absPath matches any of the match patterns and none of the exclude patterns.
// Globs are resolved relative to sourceDir.
func matchesGlobs(sourceDir, absPath string, match, exclude []string) bool {
	relPath, err := filepath.Rel(sourceDir, absPath)
	if err != nil {
		return false
	}

	// Normalize to forward slashes for consistent glob matching.
	relPath = filepath.ToSlash(relPath)

	// Don't match files outside this directory tree.
	if strings.HasPrefix(relPath, "..") {
		return false
	}

	matched := false

	for _, pattern := range match {
		ok, matchErr := doublestar.Match(pattern, relPath)
		if matchErr != nil {
			continue
		}

		if ok {
			matched = true
			break
		}
	}

	if !matched {
		return false
	}

	for _, pattern := range exclude {
		ok, matchErr := doublestar.Match(pattern, relPath)
		if matchErr != nil {
			continue
		}

		if ok {
			return false
		}
	}

	return true
}

// matchesAction checks if the requested action is included in the entry's on list.
func matchesAction(on FlexList, action Action) bool {
	if action == ActionAll {
		return true
	}

	for _, a := range on {
		if Action(a) == ActionAll || Action(a) == action {
			return true
		}
	}

	return false
}
