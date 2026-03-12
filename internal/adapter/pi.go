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

// piBashInput extracts the command from bash tool input.
type piBashInput struct {
	Command string `json:"command"`
}

// piToolToAction maps pi tool names to our universal action type.
var piToolToAction = map[string]core.Action{
	"read": core.ActionRead,
	"edit": core.ActionEdit,
	"bash": core.ActionRead,
}

// bashReadCmds maps commands that read files to their flags that consume the next argument.
// For example, head -n 20 file.go: the -n flag takes "20" as its value, not as a file path.
var bashReadCmds = map[string]map[string]bool{
	"cat":  {},
	"head": {"-n": true, "-c": true},
	"tail": {"-n": true, "-c": true},
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

	if toolInput.Path == "" && hookInput.ToolName == "bash" {
		var bashIn piBashInput
		if err := json.Unmarshal(hookInput.Input, &bashIn); err == nil {
			toolInput.Path = bashReadPath(bashIn.Command)
		}
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

// bashReadPath extracts a file path from simple cat, head, or tail commands.
// It returns "" if the command is not a recognized read pattern.
func bashReadPath(command string) string {
	// Isolate the first pipe segment: "cat go.mod | grep foo" → "cat go.mod"
	if i := strings.IndexByte(command, '|'); i >= 0 {
		command = command[:i]
	}

	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}

	valuedFlags, ok := bashReadCmds[fields[0]]
	if !ok {
		return ""
	}

	// Skip flags and their values to find the file path.
	skip := false

	for _, f := range fields[1:] {
		if skip {
			skip = false

			continue
		}

		if strings.HasPrefix(f, "-") {
			if valuedFlags[f] {
				skip = true
			}

			continue
		}

		return f
	}

	return ""
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
