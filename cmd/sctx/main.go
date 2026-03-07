package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gregology/sctx/internal/adapter"
	"github.com/gregology/sctx/internal/core"
	"github.com/gregology/sctx/internal/validator"
)

const usage = `sctx — Structured Context CLI

Usage:
  sctx hook                                Read agent hook input from stdin, return matching context
  sctx context <path> [--on <action>] [--when <timing>] [--json]
                                           Query context entries for a file
  sctx decisions <path> [--json]           Query decisions for a file
  sctx validate [<dir>]                    Validate all context files in a directory tree
  sctx init                                Create a starter CONTEXT.yaml in the current directory
  sctx version                             Print version

Actions: read, edit, create, all (default: all)
Timing:  before, after (default: before)
`

var (
	version = "dev"

	errMissingPath    = errors.New("missing required <path> argument")
	errOnNeedsValue   = errors.New("--on requires a value")
	errWhenNeedsValue = errors.New("--when requires a value")
	errFileExists     = errors.New("file already exists")
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	var err error

	switch os.Args[1] {
	case "hook":
		err = cmdHook()
	case "context":
		err = cmdContext()
	case "decisions":
		err = cmdDecisions()
	case "validate":
		err = cmdValidate()
	case "init":
		err = cmdInit()
	case "version":
		fmt.Println("sctx", version)
	case "help", "--help", "-h":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n%s", os.Args[1], usage)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func cmdHook() error {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	return adapter.HandleClaudeHook(input)
}

func cmdContext() error {
	if len(os.Args) < 3 {
		return errMissingPath
	}

	filePath := os.Args[2]
	action := core.ActionAll
	timing := core.TimingBefore
	jsonOutput := false

	for i := 3; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--on":
			if i+1 >= len(os.Args) {
				return errOnNeedsValue
			}

			i++
			action = core.Action(os.Args[i])
		case "--when":
			if i+1 >= len(os.Args) {
				return errWhenNeedsValue
			}

			i++
			timing = core.Timing(os.Args[i])
		case "--json":
			jsonOutput = true
		}
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	result, warnings, err := core.Resolve(core.ResolveRequest{
		FilePath: absPath,
		Action:   action,
		Timing:   timing,
	})
	if err != nil {
		return err
	}

	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, w)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result.ContextEntries)
	}

	if len(result.ContextEntries) == 0 {
		fmt.Println("No matching context found.")
		return nil
	}

	for _, entry := range result.ContextEntries {
		fmt.Printf("  - %s\n", entry.Content)
		fmt.Printf("    (from %s)\n", entry.SourceDir)
	}

	return nil
}

func cmdDecisions() error {
	if len(os.Args) < 3 {
		return errMissingPath
	}

	filePath := os.Args[2]
	jsonOutput := false

	for i := 3; i < len(os.Args); i++ {
		if os.Args[i] == "--json" {
			jsonOutput = true
		}
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	result, warnings, err := core.Resolve(core.ResolveRequest{
		FilePath: absPath,
		Action:   core.ActionAll,
		Timing:   core.TimingBefore,
	})
	if err != nil {
		return err
	}

	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, w)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result.DecisionEntries)
	}

	if len(result.DecisionEntries) == 0 {
		fmt.Println("No matching decisions found.")
		return nil
	}

	for _, entry := range result.DecisionEntries {
		fmt.Printf("  - %s\n", entry.Decision)
		fmt.Printf("    Rationale: %s\n", entry.Rationale)

		for _, alt := range entry.Alternatives {
			fmt.Printf("    Considered %s, rejected: %s\n", alt.Option, alt.ReasonRejected)
		}

		if entry.RevisitWhen != "" {
			fmt.Printf("    Revisit when: %s\n", entry.RevisitWhen)
		}
	}

	return nil
}

func cmdValidate() error {
	dir := "."
	if len(os.Args) > 2 {
		dir = os.Args[2]
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	validationErrors, err := validator.ValidateTree(absDir)
	if err != nil {
		return err
	}

	if len(validationErrors) == 0 {
		fmt.Println("All context files are valid.")
		return nil
	}

	hasErrors := false

	for _, e := range validationErrors {
		fmt.Println(e)

		if !e.IsWarn {
			hasErrors = true
		}
	}

	if hasErrors {
		os.Exit(1)
	}

	return nil
}

func cmdInit() error {
	filename := "CONTEXT.yaml"

	if _, err := os.Stat(filename); err == nil {
		return fmt.Errorf("%w: %s", errFileExists, filename)
	}

	content := strings.TrimSpace(`
# Structured Context — https://sctx.dev
#
# This file provides scoped context to AI agents during file operations.
# Place CONTEXT.yaml files anywhere in your codebase. Context is inherited
# and merged from parent directories.

context:
  # Example: general guideline for all files
  # - content: "Use clear, descriptive variable names"
  #   on: all         # read | edit | create | all (default: all)
  #   when: before    # before | after (default: before)

  # Example: scoped to specific files
  # - content: "All API handlers must validate input"
  #   match: ["src/api/**/*.py"]
  #   exclude: ["**/*_test.py"]
  #   on: [edit, create]
  #   when: after

decisions:
  # Example: architectural decision record
  # - decision: "REST over GraphQL for public APIs"
  #   rationale: "Team expertise, simpler caching, better tooling support"
  #   revisit_when: "We need real-time subscriptions"
  #   date: 2025-10-20
  #   match: ["src/api/**"]
`) + "\n"

	if err := os.WriteFile(filename, []byte(content), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", filename, err)
	}

	fmt.Printf("Created %s\n", filename)

	return nil
}
