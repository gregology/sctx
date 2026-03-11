---
title: Protocol specification
description: The full Structured Context file format, schema, resolution algorithm, and merge behavior
---

# Protocol specification

This defines the Structured Context protocol: the file format, schema, resolution algorithm, and merge behavior. It's written for implementors building tools that read or write context files.

## Context files

### Recognized filenames

| Filename | Notes |
|---|---|
| `AGENTS.yaml` | Primary name |
| `AGENTS.yml` | Alternate extension |

Both `.yaml` and `.yml` are standard. The protocol accepts both.

If multiple context files exist in the same directory, all are loaded and their contents merged.

### Placement

Context files can appear in any directory. Tools discover them by walking up from the target file to the project root, collecting every context file found along the way.

### Project root

The project root is the working directory where the tool was launched — not detected via file markers. This is deliberate: marker-based detection (`.git`, `pyproject.toml`, etc.) breaks in monorepos where subdirectories contain their own project markers.

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

### Context entries

Each entry is an atomic piece of guidance.

```yaml
context:
  - content: "Use snake_case for all identifiers"
    match: ["**/*.py"]
    exclude: ["**/vendor/**"]
    on: edit
    when: after
```

#### Fields

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `content` | string | yes | -- | The context to deliver |
| `match` | list of globs | no | `["**"]` | File patterns this applies to |
| `exclude` | list of globs | no | `[]` | File patterns to exclude |
| `on` | string or list | no | `all` | Action filter: `read`, `edit`, `create`, or `all` |
| `when` | string | no | `before` | Prompt positioning: `before`, `after`, or `all` |

#### `content`

A self-contained, actionable piece of guidance. Prefer multiple focused entries over one large entry. Each entry should make sense on its own without needing to read other entries.

#### `match` and `exclude`

Standard glob patterns. Same syntax as `.gitignore`, `.editorconfig`, and similar tools.

Globs are resolved relative to the directory containing the context file, not the project root. A pattern `**/*.py` in `src/api/AGENTS.yaml` matches Python files under `src/api/`, not the entire project.

The default match is `["**"]` (recursive, everything). Use `*` for single-level matching.

`exclude` is applied after `match`. A file must match at least one `match` pattern and zero `exclude` patterns.

#### `on`

The action being performed on the file.

| Value | Meaning |
|---|---|
| `read` | Agent is reading the file |
| `edit` | Agent is modifying an existing file |
| `create` | Agent is creating a new file |
| `all` | Any action (default) |

Accepts a single string (`on: edit`) or a list (`on: [edit, create]`).

#### `when`

Controls where context appears relative to the file content in the agent's prompt.

| Value | Meaning |
|---|---|
| `before` | Context appears before file content. Default. Good for general guidelines. |
| `after` | Context appears after file content. Use for high-priority instructions. |
| `all` | Context appears for both `before` and `after` timing requests. |

This matters because LLMs exhibit primacy and recency effects. They weight the beginning and end of their context window more heavily. If a piece of context is critical and the file is large, placing it `after` gives it stronger influence on the model's output.

### Decision entries

Decisions capture architectural choices, their rationale, and the alternatives that were considered. This prevents agents from re-litigating settled choices or introducing patterns that were deliberately rejected.

```yaml
decisions:
  - decision: "REST over GraphQL for public APIs"
    rationale: "Team expertise and simpler caching"
    alternatives:
      - option: "GraphQL"
        reason_rejected: "Team has no GraphQL experience, caching is complex"
      - option: "gRPC"
        reason_rejected: "Public API needs browser compatibility"
    revisit_when: "We need real-time subscriptions or complex nested queries"
    date: 2025-10-20
    match: ["src/api/**"]
```

#### Fields

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `decision` | string | yes | -- | What was decided |
| `rationale` | string | yes | -- | Why this was chosen |
| `alternatives` | list | no | -- | Other options that were considered |
| `revisit_when` | string | no | -- | Condition to reconsider |
| `date` | date | no | -- | When it was made (YYYY-MM-DD) |
| `match` | list of globs | no | `["**"]` | Scope to specific files |

#### `alternatives`

Each alternative has:

| Field | Type | Required | Description |
|---|---|---|---|
| `option` | string | yes | The alternative that was considered |
| `reason_rejected` | string | yes | Why it wasn't chosen |

Including alternatives gives agents (and humans) the full picture. Without them, an agent might suggest the exact thing you already evaluated and rejected.

#### `revisit_when`

Decisions are made under constraints. When constraints change, the decision may no longer hold. Making the trigger condition explicit lets agents flag stale decisions rather than blindly following them.

## Resolution algorithm

Given a file path, an action, and a timing:

1. **Discover** -- Walk from the target file's directory up to the project root (the working directory), collecting all context files at each level
2. **Parse** -- Parse each file. Emit warnings for invalid files but continue processing valid ones
3. **Filter by match/exclude** -- Test each entry's glob patterns against the target file path. Globs are relative to the context file's directory
4. **Filter by action** -- Keep entries where `on` includes the requested action (or is `all`)
5. **Filter by timing** -- Keep entries where `when` matches the requested timing
6. **Merge** -- Combine all matching entries. Parent directory entries come first, child directory entries come last
7. **Return** -- The ordered list of matching context entries and decisions

### Merge order

Parent directories come before child directories. This means:

- General project-level context appears first (lower specificity)
- Directory-specific context appears last (higher specificity, stronger recency in the prompt)

This ordering is intentional. The most specific, most relevant context gets the strongest position in the LLM's attention.

### Validation rules

- `content` is required on every context entry
- `decision` and `rationale` are required on every decision entry
- `on` values must be: `read`, `edit`, `create`, `all`, or a list of these
- `when` values must be: `before`, `after`, or `all`
- `match` and `exclude` must be valid glob patterns
- `date` must be YYYY-MM-DD if present
- Unknown fields produce warnings, not errors (forward compatibility)
