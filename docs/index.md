---
title: Structured Context
description: Scoped, structured context for AI agents
---

# Structured Context

Drop `CONTEXT.yaml` files into your codebase and AI agents get the right guidance at the right time.

## The problem

AI coding agents read instruction files like `AGENTS.md` to understand how to work in a codebase. These files are blunt instruments. They're scoped to a directory, written as unstructured paragraphs and code snippets, and every instruction gets loaded regardless of what the agent is actually doing.

This approach bloats context.

Take a dbt project. A single `models/` directory contains SQL model files, YAML schema files, and markdown docs. The context an agent needs for each of these is completely different:

- When editing a **SQL model**, the agent should know about your SQL style, deprecated functions, available macros, and incremental strategy.
- When editing a **YAML schema**, it should know your testing conventions, required fields, and how you structure column descriptions.
- When editing a **markdown doc**, it should know your documentation templates and style.

With `AGENTS.md`, your only option is one big file where every paragraph starts with "if you're editing SQL do X, if you're editing YAML do Y." That's conditional logic written as prose. The agent has to parse natural language to figure out which instructions apply. Wasteful and error-prone.

This pattern shows up everywhere. API directories with route handlers, middleware, tests, and OpenAPI specs. React component directories with `.tsx`, `.test.tsx`, `.stories.tsx`, and `.module.css` files. Infrastructure directories with Terraform `.tf` files, `.tfvars`, and documentation. Monorepo packages where shared conventions apply globally but each package has its own patterns.

The underlying issue: files in the same directory often have very different context requirements. Directory-scoped unstructured text can't express this.

## The approach

Structured Context defines a YAML-based file format (`CONTEXT.yaml`) that lets you:

- **Scope context to specific files** using glob patterns (`match: ["**/*.sql"]`)
- **Filter by action** -- different guidance for reading vs editing vs creating files
- **Control prompt positioning** -- place context before or after file content based on importance
- **Capture decisions** -- record what you chose, why, what else you considered, and when to revisit

Context files are placed throughout your codebase and merge with parent directories, so you get inheritance without duplication.

The protocol is agent-agnostic. The reference implementation (`sctx`) ships with a Claude Code adapter, but the core engine has no knowledge of any specific agent. Other tools can adopt the file format directly.

## Why this matters beyond better output

The obvious benefit is better output. Agents follow your conventions instead of guessing or being saturated with irrelevant context to the specific task. Precise context has compounding effects.

**Smaller models become viable.** A large model can sometimes infer the right pattern from a sprawling AGENTS.md. A smaller model may struggle. But a small model with exactly the right context can perform well. Structured Context closes the gap between model tiers by compensating with better input.

**Costs drop.** There's the obvious part: less context per call means fewer input tokens. But the bigger win is iteration count. When an agent has the right guidance every time it touches a file, it makes fewer mistakes and needs fewer fix-up cycles. A wrong assumption in call 5 can mean 30 wasted calls cleaning it up.

**Responses get faster.** Fewer input tokens means lower latency. For agents in the hot loop of an edit-test cycle, shaving tokens off every call adds up.

**Accuracy improves.** LLMs degrade when context is long and diluted. Giving the model 5 focused sentences instead of 50 scattered paragraphs means it's more likely to actually follow the instructions. The signal-to-noise ratio of your prompt directly affects output quality.

## Quick start

```bash
brew install gregology/tap/sctx
sctx init
```

Then [hook it into Claude Code](getting-started.md#hook-into-claude-code) and you're done.
