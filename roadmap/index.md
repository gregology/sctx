# Roadmap

## v1 (current)

The foundation: file format, resolution engine, Claude Code integration.

- AGENTS.yaml file discovery and parsing
- Glob-based file matching with `match`/`exclude`
- Action filtering (`on`: read, edit, create, all)
- Timing filtering (`when`: before, after)
- Directory tree merging (child merges with parent)
- `sctx hook` -- Claude Code adapter
- `sctx context` -- query context entries for a file
- `sctx decisions` -- query decisions for a file
- `sctx validate` -- schema validation
- `sctx init` -- starter file generation
- `alternatives` field on decisions -- record what else was considered and why it was rejected
- `sctx claude enable/disable` -- install/remove hooks in Claude Code settings

## v2 (planned)

Richer context management and broader agent support.

- **`ref` field** -- reference context defined in another file, maintaining a single source of truth
- **Session-aware deduplication** -- track what context has already been delivered in a session to avoid repetition
- **Additional agent adapters** -- Cursor, Windsurf, and others as they expose hook mechanisms
- **Temporal filtering** -- context that activates after a date or expires before a date, for migration periods and deprecation windows
- **Extensible filtering** -- framework for community-requested filter dimensions beyond glob, action, timing, and date

## v3 (future)

Advanced features for teams and organizations.

- **Remote context sources** -- URL-based context that accepts parameters, for centralized guidelines or dynamic context
- **Context versioning** -- track when entries were last updated, surface stale context
- **Analytics** -- which context entries are delivered most and least, helping teams identify gaps and noise
- **Context benchmarking** -- A/B test different context entries to measure which produce better agent outcomes
