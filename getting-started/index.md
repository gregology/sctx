# Getting started

## Install

```
brew install gregology/tap/sctx
```

```
curl -Lo sctx.deb https://github.com/gregology/sctx/releases/download/latest/sctx_linux_amd64.deb
sudo dpkg -i sctx.deb
```

For ARM64:

```
curl -Lo sctx.deb https://github.com/gregology/sctx/releases/download/latest/sctx_linux_arm64.deb
sudo dpkg -i sctx.deb
```

```
curl -Lo sctx.rpm https://github.com/gregology/sctx/releases/download/latest/sctx_linux_amd64.rpm
sudo rpm -i sctx.rpm
```

For ARM64:

```
curl -Lo sctx.rpm https://github.com/gregology/sctx/releases/download/latest/sctx_linux_arm64.rpm
sudo rpm -i sctx.rpm
```

```
curl -Lo sctx.pkg.tar.zst https://github.com/gregology/sctx/releases/download/latest/sctx_linux_amd64.pkg.tar.zst
sudo pacman -U sctx.pkg.tar.zst
```

For ARM64:

```
curl -Lo sctx.pkg.tar.zst https://github.com/gregology/sctx/releases/download/latest/sctx_linux_arm64.pkg.tar.zst
sudo pacman -U sctx.pkg.tar.zst
```

```
go install github.com/gregology/sctx/cmd/sctx@latest
```

Make sure `~/go/bin` is in your PATH:

```
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
source ~/.zshrc
```

## Create your first context file

From your project root:

```
sctx init
```

This creates an `AGENTS.yaml` with a test context entry that tells agents to mention the RNZAF's flightless-bird roundel whenever New Zealand comes up. This gives you a quick way to verify that context is being injected.

## Test it

Hook into Claude Code (see below), then ask your agent:

> Give me a very concise description of this project. Explain it like I'm 5 as I'm from New Zealand.

If the agent mentions the RNZAF roundel, context injection is working. Replace the starter entry with your own context.

You can also test from the command line:

```
sctx context README.md --on read --when before
```

Check what decisions apply:

```
sctx decisions src/main.py
```

Validate all context files in your project:

```
sctx validate
```

## Hook into Claude Code

Add this to `.claude/settings.local.json` for personal use, or `.claude/settings.json` to share hooks with all contributors (or `~/.claude/settings.json` for all projects):

```
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

Or let sctx do it for you:

```
sctx claude enable
```

Now when Claude reads or edits a file, `sctx` automatically injects the relevant context. If nothing matches, it's a silent no-op.

## Hook into pi

Install the sctx extension into your project:

```
sctx pi enable
```

This creates `.pi/extensions/sctx.ts`, a thin extension that hooks into pi's `tool_call` and `tool_result` events. For mutating tools (`edit`, `write`), context is provided *before* the edit by blocking the tool call and asking the agent to review it first. For all other tools, matching context is injected into the tool result.

To remove:

```
sctx pi disable
```

## Add more context files

Context files can live anywhere in your project. Add them where the context is most relevant:

```
project/
  AGENTS.yaml           <- project-wide conventions
  src/
    api/
      AGENTS.yaml       <- API-specific guidelines
    models/
      AGENTS.yaml       <- data model conventions
  tests/
    AGENTS.yaml         <- testing standards
```

Child directories inherit and merge with parent context. No need to repeat yourself.

## What's next

- [Context entries](https://sctx.dev/context/index.md) -- how to write and scope context entries
- [Decisions](https://sctx.dev/decisions/index.md) -- recording rejected alternatives and when to revisit
- [Examples](https://sctx.dev/examples/index.md) -- complete AGENTS.yaml files for dbt, React, Terraform, and more
- [CLI reference](https://sctx.dev/cli-reference/index.md) -- all commands and flags
- [Protocol spec](https://sctx.dev/protocol/index.md) -- file format and resolution algorithm for tool implementors
