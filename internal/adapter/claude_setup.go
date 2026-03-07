package adapter

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const settingsFile = ".claude/settings.local.json"
const hookCommand = "sctx hook"
const hookMatcher = "Read|Write|Edit|MultiEdit"

var (
	errNoDotClaude = errors.New(".claude/ directory not found — run this from a Claude Code project")
	errInvalidJSON = fmt.Errorf("%s contains invalid JSON", settingsFile)
)

// hookGroup is the structure for a single hook matcher group in Claude Code settings.
type hookGroup struct {
	Matcher string        `json:"matcher"`
	Hooks   []hookHandler `json:"hooks"`
}

// hookHandler is a single hook handler entry.
type hookHandler struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// sctxHookGroup returns the hook group sctx needs registered.
func sctxHookGroup() hookGroup {
	return hookGroup{
		Matcher: hookMatcher,
		Hooks: []hookHandler{
			{Type: "command", Command: hookCommand},
		},
	}
}

// EnableClaude adds sctx hooks to .claude/settings.local.json.
func EnableClaude() error {
	if err := requireDotClaude(); err != nil {
		return err
	}

	settings, err := readSettings()
	if err != nil {
		return err
	}

	if hasSctxHooks(settings) {
		fmt.Printf("sctx hooks are already enabled in %s\n", settingsFile)
		return nil
	}

	addSctxHooks(settings)

	if err := writeSettings(settings); err != nil {
		return err
	}

	fmt.Printf("sctx hooks enabled in %s\n", settingsFile)

	return nil
}

// DisableClaude removes sctx hooks from .claude/settings.local.json.
func DisableClaude() error {
	if err := requireDotClaude(); err != nil {
		return err
	}

	settings, err := readSettings()
	if err != nil {
		return err
	}

	if !hasSctxHooks(settings) {
		fmt.Printf("sctx hooks are not enabled in %s\n", settingsFile)
		return nil
	}

	removeSctxHooks(settings)

	if err := writeSettings(settings); err != nil {
		return err
	}

	fmt.Printf("sctx hooks disabled in %s\n", settingsFile)

	return nil
}

// requireDotClaude checks that the .claude/ directory exists.
func requireDotClaude() error {
	info, err := os.Stat(".claude")
	if err != nil || !info.IsDir() {
		return errNoDotClaude
	}

	return nil
}

// readSettings reads and parses the settings file, returning an empty map if the file doesn't exist.
func readSettings() (map[string]any, error) {
	data, err := os.ReadFile(settingsFile)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]any{}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", settingsFile, err)
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, errInvalidJSON
	}

	return settings, nil
}

// writeSettings marshals and writes the settings map back to disk.
func writeSettings(settings map[string]any) error {
	dir := filepath.Dir(settingsFile)

	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}

	data = append(data, '\n')

	return os.WriteFile(settingsFile, data, 0o600)
}

// hasSctxHooks checks whether any hook group in the settings contains the sctx hook command.
func hasSctxHooks(settings map[string]any) bool {
	hooks, ok := settings["hooks"]
	if !ok {
		return false
	}

	hooksMap, ok := hooks.(map[string]any)
	if !ok {
		return false
	}

	for _, event := range []string{"PreToolUse", "PostToolUse"} {
		if eventHasSctxHook(hooksMap, event) {
			return true
		}
	}

	return false
}

// eventHasSctxHook checks whether a specific event has an sctx hook.
func eventHasSctxHook(hooksMap map[string]any, event string) bool {
	groups, ok := hooksMap[event]
	if !ok {
		return false
	}

	groupList, ok := groups.([]any)
	if !ok {
		return false
	}

	for _, g := range groupList {
		group, ok := g.(map[string]any)
		if !ok {
			continue
		}

		if groupContainsSctxHook(group) {
			return true
		}
	}

	return false
}

// groupContainsSctxHook checks whether a hook group contains the sctx hook command.
func groupContainsSctxHook(group map[string]any) bool {
	handlers, ok := group["hooks"]
	if !ok {
		return false
	}

	handlerList, ok := handlers.([]any)
	if !ok {
		return false
	}

	for _, h := range handlerList {
		handler, ok := h.(map[string]any)
		if !ok {
			continue
		}

		if cmd, ok := handler["command"].(string); ok && cmd == hookCommand {
			return true
		}
	}

	return false
}

// addSctxHooks inserts the sctx hook groups into the settings map.
func addSctxHooks(settings map[string]any) {
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		hooks = map[string]any{}
		settings["hooks"] = hooks
	}

	group := sctxHookGroup()

	// Convert to map[string]any so it serializes consistently with existing entries.
	raw, err := json.Marshal(group)
	if err != nil {
		return
	}

	var groupMap map[string]any
	if err := json.Unmarshal(raw, &groupMap); err != nil {
		return
	}

	for _, event := range []string{"PreToolUse", "PostToolUse"} {
		existing, ok := hooks[event].([]any)
		if !ok {
			existing = []any{}
		}

		hooks[event] = append(existing, groupMap)
	}
}

// removeSctxHooks removes all hook groups containing the sctx hook command.
func removeSctxHooks(settings map[string]any) {
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		return
	}

	for _, event := range []string{"PreToolUse", "PostToolUse"} {
		groups, ok := hooks[event].([]any)
		if !ok {
			continue
		}

		var kept []any

		for _, g := range groups {
			group, ok := g.(map[string]any)
			if !ok {
				kept = append(kept, g)
				continue
			}

			if !groupContainsSctxHook(group) {
				kept = append(kept, g)
			}
		}

		if len(kept) == 0 {
			delete(hooks, event)
		} else {
			hooks[event] = kept
		}
	}

	if len(hooks) == 0 {
		delete(settings, "hooks")
	}
}
