package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"gopkg.in/yaml.v3"

	"github.com/gregology/sctx/internal/core"
)

// ValidationError represents a single validation issue.
type ValidationError struct {
	File    string
	Message string
	IsWarn  bool
}

// String formats the error for display.
func (e ValidationError) String() string {
	prefix := "error"
	if e.IsWarn {
		prefix = "warning"
	}

	return fmt.Sprintf("%s: %s: %s", prefix, e.File, e.Message)
}

// ValidateTree walks from root and validates all context files found.
func ValidateTree(root string) ([]ValidationError, error) {
	var errs []ValidationError

	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() {
			return walkErr //nolint:wrapcheck // walk errors are already descriptive
		}

		if slices.Contains(core.AgentsFileNames, info.Name()) {
			errs = append(errs, ValidateFile(path)...)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking directory tree: %w", err)
	}

	return errs, nil
}

// Known field sets for each struct level.
var (
	knownTopLevel      = map[string]bool{"context": true, "decisions": true}
	knownContextEntry  = map[string]bool{"content": true, "match": true, "exclude": true, "on": true, "when": true}
	knownDecisionEntry = map[string]bool{"decision": true, "rationale": true, "alternatives": true, "revisit_when": true, "date": true, "match": true}
	knownAlternative   = map[string]bool{"option": true, "reason_rejected": true}
)

// ValidateFile validates a single context file.
func ValidateFile(path string) []ValidationError {
	data, err := os.ReadFile(path) //nolint:gosec // path comes from directory walk
	if err != nil {
		return []ValidationError{{File: path, Message: "cannot read file: " + err.Error()}}
	}

	var cf core.ContextFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return []ValidationError{{File: path, Message: "invalid YAML: " + err.Error()}}
	}

	var errs []ValidationError
	errs = append(errs, checkUnknownFields(path, data)...)
	errs = append(errs, validateContextEntries(path, cf.Context)...)
	errs = append(errs, validateDecisionEntries(path, cf.Decisions)...)

	return errs
}

// checkUnknownFields decodes YAML into a raw map and warns on unrecognised keys.
func checkUnknownFields(path string, data []byte) []ValidationError {
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil // parse errors are reported elsewhere
	}

	var warns []ValidationError
	warns = append(warns, warnUnknownKeys(path, "top level", raw, knownTopLevel)...)
	warns = append(warns, checkListEntries(path, "context", raw, knownContextEntry)...)

	for i, m := range extractMaps(raw, "decisions") {
		prefix := fmt.Sprintf("decisions[%d]", i)
		warns = append(warns, warnUnknownKeys(path, prefix, m, knownDecisionEntry)...)
		warns = append(warns, checkListEntries(path, prefix+".alternatives", m, knownAlternative)...)
	}

	return warns
}

// warnUnknownKeys emits a warning for each key in m not present in known.
func warnUnknownKeys(path, prefix string, m map[string]any, known map[string]bool) []ValidationError {
	var warns []ValidationError
	for key := range m {
		if !known[key] {
			warns = append(warns, ValidationError{
				File:    path,
				Message: fmt.Sprintf("%s: unknown field %q", prefix, key),
				IsWarn:  true,
			})
		}
	}
	return warns
}

// checkListEntries extracts a named list of maps from parent and checks each entry's keys.
// The prefix is used for display; the map key is derived from the last dot-separated segment.
func checkListEntries(path, prefix string, parent map[string]any, known map[string]bool) []ValidationError {
	key := prefix
	if idx := strings.LastIndex(prefix, "."); idx >= 0 {
		key = prefix[idx+1:]
	}
	var warns []ValidationError
	for i, m := range extractMaps(parent, key) {
		entryPrefix := fmt.Sprintf("%s[%d]", prefix, i)
		warns = append(warns, warnUnknownKeys(path, entryPrefix, m, known)...)
	}
	return warns
}

// extractMaps pulls a []map[string]any from a named key in parent, skipping non-map entries.
func extractMaps(parent map[string]any, key string) []map[string]any {
	list, ok := parent[key].([]any)
	if !ok {
		return nil
	}
	var out []map[string]any
	for _, item := range list {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func validateContextEntries(path string, entries []core.ContextEntry) []ValidationError {
	var errs []ValidationError

	for i, entry := range entries {
		prefix := fmt.Sprintf("context[%d]", i)

		if strings.TrimSpace(entry.Content) == "" {
			errs = append(errs, ValidationError{
				File:    path,
				Message: prefix + ": content is required",
			})
		}

		errs = append(errs, validateGlobs(path, prefix, "match", entry.Match)...)
		errs = append(errs, validateGlobs(path, prefix, "exclude", entry.Exclude)...)

		for _, action := range entry.On {
			if !core.ValidAction(action) {
				errs = append(errs, ValidationError{
					File:    path,
					Message: fmt.Sprintf("%s: invalid action %q (must be read, edit, create, or all)", prefix, action),
				})
			}
		}

		if entry.When != "" && !core.ValidTiming(entry.When) {
			errs = append(errs, ValidationError{
				File:    path,
				Message: fmt.Sprintf("%s: invalid when %q (must be before, after, or all)", prefix, entry.When),
			})
		}
	}

	return errs
}

func validateDecisionEntries(path string, entries []core.DecisionEntry) []ValidationError {
	var errs []ValidationError

	for i, entry := range entries {
		prefix := fmt.Sprintf("decisions[%d]", i)

		if strings.TrimSpace(entry.Decision) == "" {
			errs = append(errs, ValidationError{
				File:    path,
				Message: prefix + ": decision is required",
			})
		}

		if strings.TrimSpace(entry.Rationale) == "" {
			errs = append(errs, ValidationError{
				File:    path,
				Message: prefix + ": rationale is required",
			})
		}

		if entry.Date != "" {
			if _, err := time.Parse("2006-01-02", entry.Date); err != nil {
				errs = append(errs, ValidationError{
					File:    path,
					Message: fmt.Sprintf("%s: invalid date %q (must be YYYY-MM-DD)", prefix, entry.Date),
				})
			}
		}

		errs = append(errs, validateGlobs(path, prefix, "match", entry.Match)...)
	}

	return errs
}

func validateGlobs(path, prefix, field string, patterns []string) []ValidationError {
	var errs []ValidationError

	for _, pattern := range patterns {
		if _, err := doublestar.Match(pattern, "test"); err != nil {
			errs = append(errs, ValidationError{
				File:    path,
				Message: fmt.Sprintf("%s: invalid %s glob %q: %v", prefix, field, pattern, err),
			})
		}
	}

	return errs
}
