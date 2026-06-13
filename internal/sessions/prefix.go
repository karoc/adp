package sessions

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/karoc/adp/internal/paths"
)

var (
	ErrSessionNotFound    = errors.New("session not found")
	ErrAmbiguousSessionID = errors.New("ambiguous session ID prefix")
)

// FindByPrefix finds sessions matching the given prefix.
// It implements the following matching logic:
// 1. Exact match has highest priority (returns single match)
// 2. Prefix match returns all matches
// 3. Returns ErrSessionNotFound if no matches
// 4. Returns ErrAmbiguousSessionID if multiple prefix matches
func FindByPrefix(ctx context.Context, layout paths.Layout, prefix string) ([]Summary, error) {
	if prefix == "" {
		return nil, ErrSessionNotFound
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Get all sessions
	allSessions, err := List(ctx, layout, Query{})
	if err != nil {
		return nil, err
	}

	// Try exact match first
	for _, session := range allSessions {
		if session.SessionID == prefix {
			return []Summary{session}, nil
		}
	}

	// Try prefix match
	var matches []Summary
	for _, session := range allSessions {
		if strings.HasPrefix(session.SessionID, prefix) {
			matches = append(matches, session)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("%w: no session found with prefix %q", ErrSessionNotFound, prefix)
	}
	if len(matches) > 1 {
		return matches, fmt.Errorf("%w: prefix %q matches %d sessions", ErrAmbiguousSessionID, prefix, len(matches))
	}

	return matches, nil
}
