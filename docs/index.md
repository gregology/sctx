# Structured Context

Structured Context is a protocol for storing scoped, machine-readable context alongside your code. It gives AI agents the right guidance at the right time, for the right files.

## The problem

AI coding agents read instruction files like `AGENTS.md` to understand how to work in a codebase. These files are blunt instruments. They're scoped to a directory, they're unstructured prose, and every instruction in them gets loaded regardless of what the agent is actually doing.

This falls apart fast.

Consider a dbt project. A single `models/` directory contains SQL model files, YAML schema files, and markdown documentation files. The context an agent needs for each of these is completely different:

- When editing a **SQL model**, the agent should know about your SQL style conventions, which macros are available, and that you use incremental models with a specific strategy.
- When editing a **YAML schema**, it should know your testing conventions, required fields for every model, and how you structure column descriptions.
- When editing a **markdown doc**, it should know your documentation template and that business stakeholders read these files.

With `AGENTS.md`, your only option is one big file that says "if you're editing SQL do X, if you're editing YAML do Y, if you're editing markdown do Z." That's conditional logic written as prose. The agent has to parse natural language to figure out which instructions apply. That's wasteful and error-prone.

This pattern shows up everywhere:

- **API directories** with route handlers, middleware, tests, and OpenAPI specs
- **React component directories** with `.tsx`, `.test.tsx`, `.stories.tsx`, and `.module.css` files
- **Infrastructure directories** with Terraform `.tf` files, `.tfvars`, and documentation
- **Monorepo packages** where shared conventions apply globally but each package has its own patterns

The underlying issue: files in the same directory often have very different context requirements. Directory-scoped unstructured text can't express this.

## The approach

Structured Context defines a YAML-based file format (`CONTEXT.yaml`) that lets you:

- **Scope context to specific files** using glob patterns (`match: ["**/*.sql"]`)
- **Filter by action** -- different guidance for reading vs editing vs creating files
- **Control prompt positioning** -- place context before or after file content based on importance
- **Capture decisions** -- record what you chose, why, what else you considered, and when to revisit

Context files are placed throughout your codebase and merge with parent directories, so you get inheritance without duplication.

The protocol is agent-agnostic. The reference implementation (`sctx`) ships with a Claude Code adapter, but the core engine has no knowledge of any specific agent. Other tools can adopt the file format directly.

Context entries and decisions are queried separately. `sctx context` returns actionable guidance for a file. `sctx decisions` returns the architectural decisions that apply. The hook integration (`sctx hook`) only sends context entries to the agent, keeping token costs low. Decisions are there for humans and for agents that explicitly ask.

## Why this matters beyond better output

The obvious benefit is accuracy -- agents follow your conventions instead of guessing. But precise context has compounding effects:

**Smaller models become viable.** A large model can sometimes infer the right pattern from a sprawling AGENTS.md. A smaller model can't. But give a small model exactly the right context for the file it's touching, and it performs surprisingly well. Structured Context closes the gap between model tiers by compensating with better input.

**Costs drop.** Less irrelevant context means fewer input tokens. If your AGENTS.md is 2,000 tokens but only 200 are relevant to the current file, you're paying for 1,800 tokens of noise on every tool call. Multiply that across a session with hundreds of file operations.

**Responses get faster.** Fewer input tokens means lower latency. For agents in the hot loop of an edit-test cycle, shaving tokens off every call adds up.

**Accuracy improves.** LLMs degrade when context is long and diluted. Giving the model 5 focused sentences instead of 50 scattered paragraphs means it's more likely to actually follow the instructions. The signal-to-noise ratio of your prompt directly affects output quality.

## Next steps

- [Getting started](getting-started.md) -- install `sctx` and create your first context file
- [Protocol specification](protocol.md) -- the full spec for implementors
- [Examples](examples.md) -- real-world patterns for common project types
- [CLI reference](cli-reference.md) -- `sctx` commands and flags
- [Contributing](contributing.md) -- how to work on the project
- [Roadmap](roadmap.md) -- what's planned
