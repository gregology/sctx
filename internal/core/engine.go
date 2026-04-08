package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"gopkg.in/yaml.v3"
)

var (
	errMutuallyExclusive = errors.New("FilePath and DirPath are mutually exclusive")
	errPathRequired      = errors.New("FilePath or DirPath is required")
)

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

	if req.FilePath == "" && req.DirPath == "" {
		return nil, nil, errPathRequired
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
		matchedCtx := filterContext(cf, absPath, req.Action, req.Timing, matchesFileGlobs)
		result.ContextEntries = append(result.ContextEntries, matchedCtx...)

		matchedDec := filterDecisions(cf, absPath, matchesFileGlobs)
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
		matchedCtx := filterContext(cf, absDir, req.Action, req.Timing, matchesDirGlobs)
		result.ContextEntries = append(result.ContextEntries, matchedCtx...)

		matchedDec := filterDecisions(cf, absDir, matchesDirGlobs)
		result.DecisionEntries = append(result.DecisionEntries, matchedDec...)
	}

	return result, warnings, nil
}

// ResolveAll walks the entire directory tree from Root, discovers all
// AGENTS.yaml files, and returns every context and decision entry without
// glob filtering. Action and Timing filters are still applied to context entries.
func ResolveAll(req ResolveAllRequest) (*ResolveAllResult, []string, error) {
	root := req.Root
	var err error
	if root == "" {
		root, err = os.Getwd()
		if err != nil {
			return nil, nil, fmt.Errorf("getting working directory: %w", err)
		}
	}

	root, err = filepath.Abs(root)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving absolute path: %w", err)
	}

	result := &ResolveAllResult{}
	var warnings []string

	walkErr := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}

		w := collectDir(path, root, req, result)
		warnings = append(warnings, w...)
		return nil
	})
	if walkErr != nil {
		return nil, nil, fmt.Errorf("walking directory tree: %w", walkErr)
	}

	if len(result.ContextEntries) == 0 && len(result.DecisionEntries) == 0 {
		warnings = append(warnings, "warning: no AGENTS.yaml files found")
	}

	return result, warnings, nil
}

// collectDir parses AGENTS.yaml files in dir and appends entries to result.
func collectDir(dir, root string, req ResolveAllRequest, result *ResolveAllResult) []string {
	var warnings []string

	for _, name := range AgentsFileNames {
		fp := filepath.Join(dir, name)
		data, readErr := os.ReadFile(fp) //nolint:gosec // paths come from directory walk
		if readErr != nil {
			continue
		}

		var cf ContextFile
		if parseErr := yaml.Unmarshal(data, &cf); parseErr != nil {
			warnings = append(warnings, fmt.Sprintf("warning: failed to parse %s: %v", fp, parseErr))
			continue
		}

		cf.sourceDir = dir
		applyDefaults(&cf)

		relFile, _ := filepath.Rel(root, fp)
		relFile = filepath.ToSlash(relFile)

		collectContextEntries(cf, relFile, req, result)
		collectDecisionEntries(cf, relFile, result)
	}

	return warnings
}

func collectContextEntries(cf ContextFile, relFile string, req ResolveAllRequest, result *ResolveAllResult) {
	for _, entry := range cf.Context {
		if !matchesAction(entry.On, req.Action) {
			continue
		}
		if req.Timing != "" && req.Timing != TimingAll &&
			Timing(entry.When) != TimingAll && Timing(entry.When) != req.Timing {
			continue
		}
		result.ContextEntries = append(result.ContextEntries, AllContextEntry{
			Content:    entry.Content,
			Match:      entry.Match,
			SourceFile: relFile,
		})
	}
}

func collectDecisionEntries(cf ContextFile, relFile string, result *ResolveAllResult) {
	for _, entry := range cf.Decisions {
		result.DecisionEntries = append(result.DecisionEntries, AllDecisionEntry{
			Decision:     entry.Decision,
			Rationale:    entry.Rationale,
			Alternatives: entry.Alternatives,
			RevisitWhen:  entry.RevisitWhen,
			Date:         entry.Date,
			Match:        entry.Match,
			SourceFile:   relFile,
		})
	}
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

// globMatcher checks whether a path matches the given match/exclude patterns.
type globMatcher func(sourceDir, path string, match, exclude []string) bool

// filterContext returns context entries from cf that match the given path, action, and timing.
func filterContext(cf ContextFile, absPath string, action Action, timing Timing, matcher globMatcher) []MatchedContext {
	var matched []MatchedContext

	for _, entry := range cf.Context {
		if !matcher(cf.sourceDir, absPath, entry.Match, entry.Exclude) {
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

// filterDecisions returns decision entries from cf that match the given path.
func filterDecisions(cf ContextFile, absPath string, matcher globMatcher) []DecisionEntry {
	var matched []DecisionEntry

	for _, entry := range cf.Decisions {
		if !matcher(cf.sourceDir, absPath, entry.Match, nil) {
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
//
// Match and exclude use different strictness levels. Match is generous (extra context
// is acceptable). Exclude is strict (must not remove context that should be shown).
func matchesDirGlobs(sourceDir, absDir string, match, exclude []string) bool {
	relDir, err := filepath.Rel(sourceDir, absDir)
	if err != nil {
		return false
	}

	relDir = filepath.ToSlash(relDir)

	if strings.HasPrefix(relDir, "..") {
		return false
	}

	if relDir == "." {
		relDir = ""
	}

	if !anyDirPatternMatches(relDir, match) {
		return false
	}

	return !anyDirPatternExcludes(relDir, exclude)
}

// anyDirPatternMatches reports whether any match pattern applies to the directory.
// Uses generous matching.
func anyDirPatternMatches(relDir string, patterns []string) bool {
	for _, pattern := range patterns {
		if isDirPattern(pattern) {
			if dirSlashPatternMatches(relDir, pattern) {
				return true
			}
		} else if fileGlobMatchesDir(pattern, relDir) {
			return true
		}
	}

	return false
}

// anyDirPatternExcludes reports whether any exclude pattern applies to the directory.
// Uses strict matching to avoid over-excluding.
func anyDirPatternExcludes(relDir string, patterns []string) bool {
	for _, pattern := range patterns {
		if isDirPattern(pattern) {
			if dirSlashPatternMatches(relDir, pattern) {
				return true
			}
		} else if fileGlobExcludesDir(pattern, relDir) {
			return true
		}
	}

	return false
}

// dirSlashPatternMatches checks a trailing-slash pattern against a directory path.
func dirSlashPatternMatches(relDir, pattern string) bool {
	if relDir == "" {
		// Source directory (relDir="") should only match patterns that can
		// match zero path segments. Using "./" as a stand-in is incorrect
		// because "*" matches "." in glob semantics, causing "*/" and
		// "**/*/" to falsely match the source directory.
		//
		// "./" is an explicit self-reference (used in docs/examples.md).
		// "./**/" means "self and all subdirs". Strip "./" first so the
		// **/ loop handles the rest.
		// Bare **/ chains mean "any directory" and match zero segments.
		// Everything else (*/  src/  **/src/  **/*/) requires real path
		// segments and must not match the source directory.
		trimmed := strings.TrimPrefix(pattern, "./")

		for strings.HasPrefix(trimmed, "**/") {
			trimmed = trimmed[3:]
		}

		return trimmed == ""
	}

	// Strip "./" prefix — patterns are already relative to sourceDir,
	// so "./" is redundant and doublestar treats "." as a literal segment.
	dirWithSlash := relDir + "/"
	ok, err := doublestar.Match(strings.TrimPrefix(pattern, "./"), dirWithSlash)

	return err == nil && ok
}

// fileGlobMatchesDir reports whether a file-glob pattern could match files inside relDir.
// This is used for match evaluation and is intentionally generous: if the pattern
// could possibly produce hits inside the directory, it returns true. Extra context
// is acceptable; missing context is not.
func fileGlobMatchesDir(pattern, relDir string) bool {
	if pattern == "**" || pattern == "**/*" {
		return true
	}

	// Any pattern could match files in the sourceDir itself.
	if relDir == "" {
		return true
	}

	// Patterns starting with **/ can match at any depth, so they're relevant
	// to any directory.
	if strings.HasPrefix(pattern, "**/") {
		return true
	}

	patParts := strings.Split(pattern, "/")
	dirParts := strings.Split(relDir, "/")

	return dirCouldContainMatch(patParts, dirParts)
}

// fileGlobExcludesDir reports whether a file-glob exclude pattern should exclude relDir.
// This is stricter than fileGlobMatchesDir: it only returns true when the directory
// is clearly within the exclude pattern's scope. This prevents patterns like
// "vendor/**" from excluding the root directory, and "**/vendor/**" from
// excluding every directory.
func fileGlobExcludesDir(pattern, relDir string) bool {
	if pattern == "**" || pattern == "**/*" {
		return true
	}

	if relDir == "" {
		// Only exclude the root for patterns that genuinely target everything.
		// Patterns like "vendor/**" don't target the root.
		return false
	}

	patParts := strings.Split(pattern, "/")
	dirParts := strings.Split(relDir, "/")

	return dirCouldExclude(patParts, dirParts)
}

// collapseDoubleStars removes consecutive "**" segments from a pattern.
// Multiple adjacent "**" segments are semantically equivalent to a single "**"
// but cause exponential branching in the recursive matcher.
func collapseDoubleStars(parts []string) []string {
	out := make([]string, 0, len(parts))

	for _, p := range parts {
		if p == "**" && len(out) > 0 && out[len(out)-1] == "**" {
			continue
		}

		out = append(out, p)
	}

	return out
}

// dirCouldContainMatch reports whether a directory (dirParts) could contain files
// matching the given pattern (patParts). Both are split by "/".
// Used for match evaluation (generous).
func dirCouldContainMatch(patParts, dirParts []string) bool {
	return matchSegments(collapseDoubleStars(patParts), dirParts, 0, 0)
}

// dirCouldExclude reports whether a directory should be excluded by the pattern.
// Stricter than dirCouldContainMatch: requires that at least one literal segment
// in the pattern has been validated against an actual directory segment.
// This prevents "**/vendor/**" from excluding directories that don't contain "vendor"
// in their path.
func dirCouldExclude(patParts, dirParts []string) bool {
	return matchSegmentsStrict(collapseDoubleStars(patParts), dirParts, 0, 0, false)
}

// matchSegments walks pattern and directory segments to determine if the pattern
// could produce file matches inside the directory. This is the generous version
// used for match evaluation.
func matchSegments(pat, dir []string, pi, di int) bool {
	for pi < len(pat) && di < len(dir) {
		p := pat[pi]

		if p == "**" {
			if pi == len(pat)-1 {
				return true
			}

			for skip := 0; skip <= len(dir)-di; skip++ {
				if matchSegments(pat, dir, pi+1, di+skip) {
					return true
				}
			}

			return false
		}

		ok, err := doublestar.Match(p, dir[di])
		if err != nil || !ok {
			return false
		}

		pi++
		di++
	}

	if di == len(dir) {
		return pi < len(pat)
	}

	return false
}

// matchSegmentsStrict is the strict version of matchSegments, used for exclude
// evaluation. It tracks whether at least one non-"**" pattern segment was matched
// against a real directory segment (segmentValidated). If not, the directory
// is not confirmed to be in scope and the match is rejected.
//
// Truth table:
//
//	Pattern          | Dir     | Result | Why
//	*                | foo     | false  | pattern exhausted, no remaining segments
//	foo/*            | foo     | true   | "foo" validated, * remains for children
//	**/vendor/**     | src     | false  | no segment validated, ** can't confirm alone
//	**/vendor/**     | vendor  | true   | "vendor" validated, ** remains
//	**/*.py          | src     | false  | ** consumed src but *.py is just a filename
//	*/*.py           | src     | true   | * validated against src, *.py remains
func matchSegmentsStrict(pat, dir []string, pi, di int, segmentValidated bool) bool {
	for pi < len(pat) && di < len(dir) {
		p := pat[pi]

		if p == "**" {
			if pi == len(pat)-1 {
				return true
			}

			for skip := 0; skip <= len(dir)-di; skip++ {
				if matchSegmentsStrict(pat, dir, pi+1, di+skip, segmentValidated) {
					return true
				}
			}

			return false
		}

		ok, err := doublestar.Match(p, dir[di])
		if err != nil || !ok {
			return false
		}

		segmentValidated = true
		pi++
		di++
	}

	if di == len(dir) {
		remaining := pat[pi:]

		if len(remaining) == 0 {
			return false
		}

		// At least one non-** segment must have been validated against a
		// real directory segment. Without that, we can't confirm this
		// directory is in scope. For example, **/*.py has remaining ["*.py"]
		// after ** consumes all dirs, but no directory was validated — the
		// pattern targets files, not directories.
		return segmentValidated
	}

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
