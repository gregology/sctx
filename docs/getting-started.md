# Getting started

## Install

```bash
go install github.com/gregology/sctx/cmd/sctx@latest
```

Make sure `~/go/bin` is in your PATH:

```bash
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
source ~/.zshrc
```

## Create your first context file

From your project root:

```bash
sctx init
```

This creates a `CONTEXT.yaml` with commented examples. Open it and add your first entry:

```yaml
context:
  - content: "Use clear, descriptive variable names. No single-letter names outside of loops."
    on: [edit, create]
    when: before
```

## Test it

Check what context entries apply to a file:

```bash
sctx context src/main.py --on edit --when before
```

Check what decisions apply:

```bash
sctx decisions src/main.py
```

Validate all context files in your project:

```bash
sctx validate
```

## Hook into Claude Code

Add this to `.claude/settings.json` in your project (or `~/.claude/settings.json` for all projects):

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Read|Write|Edit|MultiEdit",
        "hooks": [
          {
            "type": "command",
            "command": "sctx hook"
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Read|Write|Edit|MultiEdit",
        "hooks": [
          {
            "type": "command",
            "command": "sctx hook"
          }
        ]
      }
    ]
  }
}
```

Now when Claude reads or edits a file, `sctx` automatically injects the relevant context. If nothing matches, it's a silent no-op.

## Add more context files

Context files can live anywhere in your project. Add them where the context is most relevant:

```
project/
  CONTEXT.yaml           <- project-wide conventions
  src/
    api/
      CONTEXT.yaml       <- API-specific guidelines
    models/
      CONTEXT.yaml       <- data model conventions
  tests/
    CONTEXT.yaml         <- testing standards
```

Child directories inherit and merge with parent context. You don't need to repeat yourself.

## Next steps

- [Examples](examples.md) for patterns in dbt, React, Terraform, and more
- [Protocol spec](protocol.md) for the full schema reference
- [CLI reference](cli-reference.md) for all commands and flags
