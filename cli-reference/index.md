# CLI reference

## sctx hook

Reads agent hook input from stdin, resolves matching context entries, and writes the response to stdout. This is the primary integration point for AI agents.

```
echo '{"tool_name":"Edit","tool_input":{"file_path":"/project/src/main.py"},"hook_event_name":"PreToolUse"}' | sctx hook
```

Supports Claude Code and pi JSON formats. The source is auto-detected: input with `"source": "pi"` is routed to the pi adapter; all other input is treated as Claude Code format. The `cwd` field determines the project root — only `AGENTS.yaml` files at or below this directory are considered.

Context entries are always included in hook output when they match. Decisions are also included when Claude Code's `permission_mode` is `"plan"` — this surfaces architectural decisions during planning, before the agent writes any code. Outside plan mode, decisions are excluded to keep token costs low. Use `sctx decisions` to query decisions separately regardless of mode.

If no context matches, exits 0 with no output (a no-op for Claude Code).

The Write tool gets special treatment: `sctx` checks whether the target file exists on disk to distinguish `create` (new file) from `edit` (existing file).

## sctx context \<path>

Query context entries for a file or directory. Useful for debugging and testing your context files. The current working directory is used as the project root — only `AGENTS.yaml` files at or below it are considered.

If the path exists on disk as a directory, sctx automatically runs a directory query. For paths that don't exist on disk, append a trailing `/` to force a directory query. Directory patterns like `match: ["tests/"]` only match directory queries, not file queries. File-glob patterns like `match: ["**/*.py"]` match a directory query if they could produce hits inside that directory.

```
sctx context src/api/handler.py
sctx context src/api/handler.py --on edit --when before
sctx context src/api/                                     # directory query
sctx context src/api/handler.py --json
sctx context --all                                        # every entry from every AGENTS.yaml
sctx context --all --on edit --json
```

Use `--all` to dump every context entry from every `AGENTS.yaml` file in the tree, skipping glob matching entirely. Output includes the source file path and match patterns for each entry so you can see where each entry is defined and what it targets. `--all` and `<path>` are mutually exclusive. `--on` and `--when` filters still apply with `--all`.

### Flags

| Flag              | Default | Description                                                       |
| ----------------- | ------- | ----------------------------------------------------------------- |
| `--all`           | off     | Return all entries from all AGENTS.yaml files, skip glob matching |
| `--on <action>`   | `all`   | Filter by action: `read`, `edit`, `create`, `all`                 |
| `--when <timing>` | `all`   | Filter by timing: `before`, `after`, `all`                        |
| `--json`          | off     | Output as JSON instead of human-readable text                     |

## sctx decisions \<path>

Query decisions for a file or directory. Shows architectural decisions that apply based on glob matching. Directory queries work the same way as `sctx context` -- pass a directory path to see decisions scoped to that directory.

```
sctx decisions src/api/handler.py
sctx decisions src/api/                                    # directory query
sctx decisions src/api/handler.py --json
sctx decisions --all                                       # every decision from every AGENTS.yaml
sctx decisions --all --json
```

Use `--all` to dump every decision entry from every `AGENTS.yaml` file in the tree. Like `sctx context --all`, output includes source file paths and match patterns. `--all` and `<path>` are mutually exclusive.

### Flags

| Flag     | Default | Description                                                       |
| -------- | ------- | ----------------------------------------------------------------- |
| `--all`  | off     | Return all entries from all AGENTS.yaml files, skip glob matching |
| `--json` | off     | Output as JSON instead of human-readable text                     |

## sctx validate [\<dir>]

Validates all `AGENTS.yaml` and `AGENTS.yml` files found in a directory tree. Reports schema errors and invalid glob patterns.

```
sctx validate
sctx validate ./src
```

Defaults to the current directory if no path is given.

Exit code 0 if all files are valid. Exit code 1 if any errors are found. Warnings (like unknown fields) don't cause a non-zero exit.

## sctx init

Creates a starter `AGENTS.yaml` in the current directory with commented examples.

```
sctx init
```

Refuses to overwrite an existing `AGENTS.yaml`.

## sctx claude enable

Installs the `sctx hook` into your project's `.claude/settings.local.json`. Creates the settings file if it doesn't exist. Requires the `.claude/` directory to already be present (i.e., you've run `claude` in this project at least once). If hooks are already configured, it leaves them alone.

```
sctx claude enable
```

## sctx claude disable

Removes the `sctx hook` entries from `.claude/settings.local.json`.

```
sctx claude disable
```

## sctx pi enable

Installs a thin TypeScript extension at `.pi/extensions/sctx.ts` that hooks into pi's `tool_call` and `tool_result` events and forwards them to `sctx hook`. For mutating tools (`edit`, `write`), the extension blocks the tool call and surfaces context before the edit occurs. For all other tools, context is appended to the tool result. Requires a `.pi/` directory to exist in the current directory.

```
sctx pi enable
```

## sctx pi disable

Removes the sctx extension from `.pi/extensions/sctx.ts`. Cleans up the `extensions/` directory if empty.

```
sctx pi disable
```

## sctx version

Prints the version.

```
sctx version
```

## Exit codes

| Code | Meaning                                                        |
| ---- | -------------------------------------------------------------- |
| 0    | Success (includes "no context matched" -- that's not an error) |
| 1    | Fatal error: invalid arguments, IO failure, validation errors  |
