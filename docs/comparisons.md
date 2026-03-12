---
title: How does sctx compare?
description: How Structured Context differs from AGENTS.md, MCP, and IDE-specific rules
---

# How does sctx compare?

The short version: `sctx` provides **file-targeted, action-filtered context injection**. Instead of loading all instructions all the time, it delivers only the entries that match what the agent is doing right now.

| Tool | Scope | Format | Delivery |
|------|-------|--------|----------|
| AGENTS.md | Directory | Unstructured prose | Always loaded |
| MCP | External tools & data | RPC protocol | On demand via server |
| .cursorrules | Project root | Monolithic prompt | Always loaded |
| **sctx** | **Per-file, per-action** | **Declarative YAML** | **JIT, glob-matched** |

## AGENTS.md

[AGENTS.md](https://agents.md/) is becoming the standard project-level manifest for AI coding agents, providing a dedicated place for build commands and coding conventions.

**The distinction:** `AGENTS.md` is directory-scoped and largely unstructured prose. Developers must write natural language conditional logic ("If you are editing a SQL file, do X"). As the file grows, agents struggle to parse it — attention dilutes and the model quietly ignores the instruction that mattered most.

Structured Context improves on this with declarative YAML glob-matching (`**/*.sql`), ensuring the LLM only ever reads the context applicable to the file it is actively touching.

## Model Context Protocol (MCP)

Anthropic's [MCP](https://modelcontextprotocol.io/) is an open-source client-server protocol that standardizes how AI systems integrate with external tools and data sources.

**The distinction:** MCP is an **active RPC protocol** (like a USB-C cable for AI tools), whereas `sctx` is a **static declarative file format**. MCP connects the agent to the environment, while `sctx` dictates the *rules of engagement* for the codebase. They're complementary — an MCP server could be built to dynamically serve `sctx` contexts to an agent.

## IDE-specific rules (.cursorrules / .windsurfrules)

Project-root or directory-level markdown files where developers drop system prompts and stylistic preferences for AI IDEs like Cursor or Windsurf.

**The distinction:** These rules are generally monolithic. If a `.cursorrules` file contains React, Python, and SQL guidelines, the AI is burdened with all of them simultaneously during *any* edit. `sctx`'s action-filtering (`on: read` vs `on: edit` vs `on: create`) and precise file-path scoping offer a level of granularity that these files lack.

### Example: monolithic rules vs. targeted context

A typical `.cursorrules` file loads everything at once:

```text
# .cursorrules

When editing Python files, use snake_case for all identifiers.
When editing SQL models, use the incremental strategy macro.
When editing React components, prefer named exports.
When creating any file, add a license header.
```

The agent sees all four instructions regardless of which file it touches. With Structured Context, each instruction only appears when it's relevant:

```yaml
# AGENTS.yaml

context:
  - content: "Use snake_case for all identifiers"
    match: ["**/*.py"]
    on: edit

  - content: "Use the incremental strategy macro"
    match: ["models/**/*.sql"]
    on: edit

  - content: "Prefer named exports"
    match: ["src/components/**/*.tsx"]
    on: edit

  - content: "Add a license header"
    on: create
```

When the agent edits `models/revenue.sql`, it sees one instruction instead of four. At scale — dozens of conventions across a monorepo — the difference in signal-to-noise ratio is significant.

