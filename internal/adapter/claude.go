package adapter

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gregology/sctx/internal/core"
)

// ClaudeHookInput represents the JSON that Claude Code sends via stdin to hooks.
type ClaudeHookInput struct {
	SessionID      string          `json:"session_id"`
	HookEventName  string          `json:"hook_event_name"`
	ToolName       string          `json:"tool_name"`
	ToolInput      json.RawMessage `json:"tool_input"`
	CWD            string          `json:"cwd"`
	PermissionMode string          `json:"permission_mode"`
}

// claudeToolInput extracts the file_path from various tool input shapes.
type claudeToolInput struct {
	FilePath string `json:"file_path"`
}

// ClaudeHookOutput is the JSON structure Claude Code expects on stdout.
type ClaudeHookOutput struct {
	HookSpecificOutput *ClaudeHookSpecificOutput `json:"hookSpecificOutput,omitempty"`
}

// ClaudeHookSpecificOutput carries the event name and any context to inject.
type ClaudeHookSpecificOutput struct {
	HookEventName            string `json:"hookEventName"`
	PermissionDecision       string `json:"permissionDecision,omitempty"`
	PermissionDecisionReason string `json:"permissionDecisionReason,omitempty"`
	AdditionalContext        string `json:"additionalContext,omitempty"`
}

const (
	eventPreToolUse         = "PreToolUse"
	permissionDecisionAllow = "allow"
)

// toolToAction maps Claude Code tool names to our universal action type.
var toolToAction = map[string]core.Action{
	"Read":      core.ActionRead,
	"Edit":      core.ActionEdit,
	"MultiEdit": core.ActionEdit,
}

// eventToTiming maps Claude Code hook event names to our universal timing type.
var eventToTiming = map[string]core.Timing{
	"PreToolUse":  core.TimingBefore,
	"PostToolUse": core.TimingAfter,
}

// HandleClaudeHook reads Claude Code's stdin JSON, resolves context, and writes
// the appropriate JSON response to stdout. Returns an error only on fatal failures.
func HandleClaudeHook(input []byte, out, errOut io.Writer) error {
	var hookInput ClaudeHookInput
	if err := json.Unmarshal(input, &hookInput); err != nil {
		return fmt.Errorf("parsing hook input: %w", err)
	}

	var toolInput claudeToolInput
	if err := json.Unmarshal(hookInput.ToolInput, &toolInput); err != nil {
		return fmt.Errorf("parsing tool input: %w", err)
	}

	if toolInput.FilePath == "" {
		return nil
	}

	action := resolveAction(hookInput.ToolName, toolInput.FilePath)

	timing, ok := eventToTiming[hookInput.HookEventName]
	if !ok {
		return nil
	}

	result, warnings, err := core.Resolve(core.ResolveRequest{
		FilePath: toolInput.FilePath,
		Action:   action,
		Timing:   timing,
		Root:     hookInput.CWD,
	})
	if err != nil {
		return fmt.Errorf("resolving context: %w", err)
	}

	for _, w := range warnings {
		_, _ = fmt.Fprintln(errOut, w) // best-effort; write failures non-fatal
	}

	inPlanMode := hookInput.PermissionMode == "plan"
	hasContext := len(result.ContextEntries) > 0
	hasDecisions := inPlanMode && len(result.DecisionEntries) > 0

	if !hasContext && !hasDecisions {
		return nil
	}

	var additionalContext string
	if hasContext {
		additionalContext = formatContext(result.ContextEntries)
	}
	if hasDecisions {
		additionalContext += formatDecisions(result.DecisionEntries)
	}

	hookOutput := &ClaudeHookSpecificOutput{
		HookEventName:     hookInput.HookEventName,
		AdditionalContext: additionalContext,
	}

	if hookInput.HookEventName == eventPreToolUse {
		hookOutput.PermissionDecision = permissionDecisionAllow
		hookOutput.PermissionDecisionReason = "sctx: structured context injected"
	}

	output := ClaudeHookOutput{
		HookSpecificOutput: hookOutput,
	}

	return json.NewEncoder(out).Encode(output)
}

// resolveAction determines the action type from the tool name.
// For Write, it checks whether the file already exists to distinguish create vs edit.
func resolveAction(toolName, filePath string) core.Action {
	if mapped, ok := toolToAction[toolName]; ok {
		return mapped
	}

	if toolName == "Write" {
		if _, err := os.Stat(filePath); err != nil {
			return core.ActionCreate
		}

		return core.ActionEdit
	}

	return core.ActionAll
}

// formatContext builds a markdown string from matched context entries.
func formatContext(entries []core.MatchedContext) string {
	var b strings.Builder

	b.WriteString("## Structured Context\n")

	for _, entry := range entries {
		b.WriteString("\n- ")
		b.WriteString(entry.Content)
	}

	return b.String()
}

// formatDecisions builds a markdown string from matched decision entries.
func formatDecisions(entries []core.DecisionEntry) string {
	var b strings.Builder

	b.WriteString("\n## Architectural Decisions\n")

	for _, e := range entries {
		b.WriteString("\n- ")
		b.WriteString(e.Decision)
		b.WriteString("\n  Rationale: ")
		b.WriteString(e.Rationale)

		for _, alt := range e.Alternatives {
			fmt.Fprintf(&b, "\n  Considered %s, rejected: %s", alt.Option, alt.ReasonRejected)
		}

		if e.RevisitWhen != "" {
			b.WriteString("\n  Revisit when: ")
			b.WriteString(e.RevisitWhen)
		}
	}

	return b.String()
}
