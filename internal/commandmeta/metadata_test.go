package commandmeta

import (
	"strings"
	"testing"
)

func TestMetadataHasUniqueRootCommandsAndUsage(t *testing.T) {
	t.Parallel()

	seen := map[string]bool{}
	for _, command := range Commands() {
		if command.Name == "" {
			t.Fatal("command name must not be empty")
		}
		if seen[command.Name] {
			t.Fatalf("duplicate root command %q", command.Name)
		}
		seen[command.Name] = true
		if len(command.Usage) == 0 {
			t.Fatalf("command %q has no usage lines", command.Name)
		}
		for _, line := range command.Usage {
			if !strings.HasPrefix(line, "adp "+command.Name) {
				t.Fatalf("usage line %q does not belong to command %q", line, command.Name)
			}
		}
		assertUniqueValues(t, command.Name+" subcommands", command.Subcommands)
		assertUniqueValues(t, command.Name+" options", command.Options)
	}
}

func TestUsageIncludesEveryMetadataLine(t *testing.T) {
	t.Parallel()

	usage := Usage()
	for _, line := range UsageLines() {
		if !strings.Contains(usage, "  "+line+"\n") {
			t.Fatalf("usage missing %q:\n%s", line, usage)
		}
	}
}

func TestUsageOptionsAreDeclared(t *testing.T) {
	t.Parallel()

	for _, command := range Commands() {
		declared := map[string]bool{}
		for _, option := range command.Options {
			declared[option.Name] = true
		}
		for _, line := range command.Usage {
			for _, field := range strings.Fields(line) {
				option := usageOption(field)
				if option == "" || declared[option] {
					continue
				}
				t.Fatalf("usage line %q references undeclared option %q", line, option)
			}
		}
	}
}

func TestCommandHelpIncludesUsageAndValues(t *testing.T) {
	t.Parallel()

	help, ok := CommandHelp("tasks")
	if !ok {
		t.Fatal("CommandHelp(tasks) returned false")
	}
	for _, want := range []string{
		"adp tasks - manage the local workspace task board",
		"Usage:",
		"adp tasks add",
		"Subcommands:",
		"take - atomically claim next work",
		"Options:",
		"--workspace - workspace name",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("command help missing %q:\n%s", want, help)
		}
	}
}

func TestSubcommandHelpIncludesFocusedUsage(t *testing.T) {
	t.Parallel()

	help, ok := SubcommandHelp("phase", "commit")
	if !ok {
		t.Fatal("SubcommandHelp(phase, commit) returned false")
	}
	for _, want := range []string{
		"adp phase commit",
		"Usage:",
		"adp phase commit [--workspace <name>] <phase-id> --hash <commit-hash> [--message <text>]",
		"See also:",
		"adp phase --help",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("subcommand help missing %q:\n%s", want, help)
		}
	}
	if strings.Contains(help, "adp phase add") {
		t.Fatalf("subcommand help included unrelated usage:\n%s", help)
	}
}

func TestHelpIncludesCopyableExamples(t *testing.T) {
	t.Parallel()

	help, ok := CommandHelp("run")
	if !ok {
		t.Fatal("CommandHelp(run) returned false")
	}
	for _, want := range []string{
		"Examples:",
		"adp run codex --workspace game-a --take --owner codex-main --lease 4h",
		"adp run claude --workspace game-a --task task-20260611-0001 --keep-runtime",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("run help missing example %q:\n%s", want, help)
		}
	}

	help, ok = SubcommandHelp("tasks", "take")
	if !ok {
		t.Fatal("SubcommandHelp(tasks, take) returned false")
	}
	for _, want := range []string{
		"Examples:",
		"adp tasks take --workspace game-a --owner codex-main --lease 4h --format json",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("tasks take help missing example %q:\n%s", want, help)
		}
	}
	if strings.Contains(help, "adp tasks claim --workspace") {
		t.Fatalf("tasks take help included claim example:\n%s", help)
	}

	help, ok = SubcommandHelp("phase", "accept")
	if !ok {
		t.Fatal("SubcommandHelp(phase, accept) returned false")
	}
	if want := `adp phase accept --workspace game-a P60 --command "scripts/check-all.sh" --result passed --notes "runtime smoke passed"`; !strings.Contains(help, want) {
		t.Fatalf("phase accept help missing example %q:\n%s", want, help)
	}
}

func TestFirstUseHelpIncludesCopyableExamples(t *testing.T) {
	t.Parallel()

	commandCases := []struct {
		command string
		want    string
	}{
		{command: "workspace", want: "adp workspace doctor game-a --format json"},
		{command: "completion", want: "adp completion values tasks --workspace game-a"},
		{command: "events", want: "adp events list --workspace game-a --task task-20260611-0001 --format json"},
		{command: "sessions", want: "adp sessions resume-plan session-20260611-0001 --workspace game-a --agent claude --owner claude-main --lease 4h"},
		{command: "runtime", want: "adp runtime prune --older-than 24h --dry-run --format json"},
		{command: "plan", want: "adp plan doctor --workspace game-a --format json"},
		{command: "progress", want: "adp progress report --workspace game-a --format json"},
	}
	for _, tc := range commandCases {
		help, ok := CommandHelp(tc.command)
		if !ok {
			t.Fatalf("CommandHelp(%s) returned false", tc.command)
		}
		for _, want := range []string{"Examples:", tc.want} {
			if !strings.Contains(help, want) {
				t.Fatalf("%s help missing %q:\n%s", tc.command, want, help)
			}
		}
	}

	subcommandCases := []struct {
		command    string
		subcommand string
		want       string
	}{
		{command: "workspace", subcommand: "doctor", want: "adp workspace doctor game-a --format json"},
		{command: "completion", subcommand: "values", want: "adp completion values workspaces"},
		{command: "events", subcommand: "list", want: "adp events list --workspace game-a --task task-20260611-0001 --type run_finished --limit 5 --format json"},
		{command: "sessions", subcommand: "restore-plan", want: "adp sessions restore-plan session-20260611-0001 --format json"},
		{command: "sessions", subcommand: "resume-plan", want: "adp sessions resume-plan session-20260611-0001 --workspace game-a --agent claude --owner claude-main --lease 4h --format json"},
		{command: "runtime", subcommand: "prune", want: "adp runtime prune --older-than 24h --dry-run --format json"},
		{command: "tasks", subcommand: "stale", want: "adp tasks stale --workspace game-a --format json"},
		{command: "plan", subcommand: "doctor", want: "adp plan doctor --workspace game-a --format json"},
		{command: "progress", subcommand: "report", want: "adp progress report --workspace game-a --format json"},
	}
	for _, tc := range subcommandCases {
		help, ok := SubcommandHelp(tc.command, tc.subcommand)
		if !ok {
			t.Fatalf("SubcommandHelp(%s, %s) returned false", tc.command, tc.subcommand)
		}
		for _, want := range []string{"Examples:", tc.want} {
			if !strings.Contains(help, want) {
				t.Fatalf("%s %s help missing %q:\n%s", tc.command, tc.subcommand, want, help)
			}
		}
	}
}

func assertUniqueValues(t *testing.T, label string, values []Value) {
	t.Helper()

	seen := map[string]bool{}
	for _, value := range values {
		if value.Name == "" {
			t.Fatalf("%s contains empty value", label)
		}
		if seen[value.Name] {
			t.Fatalf("%s contains duplicate value %q", label, value.Name)
		}
		seen[value.Name] = true
	}
}

func usageOption(field string) string {
	field = strings.Trim(field, "[],")
	if strings.HasPrefix(field, "--") {
		return field
	}
	if strings.HasPrefix(field, "-") && len(field) == 2 {
		return field
	}
	return ""
}

func TestSeeAlsoSection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		command    string
		subcommand string
		wantRefs   []string
	}{
		{
			name:       "tasks take subcommand",
			command:    "tasks",
			subcommand: "take",
			wantRefs:   []string{"run --take", "tasks next --help", "tasks renew --help", "tasks --help"},
		},
		{
			name:       "tasks claim subcommand",
			command:    "tasks",
			subcommand: "claim",
			wantRefs:   []string{"tasks take --help", "tasks renew --help", "tasks --help"},
		},
		{
			name:       "doctor root command",
			command:    "doctor",
			subcommand: "",
			wantRefs:   []string{"workspace doctor --help", "plan doctor --help"},
		},
		{
			name:       "workspace doctor subcommand",
			command:    "workspace",
			subcommand: "doctor",
			wantRefs:   []string{"doctor --help", "plan doctor --help", "workspace --help"},
		},
		{
			name:       "sessions restore-plan subcommand",
			command:    "sessions",
			subcommand: "restore-plan",
			wantRefs:   []string{"sessions resume-plan --help", "run --help", "sessions --help"},
		},
		{
			name:       "sessions resume-plan subcommand",
			command:    "sessions",
			subcommand: "resume-plan",
			wantRefs:   []string{"sessions restore-plan --help", "run --take", "sessions --help"},
		},
		{
			name:       "run root command",
			command:    "run",
			subcommand: "",
			wantRefs:   []string{"tasks --help", "events --help", "sessions --help"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf strings.Builder
			writeSeeAlsoSection(&buf, tt.command, tt.subcommand)
			output := buf.String()

			if len(tt.wantRefs) == 0 {
				if output != "" {
					t.Errorf("expected no see-also section, got:\n%s", output)
				}
				return
			}

			if !strings.Contains(output, "See also:") {
				t.Errorf("see-also section missing header:\n%s", output)
			}

			for _, ref := range tt.wantRefs {
				if !strings.Contains(output, "adp "+ref) {
					t.Errorf("see-also section missing reference %q:\n%s", ref, output)
				}
			}
		})
	}
}

func TestRelationshipIntegrity(t *testing.T) {
	t.Parallel()

	// Build map of all valid commands and subcommands
	allCommands := map[string]bool{}
	for _, cmd := range rootCommands {
		allCommands[cmd.Name] = true
		for _, sub := range cmd.Subcommands {
			allCommands[cmd.Name+"."+sub.Name] = true
		}
	}

	// Check root command relationships
	for cmd, refs := range commandRelationships {
		if !allCommands[cmd] {
			t.Errorf("commandRelationships references non-existent command %q", cmd)
		}
		for _, ref := range refs {
			// Extract command name from reference (e.g., "workspace doctor" -> "workspace")
			parts := strings.Fields(ref)
			if len(parts) == 0 {
				t.Errorf("command %q has empty reference", cmd)
				continue
			}
			refCmd := parts[0]
			// Check if it's a valid root command or subcommand reference
			if len(parts) >= 2 && parts[1] != "--help" {
				// It's a subcommand reference like "workspace doctor"
				refKey := refCmd + "." + parts[1]
				if !allCommands[refKey] {
					t.Errorf("command %q references non-existent subcommand %q", cmd, refKey)
				}
			} else if !allCommands[refCmd] {
				t.Errorf("command %q references non-existent command %q", cmd, refCmd)
			}
		}
	}

	// Check subcommand relationships
	for subcmd, refs := range subcommandRelationships {
		if !allCommands[subcmd] {
			t.Errorf("subcommandRelationships references non-existent subcommand %q", subcmd)
		}
		for _, ref := range refs {
			parts := strings.Fields(ref)
			if len(parts) == 0 {
				t.Errorf("subcommand %q has empty reference", subcmd)
				continue
			}
			refCmd := parts[0]
			// Check if it's a valid root command or subcommand reference
			if len(parts) >= 2 && parts[1] != "--help" && parts[1] != "--take" {
				// It's a subcommand reference
				refKey := refCmd + "." + parts[1]
				if !allCommands[refKey] {
					t.Errorf("subcommand %q references non-existent subcommand %q", subcmd, refKey)
				}
			} else if !allCommands[refCmd] {
				t.Errorf("subcommand %q references non-existent command %q", subcmd, refCmd)
			}
		}
	}
}

func TestP0CommandsHaveSeeAlso(t *testing.T) {
	t.Parallel()

	p0Cases := []struct {
		command    string
		subcommand string
	}{
		{command: "tasks", subcommand: "take"},
		{command: "tasks", subcommand: "claim"},
		{command: "doctor", subcommand: ""},
		{command: "workspace", subcommand: "doctor"},
		{command: "sessions", subcommand: "restore-plan"},
		{command: "sessions", subcommand: "resume-plan"},
		{command: "run", subcommand: ""},
	}

	for _, tc := range p0Cases {
		var help string
		var ok bool

		if tc.subcommand != "" {
			help, ok = SubcommandHelp(tc.command, tc.subcommand)
			if !ok {
				t.Fatalf("SubcommandHelp(%s, %s) returned false", tc.command, tc.subcommand)
			}
		} else {
			help, ok = CommandHelp(tc.command)
			if !ok {
				t.Fatalf("CommandHelp(%s) returned false", tc.command)
			}
		}

		if !strings.Contains(help, "See also:") {
			t.Errorf("P0 command %s %s missing 'See also:' section:\n%s",
				tc.command, tc.subcommand, help)
		}
	}
}
