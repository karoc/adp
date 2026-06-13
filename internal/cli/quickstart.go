package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// quickstart implements the interactive onboarding command.
// It guides users through initializing ADP home and creating their first workspace.
func (a *App) quickstart(ctx context.Context, args []string) error {
	opts, err := parseQuickstartArgs(args)
	if err != nil {
		return err
	}

	if opts.NonInteractive {
		return a.quickstartNonInteractive(ctx, opts)
	}

	return a.quickstartInteractive(ctx, opts)
}

// QuickstartOptions holds configuration for the quickstart command.
type QuickstartOptions struct {
	NonInteractive bool
	ADPHome        string
	WorkspaceName  string
	ProjectRoot    string
	EnableMemory   bool
	EnableMCP      bool
	EnableAgents   bool
}

func (a *App) quickstartInteractive(ctx context.Context, opts QuickstartOptions) error {
	// Step 1: Initialize ADP home
	if err := a.interactiveInitHome(ctx, opts); err != nil {
		return err
	}

	// Step 2: Create first workspace
	if err := a.interactiveCreateWorkspace(ctx, opts); err != nil {
		return err
	}

	return nil
}

func (a *App) quickstartNonInteractive(ctx context.Context, opts QuickstartOptions) error {
	// Validate required parameters
	if opts.WorkspaceName == "" {
		return errors.New("--workspace-name is required in non-interactive mode")
	}
	if opts.ProjectRoot == "" {
		return errors.New("--project-root is required in non-interactive mode")
	}

	// Validate workspace name
	if err := validateWorkspaceName(opts.WorkspaceName); err != nil {
		return fmt.Errorf("invalid workspace name: %w", err)
	}

	// Expand and validate project root
	projectRoot, err := expandPath(opts.ProjectRoot)
	if err != nil {
		return fmt.Errorf("invalid project root path: %w", err)
	}
	if err := validateProjectRoot(projectRoot); err != nil {
		return fmt.Errorf("invalid project root: %w", err)
	}

	// Determine ADP home (use provided value or default)
	adpHome := opts.ADPHome
	if adpHome == "" {
		adpHome = a.deps.Layout.Home
	}

	// Initialize ADP home if needed
	fmt.Fprintf(a.stdout, "Initializing ADP home at %s...\n", adpHome)
	if err := a.deps.WorkspaceStore.Init(ctx); err != nil {
		return fmt.Errorf("failed to initialize ADP home: %w", err)
	}
	fmt.Fprintln(a.stdout, "✓ Initialized ADP home")

	// Create the workspace
	// Note: memory and MCP are enabled by default in workspace config
	// The --memory, --mcp, --agents flags are currently accepted but ignored
	// (workspace is created with default settings which enable these features)
	fmt.Fprintf(a.stdout, "Creating workspace %q...\n", opts.WorkspaceName)
	addArgs := []string{"add", opts.WorkspaceName, projectRoot}
	if err := a.workspace(ctx, addArgs); err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	fmt.Fprintf(a.stdout, "✓ Workspace %q created\n", opts.WorkspaceName)
	fmt.Fprintln(a.stdout, "✓ Setup complete!")

	return nil
}

func parseQuickstartArgs(args []string) (QuickstartOptions, error) {
	opts := QuickstartOptions{}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--non-interactive":
			opts.NonInteractive = true
		case "--adp-home":
			if i+1 >= len(args) {
				return opts, errors.New("--adp-home requires a value")
			}
			opts.ADPHome = args[i+1]
			i++
		case "--workspace-name":
			if i+1 >= len(args) {
				return opts, errors.New("--workspace-name requires a value")
			}
			opts.WorkspaceName = args[i+1]
			i++
		case "--project-root":
			if i+1 >= len(args) {
				return opts, errors.New("--project-root requires a value")
			}
			opts.ProjectRoot = args[i+1]
			i++
		case "--memory":
			opts.EnableMemory = true
		case "--mcp":
			opts.EnableMCP = true
		case "--agents":
			opts.EnableAgents = true
		case "--help", "-h":
			return opts, errors.New("help requested")
		default:
			return opts, fmt.Errorf("unknown option %q", arg)
		}
	}

	return opts, nil
}

// interactiveInitHome implements the ADP home initialization flow.
// It prompts for the home directory path and initializes it.
func (a *App) interactiveInitHome(ctx context.Context, opts QuickstartOptions) error {
	// Determine default ADP home path
	defaultHome := opts.ADPHome
	if defaultHome == "" {
		defaultHome = a.deps.Layout.Home
	}

	// Check if home already exists
	homeExists := false
	if _, err := os.Stat(defaultHome); err == nil {
		homeExists = true
	}

	// If home exists and wasn't explicitly provided via --adp-home, ask user
	if homeExists && opts.ADPHome == "" {
		confirmed, err := a.confirmExistingHome(defaultHome)
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(a.stdout, "\nSetup cancelled")
			return errors.New("setup cancelled by user")
		}
		fmt.Fprintln(a.stdout, "\nUsing existing ADP home")
		return nil
	}

	// Show welcome message
	fmt.Fprintln(a.stdout, "Welcome to ADP (Agent Development Platform)!")
	fmt.Fprintln(a.stdout, "\nThis wizard will help you set up your first workspace.")

	// Prompt for ADP home directory
	adpHome, err := a.promptForPath("ADP home directory", defaultHome, func(path string) error {
		if path == "" {
			return nil // Empty is OK, will use default
		}
		// Expand ~ to home directory
		expanded, err := expandPath(path)
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}
		// Check if path already exists
		if _, err := os.Stat(expanded); err == nil {
			return fmt.Errorf("directory already exists: %s", expanded)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Use default if user provided empty input
	if adpHome == "" {
		adpHome = defaultHome
	} else {
		// Expand ~ in user input
		expanded, err := expandPath(adpHome)
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}
		adpHome = expanded
	}

	// Initialize ADP home
	fmt.Fprintf(a.stdout, "\nInitializing ADP home at %s...\n", adpHome)
	if err := a.deps.WorkspaceStore.Init(ctx); err != nil {
		return fmt.Errorf("failed to initialize ADP home: %w", err)
	}

	fmt.Fprintln(a.stdout, "✓ Initialized ADP home")
	return nil
}

// confirmExistingHome asks the user if they want to continue with an existing home.
func (a *App) confirmExistingHome(homePath string) (bool, error) {
	fmt.Fprintf(a.stdout, "\nADP home already exists at: %s\n", homePath)
	fmt.Fprint(a.stdout, "Continue with existing ADP home? [y/N]: ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, err
		}
		return false, errors.New("input cancelled")
	}

	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return answer == "y" || answer == "yes", nil
}

// promptForPath prompts the user for a file path with validation.
func (a *App) promptForPath(label, defaultValue string, validate func(string) error) (string, error) {
	for {
		fmt.Fprintf(a.stdout, "%s [%s]: ", label, defaultValue)

		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return "", err
			}
			return "", errors.New("input cancelled")
		}

		input := strings.TrimSpace(scanner.Text())

		// Validate input
		if validate != nil {
			if err := validate(input); err != nil {
				fmt.Fprintf(a.stderr, "Error: %v\n", err)
				continue
			}
		}

		return input, nil
	}
}

// expandPath expands ~ to the user's home directory.
func expandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if len(path) == 1 {
			return home, nil
		}
		return filepath.Join(home, path[1:]), nil
	}
	return path, nil
}

// interactiveCreateWorkspace implements the workspace creation flow.
// It prompts for workspace name, project root, and optional configurations.
func (a *App) interactiveCreateWorkspace(ctx context.Context, opts QuickstartOptions) error {
	fmt.Fprintln(a.stdout, "\nSetting up your first workspace...")

	// Prompt for workspace name
	workspaceName, err := a.promptForWorkspaceName(opts.WorkspaceName)
	if err != nil {
		return err
	}

	// Prompt for project root
	projectRoot, err := a.promptForProjectRoot(opts.ProjectRoot)
	if err != nil {
		return err
	}

	// Prompt for optional configurations
	enableMemory := opts.EnableMemory
	enableMCP := opts.EnableMCP

	if !opts.NonInteractive {
		enableMemory, err = a.promptYesNo("Enable memory?", true)
		if err != nil {
			return err
		}

		enableMCP, err = a.promptYesNo("Enable MCP?", true)
		if err != nil {
			return err
		}
	}

	// Create the workspace
	fmt.Fprintf(a.stdout, "\nCreating workspace %q...\n", workspaceName)

	// Build workspace add arguments
	addArgs := []string{"add", workspaceName, projectRoot}
	if enableMemory {
		addArgs = append(addArgs, "--memory")
	}
	if enableMCP {
		addArgs = append(addArgs, "--mcp")
	}

	if err := a.workspace(ctx, addArgs); err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	fmt.Fprintf(a.stdout, "✓ Workspace %q created\n", workspaceName)

	// Ask if user wants to run diagnostics
	if !opts.NonInteractive {
		runDiagnostics, err := a.promptYesNo("\nRun workspace diagnostics now?", true)
		if err != nil {
			return err
		}

		if runDiagnostics {
			fmt.Fprintln(a.stdout, "\nRunning diagnostics...")
			if err := a.doctor(ctx, []string{workspaceName}); err != nil {
				// Don't fail the quickstart if diagnostics fail
				fmt.Fprintf(a.stderr, "Warning: diagnostics failed: %v\n", err)
			}
		}
	}

	// Show next steps
	a.showNextSteps(workspaceName)

	return nil
}

// promptForWorkspaceName prompts for a workspace name with validation.
func (a *App) promptForWorkspaceName(defaultValue string) (string, error) {
	if defaultValue != "" {
		// Non-interactive mode or provided via flag
		if err := validateWorkspaceName(defaultValue); err != nil {
			return "", err
		}
		return defaultValue, nil
	}

	for {
		fmt.Fprint(a.stdout, "\nWorkspace name: ")

		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return "", err
			}
			return "", errors.New("input cancelled")
		}

		name := strings.TrimSpace(scanner.Text())

		// Validate name
		if err := validateWorkspaceName(name); err != nil {
			fmt.Fprintf(a.stderr, "Error: %v\n", err)
			continue
		}

		return name, nil
	}
}

// validateWorkspaceName validates a workspace name.
func validateWorkspaceName(name string) error {
	if name == "" {
		return errors.New("workspace name cannot be empty")
	}

	// Check for invalid characters (allow alphanumeric, dash, underscore, dot)
	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '_' || ch == '.') {
			return fmt.Errorf("workspace name contains invalid character %q (use only letters, numbers, -, _, .)", ch)
		}
	}

	return nil
}

// promptForProjectRoot prompts for a project root path with validation.
func (a *App) promptForProjectRoot(defaultValue string) (string, error) {
	if defaultValue != "" {
		// Non-interactive mode or provided via flag
		expanded, err := expandPath(defaultValue)
		if err != nil {
			return "", fmt.Errorf("invalid path: %w", err)
		}
		if err := validateProjectRoot(expanded); err != nil {
			return "", err
		}
		return expanded, nil
	}

	// Suggest current directory as default
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	for {
		fmt.Fprintf(a.stdout, "Project root [%s]: ", cwd)

		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return "", err
			}
			return "", errors.New("input cancelled")
		}

		input := strings.TrimSpace(scanner.Text())

		// Use default if empty
		if input == "" {
			input = cwd
		}

		// Expand ~ in path
		expanded, err := expandPath(input)
		if err != nil {
			fmt.Fprintf(a.stderr, "Error: invalid path: %v\n", err)
			continue
		}

		// Validate path
		if err := validateProjectRoot(expanded); err != nil {
			fmt.Fprintf(a.stderr, "Error: %v\n", err)

			// Ask if user wants to create the directory
			create, err := a.promptYesNo("Create directory?", false)
			if err != nil {
				return "", err
			}

			if create {
				if err := os.MkdirAll(expanded, 0755); err != nil {
					fmt.Fprintf(a.stderr, "Error: failed to create directory: %v\n", err)
					continue
				}
				fmt.Fprintf(a.stdout, "✓ Created directory %s\n", expanded)
				return expanded, nil
			}
			continue
		}

		return expanded, nil
	}
}

// validateProjectRoot validates a project root path.
func validateProjectRoot(path string) error {
	if path == "" {
		return errors.New("project root cannot be empty")
	}

	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", path)
		}
		return fmt.Errorf("cannot access path: %w", err)
	}

	// Check if it's a directory
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}

	return nil
}

// promptYesNo prompts for a yes/no answer.
func (a *App) promptYesNo(question string, defaultYes bool) (bool, error) {
	prompt := question + " [Y/n]: "
	if !defaultYes {
		prompt = question + " [y/N]: "
	}

	fmt.Fprint(a.stdout, prompt)

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, err
		}
		return false, errors.New("input cancelled")
	}

	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))

	// Empty answer uses default
	if answer == "" {
		return defaultYes, nil
	}

	return answer == "y" || answer == "yes", nil
}

// showNextSteps displays helpful next steps after quickstart.
func (a *App) showNextSteps(workspaceName string) {
	fmt.Fprintln(a.stdout, "\n"+strings.Repeat("─", 50))
	fmt.Fprintln(a.stdout, "✓ Setup complete!")
	fmt.Fprintln(a.stdout, "\nNext steps:")
	fmt.Fprintf(a.stdout, "  - Start an agent:    adp run codex --workspace %s\n", workspaceName)
	fmt.Fprintf(a.stdout, "  - Add a task:        adp tasks add --workspace %s \"First task\"\n", workspaceName)
	fmt.Fprintln(a.stdout, "  - See all commands:  adp --help")
	fmt.Fprintln(a.stdout, strings.Repeat("─", 50))
}
