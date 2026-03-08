---
title: CLI reference
description: All sctx commands, flags, and exit codes
---

# CLI reference

## sctx hook

Reads agent hook input from stdin, resolves matching context entries, and writes the response to stdout. This is the primary integration point for AI agents.

```bash
echo '{"tool_name":"Edit","tool_input":{"file_path":"/project/src/main.py"},"hook_event_name":"PreToolUse"}' | sctx hook
```

Currently supports Claude Code's JSON format. The adapter reads `tool_name`, `tool_input.file_path`, and `hook_event_name` from stdin, resolves context, and outputs Claude Code's expected `hookSpecificOutput` JSON.

Only context entries are included in hook output. Decisions are excluded to keep token costs low. Use `sctx decisions` to query decisions separately.

If no context matches, exits 0 with no output (a no-op for Claude Code).

The Write tool gets special treatment: `sctx` checks whether the target file exists on disk to distinguish `create` (new file) from `edit` (existing file).

## sctx context \<path\>

Query context entries for a file. Useful for debugging and testing your context files.

```bash
sctx context src/api/handler.py
sctx context src/api/handler.py --on edit --when before
sctx context src/api/handler.py --json
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--on <action>` | `all` | Filter by action: `read`, `edit`, `create`, `all` |
| `--when <timing>` | `before` | Filter by timing: `before`, `after` |
| `--json` | off | Output as JSON instead of human-readable text |

## sctx decisions \<path\>

Query decisions for a file. Shows architectural decisions that apply based on glob matching.

```bash
sctx decisions src/api/handler.py
sctx decisions src/api/handler.py --json
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--json` | off | Output as JSON instead of human-readable text |

## sctx validate [\<dir\>]

Validates all `AGENTS.yaml` and `AGENTS.yaml` files found in a directory tree. Reports schema errors and invalid glob patterns.

```bash
sctx validate
sctx validate ./src
```

Defaults to the current directory if no path is given.

Exit code 0 if all files are valid. Exit code 1 if any errors are found. Warnings (like unknown fields) don't cause a non-zero exit.

## sctx init

Creates a starter `AGENTS.yaml` in the current directory with commented examples.

```bash
sctx init
```

Refuses to overwrite an existing `AGENTS.yaml`.

## sctx claude enable

Installs the `sctx hook` into your project's `.claude/settings.local.json` (personal, not committed to version control). Creates the file and directory if they don't exist. If hooks are already configured, it leaves them alone. To share hooks with all contributors, manually add the hook config to `.claude/settings.json` instead.

```bash
sctx claude enable
```

## sctx claude disable

Removes the `sctx hook` entries from `.claude/settings.local.json`.

```bash
sctx claude disable
```

## sctx version

Prints the version.

```bash
sctx version
```

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success (includes "no context matched" -- that's not an error) |
| 1 | Fatal error: invalid arguments, IO failure, validation errors |
