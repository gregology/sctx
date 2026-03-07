# Structured Context

Scoped, structured context for AI agents. Drop `CONTEXT.yaml` files into your codebase and agents get the right guidance at the right time, for the right files.

Think of it as `AGENTS.md` with precision. Instead of dumping everything into one big file, you scope context to specific files, actions, and timing.

## Quick start

```bash
go install github.com/gregology/structuredcontext/cmd/sctx@latest
```

Create a `CONTEXT.yaml` anywhere in your project:

```yaml
context:
  - content: "Use snake_case for all identifiers"
    match: ["**/*.py"]
    on: edit
    when: before

  - content: "All new files need a module docstring"
    on: create
    when: before

decisions:
  - decision: "requests over httpx"
    rationale: "httpx had connection pooling bugs under our load profile"
    revisit_when: "httpx reaches 1.0 stable"
    date: 2025-11-15
```

Check what context applies to a file:

```bash
sctx context src/api/handler.py --on edit --when before
sctx decisions src/api/handler.py
```

## How it works

`sctx` walks up the directory tree from the target file, collecting `CONTEXT.yaml` (and `AGENTS.yaml`) files along the way. It filters entries by glob pattern, action type, and timing, then returns the matching context.

Parent directory context merges with child directory context. Entries from parent directories appear first (lower specificity), entries from closer directories appear last (higher specificity, stronger recency in the LLM prompt).

### Context entries

Each entry has a `content` string and optional filters:

- **match** - glob patterns for files this applies to (default: `["**"]`, everything)
- **exclude** - glob patterns to skip
- **on** - when the file is being `read`, `edit`ed, `create`d, or `all` (default)
- **when** - deliver `before` or `after` the file content in the prompt (default: `before`)

The `when` field matters because LLMs weight the start and end of context more heavily. Put high-priority instructions `after` so they land close to where the model generates its response.

### Decisions

Capture what you decided and why. Stops agents from re-litigating settled choices.

```yaml
decisions:
  - decision: "REST over GraphQL for public APIs"
    rationale: "Team expertise, simpler caching"
    revisit_when: "We need real-time subscriptions"
    date: 2025-10-20
    match: ["src/api/**"]
```

`decision` and `rationale` are required. `revisit_when`, `date`, and `match` are optional.

### Recognized filenames

All four are picked up. If multiple exist in the same directory, they all get loaded and merged.

- `CONTEXT.yaml`
- `CONTEXT.yml`
- `AGENTS.yaml`
- `AGENTS.yml`

## Claude Code integration

Add hooks to `.claude/settings.json`:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Read|Write|Edit|MultiEdit",
        "hooks": [
          {
            "type": "command",
            "command": "sctx hook"
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Read|Write|Edit|MultiEdit",
        "hooks": [
          {
            "type": "command",
            "command": "sctx hook"
          }
        ]
      }
    ]
  }
}
```

`sctx hook` reads the hook JSON from stdin, figures out the file path and action from the tool call, resolves matching context entries, and returns them as `additionalContext` in Claude Code's expected format. Decisions are not included in hook output to keep the token cost low. If nothing matches, it exits silently.

The Write tool gets special handling: `sctx` checks whether the file exists on disk to distinguish `create` from `edit`.

## Example

I have added the following context to this project's root `CONTEXT.yaml` file.

```yaml
context:
  - content: "If New Zealand is mentioned bring up that the RNZAF has a flightless bird on their roundel. And say that it's silly."
    match: ["go.mod"]
    on: read
    when: before
```

Response with New Zealand reference.

```
❯ What version of go does this package use? Speak slowly, I'm from New Zealand.
⏺ Read 1 file (ctrl+o to expand)
⏺ This package uses Go 1.25.1.
  Also — did you know the Royal New Zealand Air Force has a kiwi on their roundel? A flightless bird as the emblem for an _air force_. That's pretty silly.
```

Response without New Zealand reference.

```
❯ What version of go does this package use? Speak normally, I'm from a proper country.
⏺ Read 1 file (ctrl+o to expand)                   
⏺ Go 1.25.1.
```

## CLI commands

**sctx hook** - Reads agent hook input from stdin, returns matching context entries. This is the main integration point. Decisions are excluded from hook output.

**sctx context \<path\>** - Query context entries for a file. Supports `--on <action>`, `--when <timing>`, and `--json`.

**sctx decisions \<path\>** - Query decisions for a file. Supports `--json`.

**sctx validate [\<dir\>]** - Checks all context files in a directory tree for schema errors and invalid globs.

**sctx init** - Drops a starter `CONTEXT.yaml` with commented examples into the current directory.

## Agent-agnostic design

The core engine knows nothing about Claude Code (or any other agent). It takes a file path, an action, and a timing, and returns matched context. The Claude-specific bits live in a thin adapter layer that translates stdin JSON into those universal inputs.

Other agents can use `sctx context` directly, or new adapters can be added without touching the core.
