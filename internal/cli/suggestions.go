package cli

import (
	"fmt"
	"sort"
	"strings"
)

// levenshteinDistance calculates the edit distance between two strings.
// This is used to find similar command names when the user makes a typo.
func levenshteinDistance(s1, s2 string) int {
	s1 = strings.ToLower(s1)
	s2 = strings.ToLower(s2)

	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create a matrix to store distances
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
	}

	// Initialize first row and column
	for i := 0; i <= len(s1); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	// Fill the matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

func min(values ...int) int {
	if len(values) == 0 {
		return 0
	}
	m := values[0]
	for _, v := range values[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

// similarCommand represents a command suggestion with its distance score.
type similarCommand struct {
	name     string
	distance int
}

// findSimilarCommands finds commands similar to the given input.
// Returns up to maxSuggestions commands with edit distance <= maxDistance.
func findSimilarCommands(input string, candidates []string, maxDistance int, maxSuggestions int) []string {
	if len(candidates) == 0 || input == "" {
		return nil
	}

	// Calculate distance for each candidate
	similar := make([]similarCommand, 0, len(candidates))
	for _, candidate := range candidates {
		distance := levenshteinDistance(input, candidate)
		if distance <= maxDistance && distance > 0 {
			similar = append(similar, similarCommand{
				name:     candidate,
				distance: distance,
			})
		}
	}

	if len(similar) == 0 {
		return nil
	}

	// Sort by distance (closest first), then alphabetically
	sort.Slice(similar, func(i, j int) bool {
		if similar[i].distance != similar[j].distance {
			return similar[i].distance < similar[j].distance
		}
		return similar[i].name < similar[j].name
	})

	// Return up to maxSuggestions
	limit := len(similar)
	if limit > maxSuggestions {
		limit = maxSuggestions
	}

	suggestions := make([]string, limit)
	for i := 0; i < limit; i++ {
		suggestions[i] = similar[i].name
	}

	return suggestions
}

// formatDidYouMean formats a "did you mean" suggestion message.
func formatDidYouMean(input string, suggestions []string) string {
	if len(suggestions) == 0 {
		return ""
	}

	var msg strings.Builder
	if len(suggestions) == 1 {
		msg.WriteString(fmt.Sprintf("\nDid you mean this?\n  %s", suggestions[0]))
	} else {
		msg.WriteString("\nDid you mean one of these?\n")
		for _, suggestion := range suggestions {
			msg.WriteString(fmt.Sprintf("  %s\n", suggestion))
		}
	}

	return msg.String()
}
