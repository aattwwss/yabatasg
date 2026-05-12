package auth

import (
	"strings"
	"testing"
)

func TestNewPhrase(t *testing.T) {
	seen := make(map[string]bool)
	for range 50 {
		p, err := NewPhrase()
		if err != nil {
			t.Fatalf("NewPhrase failed: %v", err)
		}
		parts := strings.Split(p, "-")
		if len(parts) != 4 {
			t.Fatalf("expected 4 words, got %d: %q", len(parts), p)
		}
		for _, w := range parts {
			if !inWordlist(w) {
				t.Fatalf("word %q not in EFF wordlist", w)
			}
		}
		seen[p] = true
	}
	if len(seen) < 45 {
		t.Errorf("low diversity: %d unique phrases out of 50", len(seen))
	}
}

func TestEffShortLength(t *testing.T) {
	if len(effShort) < 1000 {
		t.Errorf("effShort has %d words, expected at least 1000", len(effShort))
	}
}

func inWordlist(w string) bool {
	for _, e := range effShort {
		if e == w {
			return true
		}
	}
	return false
}
