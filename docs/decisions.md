---
title: Decisions
description: Record what you rejected, why you rejected it, and when the constraints might change
---

# Decisions

Your codebase shows what you chose. If you picked PostgreSQL, the agent can see `psycopg2` in requirements and SQL migrations on disk. If you went with React, it's right there in `package.json`. The agent doesn't need a YAML file to figure that out.

What the codebase can't show is what you *didn't* choose.

You evaluated DynamoDB and walked away because of single-table design complexity? Looked at MongoDB and rejected it over consistency guarantees? Spent two weeks on GraphQL before deciding the team didn't have the experience to maintain it? None of that is visible in code. It lives in people's heads, old Slack threads, forgotten meeting notes.

Decisions exist to capture the "nos." The rejected alternatives, the constraints behind each rejection, and the conditions under which those constraints might change.

## Why this matters

The most expensive agent mistake isn't writing bad code. It's confidently proposing a migration to something you already evaluated and ruled out. That burns review cycles and generates debate about questions you already settled. With decisions in place, the agent sees that you considered GraphQL, knows *why* you rejected it, and won't suggest it unless the constraints have actually changed.

## Fields

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `decision` | string | yes | -- | What was decided |
| `rationale` | string | yes | -- | Why this was chosen |
| `alternatives` | list | no | -- | Options that were considered and rejected |
| `revisit_when` | string | no | -- | Condition under which this should be reconsidered |
| `date` | date | no | -- | When it was made (YYYY-MM-DD) |
| `match` | list of globs | no | `["**"]` | Scope to specific files |

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

## alternatives

This is where most of the value lives.

Each alternative records a path you evaluated and the specific constraint that killed it:

| Field | Type | Required | Description |
|---|---|---|---|
| `option` | string | yes | The alternative that was considered |
| `reason_rejected` | string | yes | The constraint or tradeoff that ruled it out |

Without alternatives, an agent has no way to know you already spent a week evaluating the thing it's about to suggest. With them, the agent sees the full decision landscape and understands why you landed where you did.

## revisit_when

Every "no" is made under specific constraints. Those constraints change. The library that had bugs ships a stable release. The team that lacked GraphQL experience hires someone who knows it. Your data volume outgrows what a single Postgres instance can handle.

Making the trigger condition explicit turns a static decision into a living one. Agents and humans can check whether the constraint still holds instead of blindly following a decision that may have expired.

## Writing good decisions

A good decision answers four questions: what did you decide, why, what else did you consider, and when should someone revisit it? The third question is the one that matters most.

```yaml
# Weak: the agent already knows you use PostgreSQL from looking at your code.
# This tells it nothing new.
decisions:
  - decision: "Use PostgreSQL"
    rationale: "Good database"

# Strong: now the agent knows what you rejected and why.
# It won't suggest DynamoDB because it can see you already evaluated it.
# And if write throughput becomes a problem, it knows to revisit.
decisions:
  - decision: "PostgreSQL over MySQL or DynamoDB"
    rationale: "JSONB columns for flexible metadata, strong ecosystem for our Python stack, team expertise"
    alternatives:
      - option: "MySQL"
        reason_rejected: "Weaker JSON support, no array types"
      - option: "DynamoDB"
        reason_rejected: "Single-table design is complex, hard to run locally, vendor lock-in"
      - option: "MongoDB"
        reason_rejected: "Weaker consistency guarantees, team prefers SQL"
    revisit_when: "Write throughput exceeds what a single Postgres instance can handle"
    date: 2025-05-15
```

The weak version just restates what the code already shows. The strong version tells the agent what it *can't* see: what was rejected and why.

Each `reason_rejected` should name the specific constraint that killed the option. "Not a good fit" is useless. "Single-table design is complex, hard to run locally, vendor lock-in" gives an agent (or a new team member) the full picture. If any of those constraints change, say you move to AWS and vendor lock-in stops being a concern, the decision can be revisited on its merits instead of blindly followed.

## Dates and staleness

A decision made two years ago under different constraints might be worth questioning. One made last week probably isn't. The `date` field gives that signal.

Combine `date` with `revisit_when` and you get decisions that can expire gracefully. The agent can see both *when* a decision was made and *what would need to change* to reconsider it, instead of depending on someone remembering to check.

## Scoping with match

Like context entries, decisions support glob patterns to scope them to specific files:

```yaml
decisions:
  - decision: "Pydantic for request/response validation"
    rationale: "Native FastAPI integration, JSON Schema generation for free"
    alternatives:
      - option: "marshmallow"
        reason_rejected: "Requires separate schema definitions, no FastAPI integration"
      - option: "attrs + cattrs"
        reason_rejected: "No JSON Schema output, manual validation code"
    match: ["src/api/**"]
```

This decision only shows up when an agent is working in `src/api/`. It won't clutter context for someone editing frontend code.

See [Examples](examples.md) for complete `AGENTS.yaml` files showing decisions alongside context entries in real projects.
