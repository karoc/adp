package cli

import (
	"reflect"
	"testing"
)

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		s1       string
		s2       string
		expected int
	}{
		{"", "", 0},
		{"", "abc", 3},
		{"abc", "", 3},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"abc", "adc", 1},
		{"workspace", "workspac", 1},
		{"workspace", "wrkspc", 3},
		{"tasks", "task", 1},
		{"sessions", "session", 1},
		{"remove", "remov", 1},
		{"runtime", "runtim", 1},
		{"WORKSPACE", "workspace", 0}, // case insensitive
		{"Task", "task", 0},            // case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.s1+"_"+tt.s2, func(t *testing.T) {
			distance := levenshteinDistance(tt.s1, tt.s2)
			if distance != tt.expected {
				t.Errorf("levenshteinDistance(%q, %q) = %d, want %d", tt.s1, tt.s2, distance, tt.expected)
			}
		})
	}
}

func TestFindSimilarCommands(t *testing.T) {
	candidates := []string{"workspace", "tasks", "sessions", "events", "runtime", "quickstart"}

	tests := []struct {
		name            string
		input           string
		maxDistance     int
		maxSuggestions  int
		expectedMatches []string
	}{
		{
			name:            "exact match excluded",
			input:           "workspace",
			maxDistance:     2,
			maxSuggestions:  3,
			expectedMatches: nil,
		},
		{
			name:            "one character typo",
			input:           "workspac",
			maxDistance:     2,
			maxSuggestions:  3,
			expectedMatches: []string{"workspace"},
		},
		{
			name:            "missing vowels",
			input:           "wrkspc",
			maxDistance:     3,
			maxSuggestions:  3,
			expectedMatches: []string{"workspace"},
		},
		{
			name:            "plural to singular",
			input:           "task",
			maxDistance:     2,
			maxSuggestions:  3,
			expectedMatches: []string{"tasks"},
		},
		{
			name:            "multiple matches sorted by distance",
			input:           "session",
			maxDistance:     2,
			maxSuggestions:  3,
			expectedMatches: []string{"sessions"},
		},
		{
			name:            "max suggestions limit",
			input:           "workspac",
			maxDistance:     2,
			maxSuggestions:  1,
			expectedMatches: []string{"workspace"},
		},
		{
			name:            "empty input",
			input:           "",
			maxDistance:     2,
			maxSuggestions:  3,
			expectedMatches: nil,
		},
		{
			name:            "no matches within distance",
			input:           "xyz",
			maxDistance:     1,
			maxSuggestions:  3,
			expectedMatches: nil,
		},
		{
			name:            "case insensitive",
			input:           "WORKSPAC",
			maxDistance:     2,
			maxSuggestions:  3,
			expectedMatches: []string{"workspace"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := findSimilarCommands(tt.input, candidates, tt.maxDistance, tt.maxSuggestions)
			if !reflect.DeepEqual(suggestions, tt.expectedMatches) {
				t.Errorf("findSimilarCommands(%q) = %v, want %v", tt.input, suggestions, tt.expectedMatches)
			}
		})
	}
}

func TestFindSimilarCommandsEmptyCandidates(t *testing.T) {
	suggestions := findSimilarCommands("workspace", []string{}, 2, 3)
	if suggestions != nil {
		t.Errorf("findSimilarCommands with empty candidates should return nil, got %v", suggestions)
	}
}

func TestFormatDidYouMean(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		suggestions []string
		contains    []string
	}{
		{
			name:        "single suggestion",
			input:       "workspac",
			suggestions: []string{"workspace"},
			contains:    []string{"Did you mean this?", "workspace"},
		},
		{
			name:        "multiple suggestions",
			input:       "wrk",
			suggestions: []string{"workspace", "quickstart"},
			contains:    []string{"Did you mean one of these?", "workspace", "quickstart"},
		},
		{
			name:        "no suggestions",
			input:       "xyz",
			suggestions: []string{},
			contains:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDidYouMean(tt.input, tt.suggestions)
			if len(tt.suggestions) == 0 {
				if result != "" {
					t.Errorf("formatDidYouMean with no suggestions should return empty string, got %q", result)
				}
				return
			}

			for _, expected := range tt.contains {
				if !contains(result, expected) {
					t.Errorf("formatDidYouMean result should contain %q, got:\n%s", expected, result)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
