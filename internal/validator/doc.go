// Package validator checks AGENTS.yaml files for
// schema errors, invalid globs, and missing required fields.
//
// It walks a directory tree, finds all context files, and returns a list
// of validation errors and warnings. Warnings don't fail the check.
package validator
