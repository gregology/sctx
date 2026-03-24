package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"gopkg.in/yaml.v3"
)

var errMutuallyExclusive = fmt.Errorf("FilePath and DirPath are mutually exclusive")

// AgentsFileNames are the recognized filenames, in priority order.
var AgentsFileNames = []string{
	"AGENTS.yaml",
	"AGENTS.yml",
}

// Resolve finds all context and decisions that apply to a file or directory
// for a given action and timing. This is the primary entry point for the core engine.
// Set FilePath for file queries, DirPath for directory queries. They are mutually exclusive.
func Resolve(req ResolveRequest) (*ResolveResult, []string, error) {
	if req.FilePath != "" && req.DirPath != "" {
		return nil, nil, errMutuallyExclusive
	}

	root := req.Root
	var err error
	if root == "" {
		root, err = os.Getwd()
		if err != nil {
			return nil, nil, fmt.Errorf("getting working directory: %w", err)
		}
	}

	if req.DirPath != "" {
		return resolveDir(req, root)
	}

	return resolveFile(req, root)
}

func resolveFile(req ResolveRequest, root string) (*ResolveResult, []string, error) {
	absPath, err := filepath.Abs(req.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving absolute path: %w", err)
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

func resolveDir(req ResolveRequest, root string) (*ResolveResult, []string, error) {
	absDir, err := filepath.Abs(req.DirPath)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving absolute path: %w", err)
	}

	// For directory queries, start discovery from the directory itself,
	// not its parent (which is what filepath.Dir would give us for a file).
	files, warnings := discoverAndParse(absDir, root)

	result := &ResolveResult{}

	for _, cf := range files {
		matchedCtx := filterContextDir(cf, absDir, req.Action, req.Timing)
		result.ContextEntries = append(result.ContextEntries, matchedCtx...)

		matchedDec := filterDecisionsDir(cf, absDir)
		result.DecisionEntries = append(result.DecisionEntries, matchedDec...)
	}

	return result, warnings, nil
}

// discoverAndParse walks from startDir up to root, collecting and parsing all
// context files. Files from parent directories come first (lower specificity).
func discoverAndParse(startDir, root string) (files []ContextFile, warnings []string) {
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
		if !matchesFileGlobs(cf.sourceDir, absPath, entry.Match, entry.Exclude) {
			continue
		}

		if !matchesAction(entry.On, action) {
			continue
		}

		if timing != TimingAll && Timing(entry.When) != TimingAll && Timing(entry.When) != timing {
			continue
		}

		matched = append(matched, MatchedContext{
			Content:   entry.Content,
			SourceDir: cf.sourceDir,
		})
	}

	return matched
}

// filterContextDir returns context entries from cf that match the given directory, action, and timing.
func filterContextDir(cf ContextFile, absDir string, action Action, timing Timing) []MatchedContext {
	var matched []MatchedContext

	for _, entry := range cf.Context {
		if !matchesDirGlobs(cf.sourceDir, absDir, entry.Match, entry.Exclude) {
			continue
		}

		if !matchesAction(entry.On, action) {
			continue
		}

		if timing != TimingAll && Timing(entry.When) != TimingAll && Timing(entry.When) != timing {
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
		if !matchesFileGlobs(cf.sourceDir, absPath, entry.Match, nil) {
			continue
		}

		matched = append(matched, entry)
	}

	return matched
}

// filterDecisionsDir returns decision entries from cf that match the given directory.
func filterDecisionsDir(cf ContextFile, absDir string) []DecisionEntry {
	var matched []DecisionEntry

	for _, entry := range cf.Decisions {
		if !matchesDirGlobs(cf.sourceDir, absDir, entry.Match, nil) {
			continue
		}

		matched = append(matched, entry)
	}

	return matched
}

// isDirPattern reports whether a glob pattern targets a directory (ends with /).
func isDirPattern(pattern string) bool {
	return strings.HasSuffix(pattern, "/")
}

// matchesFileGlobs checks if absPath matches any of the match patterns and none of the exclude patterns.
// Directory patterns (trailing /) are skipped — they never match file queries.
// Globs are resolved relative to sourceDir.
func matchesFileGlobs(sourceDir, absPath string, match, exclude []string) bool {
	relPath, err := filepath.Rel(sourceDir, absPath)
	if err != nil {
		return false
	}

	relPath = filepath.ToSlash(relPath)

	if strings.HasPrefix(relPath, "..") {
		return false
	}

	matched := false

	for _, pattern := range match {
		if isDirPattern(pattern) {
			continue // directory patterns don't match files
		}

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
		if isDirPattern(pattern) {
			continue
		}

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

// matchesDirGlobs checks if absDir matches any of the match patterns for a directory query.
// For directory patterns (trailing /), the directory must match exactly.
// For file-glob patterns, the pattern must be capable of matching files inside the directory.
// Globs are resolved relative to sourceDir.
func matchesDirGlobs(sourceDir, absDir string, match, exclude []string) bool {
	relDir, err := filepath.Rel(sourceDir, absDir)
	if err != nil {
		return false
	}

	relDir = filepath.ToSlash(relDir)

	if strings.HasPrefix(relDir, "..") {
		return false
	}

	// "." means the directory is the sourceDir itself.
	if relDir == "." {
		relDir = ""
	}

	if !anyDirPatternMatches(relDir, match) {
		return false
	}

	return !anyDirPatternMatches(relDir, exclude)
}

// anyDirPatternMatches reports whether any pattern in the list matches the directory.
func anyDirPatternMatches(relDir string, patterns []string) bool {
	for _, pattern := range patterns {
		if dirPatternMatches(relDir, pattern) {
			return true
		}
	}

	return false
}

// dirPatternMatches checks a single pattern against a directory.
func dirPatternMatches(relDir, pattern string) bool {
	if isDirPattern(pattern) {
		return dirSlashPatternMatches(relDir, pattern)
	}

	return fileGlobMatchesDir(pattern, relDir)
}

// dirSlashPatternMatches checks a trailing-slash pattern against a directory path.
func dirSlashPatternMatches(relDir, pattern string) bool {
	dirWithSlash := relDir + "/"
	if relDir == "" {
		dirWithSlash = "./"
	}

	ok, err := doublestar.Match(pattern, dirWithSlash)

	return err == nil && ok
}

// fileGlobMatchesDir reports whether a file-glob pattern could match files inside relDir.
// For example, "tests/**" matches directory "tests", "**/*.py" matches any directory,
// and "src/**" does not match directory "tests".
func fileGlobMatchesDir(pattern, relDir string) bool {
	// Pattern "**" or "**/*" matches any directory.
	if pattern == "**" || pattern == "**/*" {
		return true
	}

	// If the pattern starts with "**/" it can match at any depth,
	// so it potentially matches any directory.
	if strings.HasPrefix(pattern, "**/") {
		return true
	}

	// If the queried directory is the sourceDir itself (relDir is empty),
	// any pattern could match files directly in it.
	if relDir == "" {
		return true
	}

	// Split both the pattern and the directory into segments and walk them
	// together. A pattern segment can be a literal ("src"), a wildcard ("*"),
	// or a recursive wildcard ("**"). We need to determine whether files
	// matching this pattern could exist inside relDir.
	patParts := strings.Split(pattern, "/")
	dirParts := strings.Split(relDir, "/")

	return dirCouldContainMatch(patParts, dirParts)
}

// dirCouldContainMatch reports whether a directory (dirParts) could contain files
// matching the given pattern (patParts). Both are split by "/".
//
// The key insight: we walk the pattern and directory segments together.
// If the directory is deeper than the pattern's directory portion, we need "**"
// somewhere in the pattern to bridge the gap. If the directory is shallower,
// the pattern's deeper segments might produce files under the directory.
func dirCouldContainMatch(patParts, dirParts []string) bool {
	return matchSegments(patParts, dirParts, 0, 0)
}

func matchSegments(pat, dir []string, pi, di int) bool {
	for pi < len(pat) && di < len(dir) {
		p := pat[pi]

		if p == "**" {
			// "**" can match zero or more directory segments.
			// Try consuming 0..len(dir)-di segments from dir.
			for skip := 0; skip <= len(dir)-di; skip++ {
				if matchSegments(pat, dir, pi+1, di+skip) {
					return true
				}
			}
			return false
		}

		// Check if this pattern segment matches the directory segment.
		ok, err := doublestar.Match(p, dir[di])
		if err != nil || !ok {
			return false
		}

		pi++
		di++
	}

	if di == len(dir) {
		// We've consumed all directory segments. The pattern still has remaining
		// segments (pi < len(pat)). Those remaining segments would match files
		// inside this directory, so the pattern is relevant.
		//
		// But if we also consumed the entire pattern (pi == len(pat)), the pattern
		// was fully used up matching directory segments. There's nothing left to
		// match files, so the pattern doesn't produce hits *inside* the directory.
		// Example: pattern "foo/*", dir "foo/bar" -> pattern consumed matching
		// "foo" and "bar", nothing left to match files in foo/bar/.
		return pi < len(pat)
	}

	// If we've consumed all pattern segments but there are directory segments left,
	// the directory is deeper than the pattern reaches. No match.
	return false
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
