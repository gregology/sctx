package adapter

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const piExtensionDir = ".pi/extensions"
const piExtensionFile = ".pi/extensions/sctx.ts"

var errNoDotPi = errors.New(".pi/ directory not found — run this from a pi project")

const piExtensionSource = `import { execSync } from "node:child_process";

export default function (pi) {
  const pending = new Map();

  const mutatingTools = new Set(["edit", "write"]);

  pi.on("tool_call", async (event, ctx) => {
    const text = callSctx("tool_call", event, ctx.cwd);
    if (!text) return;

    if (mutatingTools.has(event.toolName)) {
      return {
        block: true,
        reason:
          "Review the following context before proceeding, then re-apply your change.\n\n" +
          text,
      };
    }

    pending.set(event.toolCallId, text);
  });

  pi.on("tool_result", async (event, ctx) => {
    const before = pending.get(event.toolCallId);
    pending.delete(event.toolCallId);
    const after = callSctx("tool_result", event, ctx.cwd);
    const parts = [before, after].filter(Boolean);
    if (parts.length > 0) {
      return {
        content: [...event.content, { type: "text", text: parts.join("\n") }],
      };
    }
  });
}

function callSctx(event, toolEvent, cwd) {
  try {
    const payload = JSON.stringify({
      source: "pi",
      event: event,
      tool_name: toolEvent.toolName,
      input: toolEvent.input,
      cwd: cwd,
    });
    const result = execSync("sctx hook", {
      input: payload,
      encoding: "utf-8",
      timeout: 5000,
    });
    if (!result.trim()) return null;
    const parsed = JSON.parse(result);
    return parsed.additionalContext || null;
  } catch {
    return null;
  }
}
`

// EnablePi creates the pi extension file at .pi/extensions/sctx.ts.
func EnablePi() error {
	if err := requireDotPi(); err != nil {
		return err
	}

	if _, err := os.Stat(piExtensionFile); err == nil {
		fmt.Printf("sctx extension already exists at %s\n", piExtensionFile)
		return nil
	}

	if err := os.MkdirAll(piExtensionDir, 0o750); err != nil {
		return fmt.Errorf("creating %s: %w", piExtensionDir, err)
	}

	if err := os.WriteFile(piExtensionFile, []byte(piExtensionSource), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", piExtensionFile, err)
	}

	fmt.Printf("sctx extension enabled at %s\n", piExtensionFile)

	return nil
}

// DisablePi removes the pi extension file at .pi/extensions/sctx.ts.
func DisablePi() error {
	if err := requireDotPi(); err != nil {
		return err
	}

	if _, err := os.Stat(piExtensionFile); errors.Is(err, os.ErrNotExist) {
		fmt.Printf("sctx extension is not enabled in %s\n", piExtensionFile)
		return nil
	}

	if err := os.Remove(piExtensionFile); err != nil {
		return fmt.Errorf("removing %s: %w", piExtensionFile, err)
	}

	// Clean up empty extensions directory.
	entries, err := os.ReadDir(piExtensionDir)
	if err == nil && len(entries) == 0 {
		_ = os.Remove(piExtensionDir)
	}

	fmt.Printf("sctx extension disabled — removed %s\n", piExtensionFile)

	return nil
}

// requireDotPi checks that the .pi/ directory exists.
func requireDotPi() error {
	info, err := os.Stat(filepath.Dir(piExtensionDir))
	if err != nil || !info.IsDir() {
		return errNoDotPi
	}

	return nil
}
