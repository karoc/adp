package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

func (a *App) version(ctx context.Context, args []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(args) != 0 {
		return errors.New("usage: adp version")
	}
	fmt.Fprint(a.stdout, versionString())
	return nil
}

func versionString() string {
	parts := []string{"adp", nonEmpty(Version, "dev")}
	if strings.TrimSpace(Commit) != "" {
		parts = append(parts, "commit", strings.TrimSpace(Commit))
	}
	if strings.TrimSpace(BuildDate) != "" {
		parts = append(parts, "built", strings.TrimSpace(BuildDate))
	}
	return strings.Join(parts, " ") + "\n"
}

func nonEmpty(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
