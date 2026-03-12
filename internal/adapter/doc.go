// Package adapter translates agent-specific hook formats into the universal
// inputs that the core engine expects.
//
// Each adapter reads an agent's stdin (or env vars, or whatever that agent
// provides), extracts a file path, action, and timing, calls core.Resolve,
// and formats the result back into whatever the agent expects on stdout.
//
// Currently supports Claude Code and pi. Adding a new agent means writing a new
// adapter function, not changing the core engine.
package adapter
