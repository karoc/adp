package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
)

type versionOutputOptions struct {
	format outputFormat
}

func (a *App) version(ctx context.Context, args []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	opts, err := parseVersionArgs(args)
	if err != nil {
		return err
	}

	if opts.format == outputFormatJSON {
		return a.versionJSON()
	}

	fmt.Fprint(a.stdout, versionString())
	return nil
}

func parseVersionArgs(args []string) (versionOutputOptions, error) {
	opts := versionOutputOptions{format: outputFormatText}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--format":
			value, next, err := requireValue(args, i, arg)
			if err != nil {
				return versionOutputOptions{}, err
			}
			format, err := parseOutputFormat(value)
			if err != nil {
				return versionOutputOptions{}, err
			}
			opts.format, i = format, next
		default:
			return versionOutputOptions{}, fmt.Errorf("unknown version option %q", arg)
		}
	}
	return opts, nil
}

func (a *App) versionJSON() error {
	type output struct {
		Version   string `json:"version"`
		Commit    string `json:"commit,omitempty"`
		BuiltAt   string `json:"built_at,omitempty"`
		GoVersion string `json:"go_version"`
		Platform  string `json:"platform"`
	}

	out := output{
		Version:   nonEmpty(Version, "dev"),
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}
	if commit := strings.TrimSpace(Commit); commit != "" {
		out.Commit = commit
	}
	if buildDate := strings.TrimSpace(BuildDate); buildDate != "" {
		out.BuiltAt = buildDate
	}

	encoder := json.NewEncoder(a.stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(out)
}

func versionString() string {
	var parts []string
	parts = append(parts, "adp version "+nonEmpty(Version, "dev"))

	if commit := strings.TrimSpace(Commit); commit != "" {
		parts = append(parts, "commit: "+commit)
	}
	if buildDate := strings.TrimSpace(BuildDate); buildDate != "" {
		parts = append(parts, "built: "+buildDate)
	}

	parts = append(parts, "go: "+runtime.Version())
	parts = append(parts, "platform: "+runtime.GOOS+"/"+runtime.GOARCH)

	return strings.Join(parts, "\n") + "\n"
}

func nonEmpty(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
