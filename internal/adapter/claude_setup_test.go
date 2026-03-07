package adapter

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestEnableClaude(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T)
		wantErr  string
		wantFile bool
		check    func(t *testing.T, settings map[string]any)
	}{
		{
			name: "no .claude directory",
			setup: func(t *testing.T) {
				t.Helper()
			},
			wantErr: ".claude/ directory not found",
		},
		{
			name: "creates settings file when missing",
			setup: func(t *testing.T) {
				t.Helper()
				mkdirClaude(t)
			},
			wantFile: true,
			check:    checkHooksPresent,
		},
		{
			name: "adds hooks to empty settings",
			setup: func(t *testing.T) {
				t.Helper()
				mkdirClaude(t)
				writeTestSettings(t, map[string]any{})
			},
			wantFile: true,
			check:    checkHooksPresent,
		},
		{
			name: "preserves existing settings",
			setup: func(t *testing.T) {
				t.Helper()
				mkdirClaude(t)
				writeTestSettings(t, map[string]any{
					"model": "claude-opus",
				})
			},
			wantFile: true,
			check: func(t *testing.T, s map[string]any) {
				t.Helper()
				checkHooksPresent(t, s)

				if s["model"] != "claude-opus" {
					t.Errorf("existing setting lost: got model=%v", s["model"])
				}
			},
		},
		{
			name: "preserves existing hooks",
			setup: func(t *testing.T) {
				t.Helper()
				mkdirClaude(t)
				writeTestSettings(t, map[string]any{
					"hooks": map[string]any{
						"PreToolUse": []any{
							map[string]any{
								"matcher": "Bash",
								"hooks": []any{
									map[string]any{"type": "command", "command": "other-tool"},
								},
							},
						},
					},
				})
			},
			wantFile: true,
			check: func(t *testing.T, s map[string]any) {
				t.Helper()
				checkHooksPresent(t, s)

				hooks, ok := s["hooks"].(map[string]any)
				if !ok {
					t.Fatal("missing hooks key")
				}

				pre, ok := hooks["PreToolUse"].([]any)
				if !ok {
					t.Fatal("missing PreToolUse key")
				}

				if len(pre) != 2 {
					t.Errorf("expected 2 PreToolUse groups, got %d", len(pre))
				}
			},
		},
		{
			name: "already enabled",
			setup: func(t *testing.T) {
				t.Helper()
				mkdirClaude(t)
				enabledSettings(t)
			},
			wantFile: true,
			check:    checkHooksPresent,
		},
		{
			name: "invalid JSON",
			setup: func(t *testing.T) {
				t.Helper()
				mkdirClaude(t)

				if err := os.WriteFile(settingsFile, []byte("{bad json"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: "invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			tt.setup(t)

			err := EnableClaude()

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}

				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantFile {
				s := readSettingsFile(t)
				tt.check(t, s)
			}
		})
	}
}

func TestDisableClaude(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T)
		wantErr string
		check   func(t *testing.T, settings map[string]any)
	}{
		{
			name: "no .claude directory",
			setup: func(t *testing.T) {
				t.Helper()
			},
			wantErr: ".claude/ directory not found",
		},
		{
			name: "not enabled",
			setup: func(t *testing.T) {
				t.Helper()
				mkdirClaude(t)
				writeTestSettings(t, map[string]any{})
			},
			check: func(t *testing.T, s map[string]any) {
				t.Helper()

				if _, ok := s["hooks"]; ok {
					t.Error("hooks key should not be present")
				}
			},
		},
		{
			name: "removes sctx hooks",
			setup: func(t *testing.T) {
				t.Helper()
				mkdirClaude(t)
				enabledSettings(t)
			},
			check: func(t *testing.T, s map[string]any) {
				t.Helper()

				if _, ok := s["hooks"]; ok {
					t.Error("hooks key should be removed when empty")
				}
			},
		},
		{
			name: "preserves other hooks",
			setup: func(t *testing.T) {
				t.Helper()
				mkdirClaude(t)
				writeTestSettings(t, map[string]any{
					"hooks": map[string]any{
						"PreToolUse": []any{
							map[string]any{
								"matcher": "Bash",
								"hooks": []any{
									map[string]any{"type": "command", "command": "other-tool"},
								},
							},
							map[string]any{
								"matcher": hookMatcher,
								"hooks": []any{
									map[string]any{"type": "command", "command": hookCommand},
								},
							},
						},
						"PostToolUse": []any{
							map[string]any{
								"matcher": hookMatcher,
								"hooks": []any{
									map[string]any{"type": "command", "command": hookCommand},
								},
							},
						},
					},
				})
			},
			check: func(t *testing.T, s map[string]any) {
				t.Helper()

				hooks, ok := s["hooks"].(map[string]any)
				if !ok {
					t.Fatal("missing hooks key")
				}

				pre, ok := hooks["PreToolUse"].([]any)
				if !ok {
					t.Fatal("missing PreToolUse key")
				}

				if len(pre) != 1 {
					t.Errorf("expected 1 remaining PreToolUse group, got %d", len(pre))
				}

				if _, ok := hooks["PostToolUse"]; ok {
					t.Error("PostToolUse should be removed when empty")
				}
			},
		},
		{
			name: "preserves other settings",
			setup: func(t *testing.T) {
				t.Helper()
				mkdirClaude(t)
				writeTestSettings(t, map[string]any{
					"model": "claude-opus",
					"hooks": map[string]any{
						"PreToolUse": []any{
							map[string]any{
								"matcher": hookMatcher,
								"hooks": []any{
									map[string]any{"type": "command", "command": hookCommand},
								},
							},
						},
						"PostToolUse": []any{
							map[string]any{
								"matcher": hookMatcher,
								"hooks": []any{
									map[string]any{"type": "command", "command": hookCommand},
								},
							},
						},
					},
				})
			},
			check: func(t *testing.T, s map[string]any) {
				t.Helper()

				if s["model"] != "claude-opus" {
					t.Errorf("existing setting lost: got model=%v", s["model"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			tt.setup(t)

			err := DisableClaude()

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}

				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			s := readSettingsFile(t)
			tt.check(t, s)
		})
	}
}

// helpers

func mkdirClaude(t *testing.T) {
	t.Helper()

	if err := os.MkdirAll(".claude", 0o750); err != nil {
		t.Fatal(err)
	}
}

func writeTestSettings(t *testing.T, v any) {
	t.Helper()

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(".claude", 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(settingsFile, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func readSettingsFile(t *testing.T) map[string]any {
	t.Helper()

	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}

	return m
}

func enabledSettings(t *testing.T) {
	t.Helper()

	writeTestSettings(t, map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					"matcher": hookMatcher,
					"hooks": []any{
						map[string]any{"type": "command", "command": hookCommand},
					},
				},
			},
			"PostToolUse": []any{
				map[string]any{
					"matcher": hookMatcher,
					"hooks": []any{
						map[string]any{"type": "command", "command": hookCommand},
					},
				},
			},
		},
	})
}

func checkHooksPresent(t *testing.T, s map[string]any) {
	t.Helper()

	hooks, ok := s["hooks"].(map[string]any)
	if !ok {
		t.Fatal("missing hooks key")
	}

	for _, event := range []string{"PreToolUse", "PostToolUse"} {
		groups, ok := hooks[event].([]any)
		if !ok {
			t.Fatalf("missing %s key", event)
		}

		found := false

		for _, g := range groups {
			group, ok := g.(map[string]any)
			if !ok {
				continue
			}

			if groupContainsSctxHook(group) {
				found = true

				break
			}
		}

		if !found {
			t.Errorf("sctx hook not found in %s", event)
		}
	}
}
