package core

import "testing"

func TestValidAction(t *testing.T) {
	valid := []string{"read", "edit", "create", "all"}
	for _, v := range valid {
		if !ValidAction(v) {
			t.Errorf("ValidAction(%q) = false, want true", v)
		}
	}

	invalid := []string{"banana", "edti", "READ", ""}
	for _, v := range invalid {
		if ValidAction(v) {
			t.Errorf("ValidAction(%q) = true, want false", v)
		}
	}
}

func TestValidTiming(t *testing.T) {
	valid := []string{"before", "after"}
	for _, v := range valid {
		if !ValidTiming(v) {
			t.Errorf("ValidTiming(%q) = false, want true", v)
		}
	}

	invalid := []string{"yesterday", "during", "BEFORE", ""}
	for _, v := range invalid {
		if ValidTiming(v) {
			t.Errorf("ValidTiming(%q) = true, want false", v)
		}
	}
}
