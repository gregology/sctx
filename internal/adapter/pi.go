package adapter

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gregology/sctx/internal/core"
)

// PiHookInput represents the JSON that the pi extension sends via stdin to sctx hook.
type PiHookInput struct {
	Source   string          `json:"source"`
	Event    string          `json:"event"`
	ToolName string          `json:"tool_name"`
	Input    json.RawMessage `json:"input"`
	CWD      string          `json:"cwd"`
}

// piToolInput extracts the path from pi tool input shapes.
type piToolInput struct {
	Path string `json:"path"`
}

// PiHookOutput is the JSON structure the pi extension expects on stdout.
type PiHookOutput struct {
	AdditionalContext string `json:"additionalContext,omitempty"`
}

// piToolToAction maps pi tool names to our universal action type.
var piToolToAction = map[string]core.Action{
	"read": core.ActionRead,
	"edit": core.ActionEdit,
}

// piEventToTiming maps pi hook event names to our universal timing type.
var piEventToTiming = map[string]core.Timing{
	"tool_call":   core.TimingBefore,
	"tool_result": core.TimingAfter,
}

// HandlePiHook reads pi's stdin JSON, resolves context, and writes
// the appropriate JSON response to stdout. Returns an error only on fatal failures.
func HandlePiHook(input []byte) error {
	var hookInput PiHookInput
	if err := json.Unmarshal(input, &hookInput); err != nil {
		return fmt.Errorf("parsing hook input: %w", err)
	}

	var toolInput piToolInput
	if err := json.Unmarshal(hookInput.Input, &toolInput); err != nil {
		return fmt.Errorf("parsing tool input: %w", err)
	}

	if toolInput.Path == "" {
		return nil
	}

	action := resolvePiAction(hookInput.ToolName, toolInput.Path)

	timing, ok := piEventToTiming[hookInput.Event]
	if !ok {
		return nil
	}

	result, warnings, err := core.Resolve(core.ResolveRequest{
		FilePath: toolInput.Path,
		Action:   action,
		Timing:   timing,
		Root:     hookInput.CWD,
	})
	if err != nil {
		return fmt.Errorf("resolving context: %w", err)
	}

	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, w)
	}

	if len(result.ContextEntries) == 0 {
		return nil
	}

	output := PiHookOutput{
		AdditionalContext: formatContext(result.ContextEntries),
	}

	return json.NewEncoder(os.Stdout).Encode(output)
}

// resolvePiAction determines the action type from the pi tool name.
// For write, it checks whether the file already exists to distinguish create vs edit.
func resolvePiAction(toolName, filePath string) core.Action {
	if mapped, ok := piToolToAction[toolName]; ok {
		return mapped
	}

	if toolName == "write" {
		if _, err := os.Stat(filePath); err != nil {
			return core.ActionCreate
		}

		return core.ActionEdit
	}

	return core.ActionAll
}

// IsPiHook checks whether the raw JSON input is from pi (has "source": "pi").
func IsPiHook(input []byte) bool {
	var probe struct {
		Source string `json:"source"`
	}

	if err := json.Unmarshal(input, &probe); err != nil {
		return false
	}

	return strings.EqualFold(probe.Source, "pi")
}
