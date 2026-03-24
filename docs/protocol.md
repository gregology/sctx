---
title: Protocol specification
description: File format, resolution algorithm, and merge behavior for implementors
---

# Protocol specification

This is the spec for the Structured Context file format. It covers file discovery, resolution, merge behavior, and validation. It's aimed at people building tools that read or write `AGENTS.yaml` files.

If you're writing `AGENTS.yaml` files for your project (not building a tool), start with [Context entries](context.md) and [Decisions](decisions.md) instead.

## Context files

### Recognized filenames

| Filename | Notes |
|---|---|
| `AGENTS.yaml` | Primary name |
| `AGENTS.yml` | Alternate extension |

Both are standard YAML extensions. The protocol accepts both. If multiple context files exist in the same directory, all are loaded and their contents merged.

### Placement

Context files can appear in any directory. Tools discover them by walking up from the target file to the project root, collecting every context file found along the way.

### Project root

The project root is the working directory where the tool was launched. Not detected via file markers. This is deliberate: marker-based detection (`.git`, `pyproject.toml`, etc.) breaks in monorepos where subdirectories contain their own project markers.

- **Hook mode** (`sctx hook`): The root is the `cwd` field from the agent's JSON input. For Claude Code, this is the directory where `claude` was started.
- **CLI mode** (`sctx context`, `sctx decisions`): The root is the current working directory where `sctx` is run.

Only `AGENTS.yaml` files at or below the root are considered. Files above the root are never seen.

### Missing files

If no context files exist in the project, tools should emit a warning and return gracefully. Missing files are not errors.

## Schema

A context file has two optional top-level keys:

```yaml
context:
  - # ... context entries

decisions:
  - # ... decision entries
```

Both are lists. Both are optional.

See [Context entries](context.md) for the full context entry schema and [Decisions](decisions.md) for the full decision entry schema.

### Context entry fields (summary)

| Field | Type | Required | Default |
|---|---|---|---|
| `content` | string | yes | -- |
| `match` | list of globs | no | `["**"]` |
| `exclude` | list of globs | no | `[]` |
| `on` | string or list | no | `all` |
| `when` | string | no | `before` |

### Decision entry fields (summary)

| Field | Type | Required | Default |
|---|---|---|---|
| `decision` | string | yes | -- |
| `rationale` | string | yes | -- |
| `alternatives` | list | no | -- |
| `revisit_when` | string | no | -- |
| `date` | date | no | -- |
| `match` | list of globs | no | `["**"]` |

## Resolution algorithm

Given a file path, an action, and a timing:

1. **Discover** -- Walk from the target file's directory up to the project root, collecting all context files at each level
2. **Parse** -- Parse each file. Emit warnings for invalid files but continue processing valid ones
3. **Filter by match/exclude** -- Test each entry's glob patterns against the target file path. Globs are relative to the context file's directory
4. **Filter by action** -- Keep entries where `on` includes the requested action (or is `all`)
5. **Filter by timing** -- Keep entries where `when` matches the requested timing
6. **Merge** -- Combine all matching entries. Parent directory entries come first, child directory entries come last
7. **Return** -- The ordered list of matching context entries and decisions

## Merge order

Parent directories come before child directories.

- General project-level context appears first (lower specificity)
- Directory-specific context appears last (higher specificity, stronger recency in the prompt)

This ordering is intentional. The most specific context gets the strongest position in the LLM's attention window.

## Validation rules

- `content` is required on every context entry
- `decision` and `rationale` are required on every decision entry
- `on` values must be: `read`, `edit`, `create`, `all`, or a list of these
- `when` values must be: `before`, `after`, or `all`
- `match` and `exclude` must be valid glob patterns
- `date` must be YYYY-MM-DD if present
- Unknown fields produce warnings, not errors (forward compatibility)

## Structured Context

- This is a Go project. Follow idiomatic Go conventions: short variable names, table-driven tests, no assertion libraries.
- The core engine (internal/core) must have zero knowledge of any specific AI agent. Agent-specific logic lives in internal/adapter.
- Both AGENTS.yaml and AGENTS.yml are recognized. If both exist in the same directory, they are loaded and merged.
