---
title: Roadmap
description: What's shipped, what's planned, and what's on the horizon
---

# Roadmap

## v1 (current)

The foundation: file format, resolution engine, Claude Code integration.

- [x] CONTEXT.yaml and AGENTS.yaml file discovery and parsing
- [x] Glob-based file matching with `match`/`exclude`
- [x] Action filtering (`on`: read, edit, create, all)
- [x] Timing filtering (`when`: before, after)
- [x] Directory tree merging (child merges with parent)
- [x] `sctx hook` -- Claude Code adapter
- [x] `sctx context` -- query context entries for a file
- [x] `sctx decisions` -- query decisions for a file
- [x] `sctx validate` -- schema validation
- [x] `sctx init` -- starter file generation
- [x] `alternatives` field on decisions -- record what else was considered and why it was rejected

## v2 (planned)

Richer context management and broader agent support.

- [ ] **`ref` field** -- reference context defined in another file, maintaining a single source of truth
- [ ] **`sctx setup claude`** -- auto-install hooks into Claude Code settings
- [ ] **Session-aware deduplication** -- track what context has already been delivered in a session to avoid repetition
- [ ] **Additional agent adapters** -- Cursor, Windsurf, and others as they expose hook mechanisms
- [ ] **Temporal filtering** -- context that activates after a date or expires before a date, for migration periods and deprecation windows
- [ ] **Extensible filtering** -- framework for community-requested filter dimensions beyond glob, action, timing, and date

## v3 (future)

Advanced features for teams and organizations.

- [ ] **Remote context sources** -- URL-based context that accepts parameters, for centralized guidelines or dynamic context
- [ ] **Context versioning** -- track when entries were last updated, surface stale context
- [ ] **Analytics** -- which context entries are delivered most and least, helping teams identify gaps and noise
- [ ] **Context benchmarking** -- A/B test different context entries to measure which produce better agent outcomes
