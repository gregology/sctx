package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"gopkg.in/yaml.v3"

	"github.com/gregology/sctx/internal/core"
)

var validActions = map[string]bool{
	"read": true, "edit": true, "create": true, "all": true,
}

var validTimings = map[string]bool{
	"before": true, "after": true,
}

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
	errs = append(errs, validateContextEntries(path, cf.Context)...)
	errs = append(errs, validateDecisionEntries(path, cf.Decisions)...)

	return errs
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
			if !validActions[action] {
				errs = append(errs, ValidationError{
					File:    path,
					Message: fmt.Sprintf("%s: invalid action %q (must be read, edit, create, or all)", prefix, action),
				})
			}
		}

		if entry.When != "" && !validTimings[entry.When] {
			errs = append(errs, ValidationError{
				File:    path,
				Message: fmt.Sprintf("%s: invalid when %q (must be before or after)", prefix, entry.When),
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
