// Package core implements the Structured Context resolution engine.
//
// Given a file path, an action (read/edit/create), and a timing (before/after),
// the engine discovers CONTEXT.yaml and AGENTS.yaml files by walking up the
// directory tree, parses them, filters entries by glob match and action, and
// returns the combined results.
//
// This package has no knowledge of any specific AI agent. It works with
// universal inputs and outputs. Agent-specific translation happens in the
// adapter package.
package core
