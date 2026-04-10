import { execSync } from "node:child_process";

export default function (pi) {
  const pending = new Map();

  const mutatingTools = new Set(["edit", "write"]);

  pi.on("tool_call", async (event, ctx) => {
    const text = callSctx("tool_call", event, ctx.cwd, pi);
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
    const after = callSctx("tool_result", event, ctx.cwd, pi);
    const parts = [before, after].filter(Boolean);
    if (parts.length > 0) {
      return {
        content: [...event.content, { type: "text", text: parts.join("\n") }],
      };
    }
  });
}

// Detect planning mode by checking if mutating tools are absent from the
// active tool set. When pi's plan-mode extension is active it restricts
// available tools to read-only ones. This is a heuristic — it depends on
// the plan-mode extension being installed and using getActiveTools().
function isPlanningMode(pi): boolean {
  if (typeof pi.getActiveTools !== "function") return false;
  const active = pi.getActiveTools();
  if (!active || active.length === 0) return false;
  return !active.some((t) => t === "edit" || t === "write");
}

function callSctx(event, toolEvent, cwd, pi) {
  try {
    const includeDecisions = isPlanningMode(pi);
    const payload = JSON.stringify({
      source: "pi",
      event: event,
      tool_name: toolEvent.toolName,
      input: toolEvent.input,
      cwd: cwd,
      include_decisions: includeDecisions,
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
