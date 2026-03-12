import { execSync } from "node:child_process";

export default function (pi) {
  const pending = new Map();

  pi.on("tool_call", async (event, ctx) => {
    const text = callSctx("tool_call", event, ctx.cwd);
    if (text) pending.set(event.toolCallId, text);
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
