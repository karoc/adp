package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"text/tabwriter"

	"github.com/karoc/adp/internal/schema"
	"github.com/karoc/adp/internal/workspace"
)

func (a *App) init(ctx context.Context, args []string) error {
	if len(args) != 0 {
		return errors.New("usage: adp init")
	}
	if a.deps.WorkspaceStore == nil {
		return errors.New("workspace store is not configured")
	}
	if err := a.deps.WorkspaceStore.Init(ctx); err != nil {
		return err
	}

	homeDir := a.deps.Layout.Home
	fmt.Fprintf(a.stdout, "initialized ADP home at %s\n", homeDir)
	fmt.Fprintln(a.stdout)
	fmt.Fprintln(a.stdout, "Next steps:")
	fmt.Fprintln(a.stdout, "  1. Add a workspace:")
	fmt.Fprintln(a.stdout, "     adp workspace add <name> /path/to/project")
	fmt.Fprintln(a.stdout)
	fmt.Fprintln(a.stdout, "  2. Check diagnostics:")
	fmt.Fprintln(a.stdout, "     adp workspace doctor <name>")
	fmt.Fprintln(a.stdout)
	fmt.Fprintln(a.stdout, "  3. See all commands:")
	fmt.Fprintln(a.stdout, "     adp --help")
	fmt.Fprintln(a.stdout)
	fmt.Fprintln(a.stdout, "Documentation: https://github.com/karoc/adp#quick-start")
	return nil
}

func (a *App) workspace(ctx context.Context, args []string) error {
	if a.deps.WorkspaceStore == nil {
		return errors.New("workspace store is not configured")
	}
	if len(args) == 0 {
		return errors.New("usage: adp workspace <add|list|show|remove|rename|doctor>")
	}

	switch args[0] {
	case "add":
		if len(args) != 3 {
			return errors.New("usage: adp workspace add <name> <project-root>")
		}
		name, projectRoot := args[1], args[2]

		// Check for name conflict first
		if existing, _, err := a.deps.WorkspaceStore.Get(ctx, name); err == nil {
			fmt.Fprintf(a.stderr, "adp: workspace %q already exists\n", name)
			fmt.Fprintf(a.stderr, "  current project root: %s\n", existing.Project.Root)
			fmt.Fprintf(a.stderr, "\nUse a different name or remove the existing workspace with:\n")
			fmt.Fprintf(a.stderr, "  adp workspace remove %s\n", name)
			return processExitError{code: 1}
		}

		if _, err := a.deps.WorkspaceStore.Add(ctx, name, projectRoot); err != nil {
			return err
		}
		fmt.Fprintf(a.stdout, "workspace %q added\n", name)
	case "list":
		return a.workspaceList(ctx, args[1:])
	case "show":
		return a.workspaceShow(ctx, args[1:])
	case "remove":
		if len(args) != 2 {
			return errors.New("usage: adp workspace remove <name>")
		}
		if err := a.deps.WorkspaceStore.Remove(ctx, args[1]); err != nil {
			return err
		}
		fmt.Fprintf(a.stdout, "workspace %q removed\n", args[1])
	case "rename":
		if len(args) != 3 {
			return errors.New("usage: adp workspace rename <old-name> <new-name>")
		}
		if _, err := a.deps.WorkspaceStore.Rename(ctx, args[1], args[2]); err != nil {
			return err
		}
		fmt.Fprintf(a.stdout, "workspace %q renamed to %q\n", args[1], args[2])
	case "doctor":
		return a.doctor(ctx, args[1:])
	default:
		return fmt.Errorf("unknown workspace command %q", args[0])
	}
	return nil
}

func (a *App) doctor(ctx context.Context, args []string) error {
	opts, err := parseDoctorArgs(args)
	if err != nil {
		return err
	}
	if a.deps.WorkspaceStore == nil {
		return errors.New("workspace store is not configured")
	}
	if opts.workspace != "" {
		report, err := a.deps.WorkspaceStore.Diagnose(ctx, opts.workspace)
		if err != nil {
			return err
		}
		return a.workspaceDoctorReports([]workspace.DiagnosticReport{report}, opts)
	}
	reports, err := a.deps.WorkspaceStore.DiagnoseAll(ctx)
	if err != nil {
		return err
	}
	return a.workspaceDoctorReports(reports, opts)
}

func (a *App) workspaceList(ctx context.Context, args []string) error {
	opts, err := parseWorkspaceOutputArgs(args, "adp workspace list [--format <text|json>]")
	if err != nil {
		return err
	}

	records, err := a.deps.WorkspaceStore.List(ctx)
	if err != nil {
		return err
	}

	if opts.format == outputFormatJSON {
		return a.workspaceListJSON(records)
	}

	writer := tabwriter.NewWriter(a.stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "NAME\tPROJECT ROOT\tWORKSPACE DIR")
	for _, record := range records {
		fmt.Fprintf(writer, "%s\t%s\t%s\n", record.Name, record.ProjectRoot, record.WorkspaceDir)
	}
	return writer.Flush()
}

func (a *App) workspaceListJSON(records []workspace.Record) error {
	type workspaceItem struct {
		Name         string `json:"name"`
		ProjectRoot  string `json:"project_root"`
		WorkspaceDir string `json:"workspace_dir"`
	}
	type output struct {
		Workspaces []workspaceItem `json:"workspaces"`
		Count      int             `json:"count"`
	}
	items := make([]workspaceItem, len(records))
	for i, record := range records {
		items[i] = workspaceItem{
			Name:         record.Name,
			ProjectRoot:  record.ProjectRoot,
			WorkspaceDir: record.WorkspaceDir,
		}
	}
	out := output{
		Workspaces: items,
		Count:      len(items),
	}
	encoder := json.NewEncoder(a.stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(out)
}

func (a *App) workspaceShow(ctx context.Context, args []string) error {
	opts, err := parseWorkspaceShowArgs(args)
	if err != nil {
		return err
	}
	cfg, workspaceDir, err := a.deps.WorkspaceStore.Get(ctx, opts.workspace)
	if err != nil {
		return err
	}
	if opts.format == outputFormatJSON {
		return a.workspaceShowJSON(cfg, workspaceDir)
	}
	fmt.Fprintf(a.stdout, "name: %s\n", cfg.Workspace.Name)
	fmt.Fprintf(a.stdout, "project_root: %s\n", cfg.Project.Root)
	fmt.Fprintf(a.stdout, "workspace_dir: %s\n", workspaceDir)
	fmt.Fprintf(a.stdout, "memory_enabled: %t\n", cfg.Memory.Enabled)
	fmt.Fprintf(a.stdout, "mcp_enabled: %t\n", cfg.MCP.Enabled)
	return nil
}

func (a *App) workspaceShowJSON(cfg *schema.Config, workspaceDir string) error {
	type agentInfo struct {
		Command        string `json:"command"`
		DefaultProfile string `json:"default_profile,omitempty"`
	}
	type output struct {
		Name          string               `json:"name"`
		ProjectRoot   string               `json:"project_root"`
		WorkspaceDir  string               `json:"workspace_dir"`
		MemoryEnabled bool                 `json:"memory_enabled"`
		MCPEnabled    bool                 `json:"mcp_enabled"`
		Agents        map[string]agentInfo `json:"agents,omitempty"`
	}

	out := output{
		Name:          cfg.Workspace.Name,
		ProjectRoot:   cfg.Project.Root,
		WorkspaceDir:  workspaceDir,
		MemoryEnabled: cfg.Memory.Enabled,
		MCPEnabled:    cfg.MCP.Enabled,
	}

	if len(cfg.Agents) > 0 {
		out.Agents = make(map[string]agentInfo)
		for name, agent := range cfg.Agents {
			out.Agents[name] = agentInfo{
				Command:        agent.Command,
				DefaultProfile: agent.Profile,
			}
		}
	}

	encoder := json.NewEncoder(a.stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(out)
}

func (a *App) workspaceDoctorReports(reports []workspace.DiagnosticReport, opts doctorOptions) error {
	if opts.format == outputFormatJSON {
		return a.workspaceDoctorJSON(reports)
	}

	writer := tabwriter.NewWriter(a.stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "WORKSPACE\tLEVEL\tCODE\tMESSAGE\tPATH")
	hasErrors := false
	for _, report := range reports {
		if report.HasErrors() {
			hasErrors = true
		}
		diagnostics := visibleDiagnostics(report.Diagnostics, opts.verbose)
		if len(diagnostics) == 0 {
			fmt.Fprintf(writer, "%s\tok\t-\tno issues\t%s\n", valueOrDash(report.Workspace), valueOrDash(report.WorkspaceDir))
			continue
		}
		for _, diagnostic := range diagnostics {
			fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n",
				valueOrDash(report.Workspace),
				diagnostic.Level,
				valueOrDash(diagnostic.Code),
				valueOrDash(diagnostic.Message),
				valueOrDash(diagnostic.Path),
			)
		}
	}
	if err := writer.Flush(); err != nil {
		return err
	}
	if hasErrors {
		return processExitError{code: 2}
	}
	return nil
}

func (a *App) workspaceDoctorJSON(reports []workspace.DiagnosticReport) error {
	out := workspaceDoctorOutput(reports)
	encoder := json.NewEncoder(a.stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(out); err != nil {
		return err
	}
	if out.HasErrors {
		return processExitError{code: 2}
	}
	return nil
}

func visibleDiagnostics(diagnostics []workspace.Diagnostic, verbose bool) []workspace.Diagnostic {
	if verbose {
		return diagnostics
	}
	out := make([]workspace.Diagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		if diagnostic.Level == workspace.DiagnosticLevelInfo {
			continue
		}
		out = append(out, diagnostic)
	}
	return out
}
