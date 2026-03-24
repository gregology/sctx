# Context entries

Context entries are the core building block of an `AGENTS.yaml` file. Each one is a piece of guidance scoped to specific files and actions.

```
context:
  - content: "Use snake_case for all identifiers"
    match: ["**/*.py"]
    on: edit
    when: before
```

When an agent edits a Python file, it sees that instruction. When it edits a SQL file, it doesn't. That's the whole idea.

## Fields

| Field     | Type           | Required | Default  | Description                                       |
| --------- | -------------- | -------- | -------- | ------------------------------------------------- |
| `content` | string         | yes      | --       | The guidance to deliver                           |
| `match`   | list of globs  | no       | `["**"]` | File patterns this applies to                     |
| `exclude` | list of globs  | no       | `[]`     | File patterns to skip                             |
| `on`      | string or list | no       | `all`    | Action filter: `read`, `edit`, `create`, or `all` |
| `when`    | string         | no       | `before` | Prompt positioning: `before`, `after`, or `all`   |

## content

The actual instruction. Keep each entry focused on one concern. If you're writing a paragraph that covers naming conventions *and* error handling *and* testing patterns, split it into separate entries. Each entry should make sense on its own.

```
# Too much in one entry
context:
  - content: |
      Use snake_case. Always handle errors with Result types.
      Tests go in _test.go files and use table-driven patterns.
      Never use globals.
    match: ["**/*.go"]

# Better: one concern per entry, each independently scoped
context:
  - content: "Use snake_case for all identifiers"
    match: ["**/*.go"]
    on: [edit, create]

  - content: "Handle errors with Result types, never panic in library code"
    match: ["**/*.go"]
    exclude: ["**/*_test.go"]
    on: [edit, create]

  - content: "Table-driven tests. No assertion libraries."
    match: ["**/*_test.go"]
    on: [edit, create]
```

Splitting entries matters because each one can have its own `match`, `on`, and `when` filters. The error handling guidance doesn't apply to test files. The testing guidance only applies to test files. You can't express that with one big entry.

## match and exclude

Standard glob patterns. Same syntax as `.gitignore` and `.editorconfig`.

Globs resolve relative to the directory containing the `AGENTS.yaml` file, not the project root. A pattern `**/*.py` in `src/api/AGENTS.yaml` matches Python files under `src/api/`, not the entire project.

```
# Matches all Python files in this directory tree
match: ["**/*.py"]

# Matches only direct children, not nested files
match: ["*.py"]

# Multiple patterns
match: ["**/*.ts", "**/*.tsx"]

# Match everything except vendor code
match: ["**/*.py"]
exclude: ["**/vendor/**"]
```

The default match is `["**"]` (recursive, everything). `exclude` is applied after `match`. A file must match at least one `match` pattern and zero `exclude` patterns.

## on

What the agent is doing with the file.

| Value    | Meaning                             |
| -------- | ----------------------------------- |
| `read`   | Agent is reading the file           |
| `edit`   | Agent is modifying an existing file |
| `create` | Agent is creating a new file        |
| `all`    | Any action (default)                |

Accepts a single string or a list:

```
on: edit
on: [edit, create]
```

This filter is useful because reading and writing need different guidance. When an agent reads a file, you might want it to understand the architecture. When it edits, you want it to follow conventions. When it creates, you might want it to include boilerplate like license headers or module docstrings.

## when

Where the context appears relative to the file content in the agent's prompt.

| Value    | Meaning                                       |
| -------- | --------------------------------------------- |
| `before` | Context appears before file content (default) |
| `after`  | Context appears after file content            |
| `all`    | Both positions                                |

This matters because LLMs have primacy and recency bias. They pay more attention to the beginning and end of their context window. If the file is large (hundreds of lines) and you have a critical instruction, putting it `after` means it lands right before the model generates its response. That gives it stronger influence.

General guidelines work fine as `before`. High-priority rules that agents keep ignoring should go `after`.

## Writing good entries

**Be specific.** "Follow best practices" is useless. "Prefix staging models with `stg_`, intermediate with `int_`, marts with `fct_` or `dim_`" is actionable.

**Be self-contained.** Each entry should make sense without reading other entries. An agent might only see two of your ten entries for a given file. Don't write entry #4 assuming the agent has already read entries #1-3.

**Scope tightly.** The more specific your `match` pattern, the less noise the agent sees. `match: ["handlers/**/*.py"]` is better than `match: ["**/*.py"]` when the guidance only applies to handlers.

**Use `exclude` for exceptions.** Test files, generated code, vendor directories. If a convention applies to your code but not to third-party code, exclude it:

```
context:
  - content: "All functions need docstrings"
    match: ["**/*.py"]
    exclude: ["**/vendor/**", "**/*_pb2.py"]
```

## Directory inheritance

Context files at different levels merge together. Parent entries come first, child entries come last.

```
project/
  AGENTS.yaml           <- "Use ESM imports everywhere"
  src/
    payments/
      AGENTS.yaml       <- "Money values are integers in cents"
      checkout.ts
```

When an agent edits `checkout.ts`, it sees the project-level ESM rule first, then the payments-specific money rule. The payments context appears last, giving it stronger recency in the prompt.

You don't need to repeat parent context in child files. It's inherited automatically.

See [Examples](https://sctx.dev/examples/index.md) for complete `AGENTS.yaml` files showing context entries in real projects.
