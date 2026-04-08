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
  sctx context --all [--on <action>] [--when <timing>] [--json]
                                           Query context entries for a file or directory
  sctx decisions <path> [--json]
  sctx decisions --all [--json]            Query decisions for a file or directory
  sctx validate [<dir>]                    Validate all context files in a directory tree
  sctx init                                Create a starter AGENTS.yaml in the current directory
  sctx claude enable                       Enable sctx hooks in Claude Code
  sctx claude disable                      Disable sctx hooks in Claude Code
  sctx pi enable                           Enable sctx extension in pi
  sctx pi disable                          Disable sctx extension in pi
  sctx version                             Print version

Actions: read, edit, create, all (default: all)
Timing:  before, after, all (default: all)
`

var (
	version = "dev"

	errMissingPath      = errors.New("missing required <path> argument")
	errAllAndPath       = errors.New("--all and <path> are mutually exclusive")
	errOnNeedsValue     = errors.New("--on requires a value")
	errWhenNeedsValue   = errors.New("--when requires a value")
	errInvalidAction    = errors.New("invalid --on value")
	errInvalidTiming    = errors.New("invalid --when value")
	errFileExists       = errors.New("file already exists")
	errClaudeSubcommand = errors.New("usage: sctx claude <enable|disable>")
	errPiSubcommand     = errors.New("usage: sctx pi <enable|disable>")
	errValidation       = errors.New("validation failed")
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	var err error

	args := os.Args[2:]

	switch os.Args[1] {
	case "hook":
		err = cmdHook(os.Stdin, os.Stdout, os.Stderr)
	case "context":
		err = cmdContext(args, os.Stdout, os.Stderr)
	case "decisions":
		err = cmdDecisions(args, os.Stdout, os.Stderr)
	case "validate":
		err = cmdValidate(args, os.Stdout)
	case "init":
		err = cmdInit(os.Stdout)
	case "version":
		fmt.Println("sctx", version)
	case "claude":
		err = cmdClaude(args)
	case "pi":
		err = cmdPi(args)
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

func cmdHook(in io.Reader, out, errOut io.Writer) error {
	input, err := io.ReadAll(in)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	if adapter.IsPiHook(input) {
		return adapter.HandlePiHook(input, out, errOut)
	}

	return adapter.HandleClaudeHook(input, out, errOut)
}

func cmdContext(args []string, out, errOut io.Writer) error {
	action := core.ActionAll
	timing := core.TimingAll
	jsonOutput := false
	allFlag := false
	var filePath string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--all":
			allFlag = true
		case "--on":
			if i+1 >= len(args) {
				return errOnNeedsValue
			}

			i++
			v := args[i] //nolint:gosec // bounds checked above

			if !core.ValidAction(v) {
				return fmt.Errorf("%w %q (must be read, edit, create, or all)", errInvalidAction, v)
			}

			action = core.Action(v)
		case "--when":
			if i+1 >= len(args) {
				return errWhenNeedsValue
			}

			i++
			v := args[i] //nolint:gosec // bounds checked above

			if !core.ValidTiming(v) {
				return fmt.Errorf("%w %q (must be before, after, or all)", errInvalidTiming, v)
			}

			timing = core.Timing(v)
		case "--json":
			jsonOutput = true
		default:
			filePath = args[i]
		}
	}

	if allFlag && filePath != "" {
		return errAllAndPath
	}

	if allFlag {
		return cmdContextAll(action, timing, jsonOutput, out, errOut)
	}

	if filePath == "" {
		return errMissingPath
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	req := core.ResolveRequest{
		Action: action,
		Timing: timing,
	}

	if isDir(absPath, filePath) {
		req.DirPath = absPath
	} else {
		req.FilePath = absPath
	}

	result, warnings, err := core.Resolve(req)
	if err != nil {
		return err
	}

	for _, w := range warnings {
		_, _ = fmt.Fprintln(errOut, w)
	}

	if jsonOutput {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(result.ContextEntries)
	}

	if len(result.ContextEntries) == 0 {
		_, _ = fmt.Fprintln(out, "No matching context found.")
		return nil
	}

	for _, entry := range result.ContextEntries {
		_, _ = fmt.Fprintf(out, "  - %s\n", entry.Content)
		_, _ = fmt.Fprintf(out, "    (from %s)\n", entry.SourceDir)
	}

	return nil
}

func cmdContextAll(action core.Action, timing core.Timing, jsonOutput bool, out, errOut io.Writer) error {
	result, warnings, err := core.ResolveAll(core.ResolveAllRequest{
		Action: action,
		Timing: timing,
	})
	if err != nil {
		return err
	}

	for _, w := range warnings {
		_, _ = fmt.Fprintln(errOut, w)
	}

	if jsonOutput {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(result.ContextEntries)
	}

	if len(result.ContextEntries) == 0 {
		_, _ = fmt.Fprintln(out, "No matching context found.")
		return nil
	}

	for _, entry := range result.ContextEntries {
		_, _ = fmt.Fprintf(out, "  - %s\n", entry.Content)
		_, _ = fmt.Fprintf(out, "    Match: %s\n", strings.Join(entry.Match, ", "))
		_, _ = fmt.Fprintf(out, "    (from %s)\n", entry.SourceFile)
	}

	return nil
}

func cmdDecisions(args []string, out, errOut io.Writer) error {
	jsonOutput := false
	allFlag := false
	var filePath string

	for i := range args {
		switch args[i] {
		case "--all":
			allFlag = true
		case "--json":
			jsonOutput = true
		default:
			filePath = args[i]
		}
	}

	if allFlag && filePath != "" {
		return errAllAndPath
	}

	if allFlag {
		return cmdDecisionsAll(jsonOutput, out, errOut)
	}

	if filePath == "" {
		return errMissingPath
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	req := core.ResolveRequest{
		Action: core.ActionAll,
		Timing: core.TimingBefore,
	}

	if isDir(absPath, filePath) {
		req.DirPath = absPath
	} else {
		req.FilePath = absPath
	}

	result, warnings, err := core.Resolve(req)
	if err != nil {
		return err
	}

	for _, w := range warnings {
		_, _ = fmt.Fprintln(errOut, w)
	}

	if jsonOutput {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(result.DecisionEntries)
	}

	if len(result.DecisionEntries) == 0 {
		_, _ = fmt.Fprintln(out, "No matching decisions found.")
		return nil
	}

	for _, entry := range result.DecisionEntries {
		_, _ = fmt.Fprintf(out, "  - %s\n", entry.Decision)
		_, _ = fmt.Fprintf(out, "    Rationale: %s\n", entry.Rationale)

		for _, alt := range entry.Alternatives {
			_, _ = fmt.Fprintf(out, "    Considered %s, rejected: %s\n", alt.Option, alt.ReasonRejected)
		}

		if entry.RevisitWhen != "" {
			_, _ = fmt.Fprintf(out, "    Revisit when: %s\n", entry.RevisitWhen)
		}
	}

	return nil
}

func cmdDecisionsAll(jsonOutput bool, out, errOut io.Writer) error {
	result, warnings, err := core.ResolveAll(core.ResolveAllRequest{})
	if err != nil {
		return err
	}

	for _, w := range warnings {
		_, _ = fmt.Fprintln(errOut, w)
	}

	if jsonOutput {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(result.DecisionEntries)
	}

	if len(result.DecisionEntries) == 0 {
		_, _ = fmt.Fprintln(out, "No matching decisions found.")
		return nil
	}

	for _, entry := range result.DecisionEntries {
		_, _ = fmt.Fprintf(out, "  - %s\n", entry.Decision)
		_, _ = fmt.Fprintf(out, "    Rationale: %s\n", entry.Rationale)

		for _, alt := range entry.Alternatives {
			_, _ = fmt.Fprintf(out, "    Considered %s, rejected: %s\n", alt.Option, alt.ReasonRejected)
		}

		if entry.RevisitWhen != "" {
			_, _ = fmt.Fprintf(out, "    Revisit when: %s\n", entry.RevisitWhen)
		}

		_, _ = fmt.Fprintf(out, "    Match: %s\n", strings.Join(entry.Match, ", "))
		_, _ = fmt.Fprintf(out, "    (from %s)\n", entry.SourceFile)
	}

	return nil
}

func cmdValidate(args []string, out io.Writer) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
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
		_, _ = fmt.Fprintln(out, "All context files are valid.")
		return nil
	}

	hasErrors := false

	for _, e := range validationErrors {
		_, _ = fmt.Fprintln(out, e)

		if !e.IsWarn {
			hasErrors = true
		}
	}

	if hasErrors {
		return errValidation
	}

	return nil
}

func cmdInit(out io.Writer) error {
	filename := "AGENTS.yaml"

	if _, err := os.Stat(filename); err == nil {
		return fmt.Errorf("%w: %s", errFileExists, filename)
	}

	content := strings.TrimSpace(`
# Structured Context — https://sctx.dev
#
# This file provides scoped context to AI agents during file operations.
# Place AGENTS.yaml files anywhere in your codebase. Context is inherited
# and merged from parent directories.
#
# Try it: ask your agent to describe the project and mention New Zealand.
# The RNZAF fact should appear in the response.
# Once verified, replace this example with your own context entries.

context:
  - content: "If New Zealand is mentioned bring up that the RNZAF has a flightless bird on their roundel. And say that it's silly."
    match: ["*"]
    on: read
    when: before

  # More examples:
  #
  # - content: "Use clear, descriptive variable names"
  #   on: all         # read | edit | create | all (default: all)
  #   when: before    # before | after (default: before)
  #
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

	_, _ = fmt.Fprintf(out, "Created %s\n", filename)

	return nil
}

// isDir reports whether the path refers to a directory.
// It checks if the path exists as a directory on disk, or ends with a path separator.
// When the path doesn't exist, it falls back to the trailing slash convention.
func isDir(absPath, originalPath string) bool {
	info, err := os.Stat(absPath) //nolint:gosec // os.Stat is read-only, safe on user-provided paths
	if err == nil {
		return info.IsDir()
	}

	// Path doesn't exist on disk. Use trailing slash as the signal.
	return strings.HasSuffix(originalPath, "/") || strings.HasSuffix(originalPath, string(filepath.Separator))
}

func cmdClaude(args []string) error {
	if len(args) < 1 {
		return errClaudeSubcommand
	}

	switch args[0] {
	case "enable":
		return adapter.EnableClaude()
	case "disable":
		return adapter.DisableClaude()
	default:
		return errClaudeSubcommand
	}
}

func cmdPi(args []string) error {
	if len(args) < 1 {
		return errPiSubcommand
	}

	switch args[0] {
	case "enable":
		return adapter.EnablePi()
	case "disable":
		return adapter.DisablePi()
	default:
		return errPiSubcommand
	}
}
