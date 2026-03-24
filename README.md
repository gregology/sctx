# Structured Context

Scoped, structured context for AI agents. Drop `AGENTS.yaml` files into your codebase and agents get the right guidance at the right time, for the right files.

Think of it as `AGENTS.md` with precision. Instead of dumping everything into one big file, you scope context to specific files, actions, and timing.

## Quick start

```bash
brew install gregology/tap/sctx
```

On Ubuntu/Debian:

```bash
curl -Lo sctx.deb https://github.com/gregology/sctx/releases/download/latest/sctx_linux_amd64.deb
sudo dpkg -i sctx.deb
```

On Fedora/RHEL/CentOS:

```bash
curl -Lo sctx.rpm https://github.com/gregology/sctx/releases/download/latest/sctx_linux_amd64.rpm
sudo rpm -i sctx.rpm
```

Or with Go:

```bash
go install github.com/gregology/sctx/cmd/sctx@latest
```

Create a `AGENTS.yaml` anywhere in your project:

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

`sctx` walks up the directory tree from the target file to the project root, collecting `AGENTS.yaml` files along the way. It filters entries by glob pattern, action type, and timing, then returns the matching context.

The project root is the working directory — the directory where the tool was launched. In hook mode, this is the `cwd` from the agent's input (e.g. where `claude` was started). In CLI mode, it's where you run `sctx`. Only `AGENTS.yaml` files at or below the root are considered.

Parent directory context merges with child directory context. Entries from parent directories appear first (lower specificity), entries from closer directories appear last (higher specificity, stronger recency in the LLM prompt).

### Context entries

Each entry has a `content` string and optional filters:

- **match** - glob patterns for files this applies to (default: `["**"]`, everything)
- **exclude** - glob patterns to skip
- **on** - when the file is being `read`, `edit`ed, `create`d, or `all` (default)
- **when** - deliver `before` or `after` the file content in the prompt, or `all` for both (default: `before`)

The `when` field matters because LLMs weight the start and end of context more heavily. Put high-priority instructions `after` so they land close to where the model generates its response.

See [sctx.dev/context](https://sctx.dev/context/) for detailed field documentation.

### Decisions

The code shows what you chose. Decisions capture what you *didn't* choose and why you rejected it. That's the part that's invisible in a codebase and the part agents keep getting wrong -- suggesting tools and patterns you already evaluated and ruled out.

```yaml
decisions:
  - decision: "REST over GraphQL for public APIs"
    rationale: "Team expertise, simpler caching"
    alternatives:
      - option: "GraphQL"
        reason_rejected: "Team has no GraphQL experience, caching is complex"
      - option: "gRPC"
        reason_rejected: "Public API needs browser compatibility"
    revisit_when: "We need real-time subscriptions"
    date: 2025-10-20
    match: ["src/api/**"]
```

`decision` and `rationale` are required. `alternatives`, `revisit_when`, `date`, and `match` are optional. The `alternatives` field is where most of the value lives as each rejected option records the specific constraint that killed it, so agents know not to suggest it again. `revisit_when` captures the condition under which the constraint might change, turning a static decision into one that can expire gracefully.

See [sctx.dev/decisions](https://sctx.dev/decisions/) for the full "nos" framing and field documentation.

### Recognized filenames

Both are picked up. If both exist in the same directory, they get loaded and merged.

- `AGENTS.yaml`
- `AGENTS.yml`

## Claude Code integration

Add hooks to `.claude/settings.local.json`:

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

Or let sctx configure it automatically:

```
sctx claude enable
```

`sctx hook` reads the hook JSON from stdin, figures out the file path and action from the tool call, resolves matching context entries, and returns them as `additionalContext` in Claude Code's expected format. Decisions are not included in hook output to keep the token cost low. If nothing matches, it exits silently.

The Write tool gets special handling: `sctx` checks whether the file exists on disk to distinguish `create` from `edit`.

## Example

I have added the following context to this project's root `AGENTS.yaml` file.

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

**sctx init** - Drops a starter `AGENTS.yaml` with commented examples into the current directory.

**sctx claude enable** - Install sctx hooks in Claude Code settings

**sctx claude disable** - Remove sctx hooks from Claude Code settings

**sctx pi enable** - Install sctx extension for pi

**sctx pi disable** - Remove sctx extension for pi

**sctx version** - Print version

## How does sctx compare?

| Tool | Scope | Format | Delivery |
|------|-------|--------|----------|
| AGENTS.md | Directory | Unstructured prose | Always loaded |
| MCP | External tools & data | RPC protocol | On demand via server |
| .cursorrules | Project root | Monolithic prompt | Always loaded |
| **sctx** | **Per-file, per-action** | **Declarative YAML** | **JIT, glob-matched** |

`sctx` is not a replacement for these tools — it fills a different gap. MCP connects agents to external systems and `AGENTS.md` captures broad project guidance. `sctx` adds **fine-grained, file-targeted, action-filtered context** so the agent only sees what's relevant to the file it's touching right now.

See the full breakdown at [sctx.dev/comparisons](https://sctx.dev/comparisons/).

## pi integration

Install the sctx extension into your project's `.pi/extensions/` directory:

```bash
sctx pi enable
```

This creates a thin TypeScript extension that hooks into pi's `tool_call` and `tool_result` events. When pi reads, writes, or edits a file, the extension forwards the event to `sctx hook` and injects any matching context into the tool result.

To remove:

```bash
sctx pi disable
```

## Agent-agnostic design

The core engine knows nothing about Claude Code, pi, or any other agent. It takes a file path, an action, and a timing, and returns matched context. Agent-specific bits live in thin adapter layers that translate stdin JSON into those universal inputs.

Other agents can use `sctx context` directly, or new adapters can be added without touching the core.
