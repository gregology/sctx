package core

import "testing"

func TestValidAction(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"read", true},
		{"edit", true},
		{"create", true},
		{"all", true},
		{"banana", false},
		{"", false},
		{"READ", false},
		{"delete", false},
	}

	for _, tt := range tests {
		if got := ValidAction(tt.input); got != tt.want {
			t.Errorf("ValidAction(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestValidTiming(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"before", true},
		{"after", true},
		{"yesterday", false},
		{"", false},
		{"BEFORE", false},
		{"during", false},
	}

	for _, tt := range tests {
		if got := ValidTiming(tt.input); got != tt.want {
			t.Errorf("ValidTiming(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
