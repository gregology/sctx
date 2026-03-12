package adapter

import (
	"os"
	"strings"
	"testing"
)

func TestEnablePi(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T)
		wantErr  string
		wantFile bool
	}{
		{
			name: "no .pi directory",
			setup: func(t *testing.T) {
				t.Helper()
			},
			wantErr: ".pi/ directory not found",
		},
		{
			name: "creates extension file",
			setup: func(t *testing.T) {
				t.Helper()
				mkdirPi(t)
			},
			wantFile: true,
		},
		{
			name: "creates extensions subdirectory",
			setup: func(t *testing.T) {
				t.Helper()
				mkdirPi(t)
			},
			wantFile: true,
		},
		{
			name: "already enabled",
			setup: func(t *testing.T) {
				t.Helper()
				mkdirPi(t)
				writePiExtension(t)
			},
			wantFile: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			tt.setup(t)

			err := EnablePi()

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
				checkPiExtensionFile(t)
			}
		})
	}
}

func TestDisablePi(t *testing.T) {
	t.Run("no .pi directory", func(t *testing.T) {
		t.Chdir(t.TempDir())

		err := DisablePi()
		if err == nil || !strings.Contains(err.Error(), ".pi/ directory not found") {
			t.Fatalf("expected .pi/ not found error, got: %v", err)
		}
	})

	t.Run("not enabled", func(t *testing.T) {
		t.Chdir(t.TempDir())
		mkdirPi(t)

		if err := DisablePi(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, err := os.Stat(piExtensionFile); err == nil {
			t.Error("extension file should not exist")
		}
	})

	t.Run("removes extension file", func(t *testing.T) {
		t.Chdir(t.TempDir())
		mkdirPi(t)
		writePiExtension(t)

		if err := DisablePi(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, err := os.Stat(piExtensionFile); err == nil {
			t.Error("extension file should be removed")
		}

		if _, err := os.Stat(piExtensionDir); err == nil {
			t.Error("empty extensions directory should be removed")
		}
	})

	t.Run("preserves other extensions", func(t *testing.T) {
		t.Chdir(t.TempDir())
		mkdirPi(t)
		writePiExtension(t)

		if err := os.WriteFile(".pi/extensions/other.ts", []byte("// other"), 0o600); err != nil {
			t.Fatal(err)
		}

		if err := DisablePi(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, err := os.Stat(piExtensionFile); err == nil {
			t.Error("sctx extension file should be removed")
		}

		if _, err := os.Stat(".pi/extensions/other.ts"); err != nil {
			t.Error("other extension should be preserved")
		}

		if _, err := os.Stat(piExtensionDir); err != nil {
			t.Error("extensions directory should be preserved when not empty")
		}
	})
}

// helpers

func mkdirPi(t *testing.T) {
	t.Helper()

	if err := os.MkdirAll(".pi", 0o750); err != nil {
		t.Fatal(err)
	}
}

func writePiExtension(t *testing.T) {
	t.Helper()

	if err := os.MkdirAll(piExtensionDir, 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(piExtensionFile, []byte(piExtensionSource), 0o600); err != nil {
		t.Fatal(err)
	}
}

func checkPiExtensionFile(t *testing.T) {
	t.Helper()

	data, err := os.ReadFile(piExtensionFile)
	if err != nil {
		t.Fatalf("expected extension file to exist: %v", err)
	}

	if !strings.Contains(string(data), "sctx hook") {
		t.Error("extension file should contain sctx hook command")
	}

	if !strings.Contains(string(data), `source: "pi"`) {
		t.Error("extension file should set source to pi")
	}
}
