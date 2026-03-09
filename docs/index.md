---
title: Structured Context
description: Give AI agents the right context at the right time
---

# Structured Context

AI agents work best with focused & relevant context. Existing context standards like `AGENTS.md` are broad. The result is wasted cycles because of wrong patterns applied, conventions broken, and then more calls spent fixing the damage.

Structured Context gives you fine-grained control over what an agent sees and when. Drop `AGENTS.yaml` files into your codebase and the right guidance shows up for the right files, filtered by what the agent is actually doing.

!!! tip "JIT context, not monolithic prompts"
    Unlike `.cursorrules`, `AGENTS.md`, and other all-at-once approaches, Structured Context delivers **just-in-time, file-targeted context** — the agent only sees instructions that match the file it's touching and the action it's performing. [See how sctx compares to other tools.](comparisons.md)

## Quick start

```bash
brew install gregology/tap/sctx
sctx init
sctx claude enable
```

This installs `sctx`, creates an `AGENTS.yaml` with a test context entry, and hooks it into Claude Code. Try it out — ask your agent:

> Give me a very concise description of this project. Explain it like I'm 5 as I'm from New Zealand.

If everything is working, the agent will read your README and mention that the RNZAF has a flightless bird on their roundel (because the starter `AGENTS.yaml` tells it to). Though sometimes these agents are too smart and will ignore unrelated requests so you can just ask something like `Did you receive any context about the RNZAF in your prehooks?` to verify.

Once verified, replace the example with your own context entries.

See [Getting started](getting-started.md) for more install options and details.

## The problem

Instruction files like `AGENTS.md` and `CLAUDE.md` are good for project-wide guidance. But they're directory-scoped and unstructured, so every instruction gets loaded regardless of what the agent is actually doing. As they grow, LLMs start losing signal in the noise. Recency bias kicks in, attention dilutes, and the model quietly ignores the instruction that mattered most.

The real gap is granularity. Files in the same directory often need completely different context.

Take a dbt project. A single `models/` directory contains SQL model files, YAML schema files, and markdown docs. When an agent edits a SQL model, it should know about your SQL style, available macros, and incremental strategy. When it edits a YAML schema, it needs your testing conventions and required fields. Markdown docs need your documentation templates.

You can put all of this in one `AGENTS.md` where every paragraph starts with "if you're editing SQL do X, if you're editing YAML do Y." That works, but it's conditional logic written as prose. The agent has to parse natural language to figure out which instructions apply, and as the file gets longer it increasingly won't.

This pattern shows up everywhere. API directories with route handlers, middleware, and OpenAPI specs. React component directories with `.tsx`, `.test.tsx`, `.stories.tsx`, and `.module.css` files. Monorepo packages where shared conventions apply globally but each package has its own patterns.

## The approach

Structured Context defines a YAML-based file format (`AGENTS.yaml`) that lets you:

- **Scope context to specific files** using glob patterns (`match: ["**/*.sql"]`)
- **Filter by action** -- different guidance for reading vs editing vs creating files
- **Control prompt positioning** -- place context before or after file content based on importance
- **Capture decisions** -- record what you chose, why, what else you considered, and when to revisit

Context files are placed throughout your codebase and merge with parent directories, so you get inheritance without duplication.

The protocol is agent-agnostic. The reference implementation (`sctx`) ships with a Claude Code adapter, but the core engine has no knowledge of any specific agent. Other tools can adopt the file format directly.

## Why this matters

Precise context compounds.

**Smaller models become viable.** A large model can sometimes power through a long instruction file. A smaller model can't. It gets lost, introduces anti-patterns, burns cycles reverting its own mistakes. But a small model with exactly the right context can match a large model running on vague context. Better input closes the gap between model tiers.

**Costs drop.** Fewer input tokens is the obvious part. The bigger win is iteration count. When an agent has the right guidance every time it touches a file, it makes fewer mistakes and needs fewer fix-up cycles. A wrong assumption in call 5 can mean 30 wasted calls cleaning it up. Tasks that used to stall, with the agent breaking things faster than it fixed them, can start completing.

**Responses get faster.** Fewer input tokens means lower latency. For agents in the hot loop of an edit-test cycle, shaving tokens off every call adds up.

**Accuracy improves.** LLMs degrade when context is long and diluted. Five focused sentences beat fifty scattered paragraphs. The signal-to-noise ratio of your prompt directly affects output quality.
