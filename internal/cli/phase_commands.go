package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"text/tabwriter"

	taskstore "github.com/karoc/adp/internal/tasks"
)

func (a *App) phase(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: adp phase <add|list|show|start|accept|commit|push>")
	}

	switch args[0] {
	case "add":
		return a.phaseAdd(ctx, args[1:])
	case "list":
		return a.phaseList(ctx, args[1:])
	case "show":
		return a.phaseShow(ctx, args[1:])
	case "start":
		return a.phaseStart(ctx, args[1:])
	case "accept":
		return a.phaseAccept(ctx, args[1:])
	case "commit":
		return a.phaseCommit(ctx, args[1:])
	case "push":
		return a.phasePush(ctx, args[1:])
	default:
		return fmt.Errorf("unknown phase command %q", args[0])
	}
}

func (a *App) phaseAdd(ctx context.Context, args []string) error {
	opts, err := parsePhaseAddArgs(args)
	if err != nil {
		return err
	}
	store, _, err := a.loadTaskStore(ctx, opts.workspace)
	if err != nil {
		return err
	}
	phase, err := store.AddPhase(ctx, taskstore.PhaseAddRequest{ID: opts.id, Title: opts.title, Goal: opts.goal})
	if err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "phase %s added\n", phase.ID)
	return nil
}

func (a *App) phaseList(ctx context.Context, args []string) error {
	workspace, err := parseWorkspaceOnlyArgs(args, "adp phase list [--workspace <name>]")
	if err != nil {
		return err
	}
	store, _, err := a.loadTaskStore(ctx, workspace)
	if err != nil {
		return err
	}
	phases, err := store.ListPhases(ctx)
	if err != nil {
		return err
	}

	writer := tabwriter.NewWriter(a.stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "ID\tSTATUS\tUPDATED\tTITLE")
	for _, phase := range phases {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", phase.ID, phase.Status, formatEventTime(phase.UpdatedAt), phase.Title)
	}
	return writer.Flush()
}

func (a *App) phaseShow(ctx context.Context, args []string) error {
	workspace, phaseID, err := parsePhaseIDArgs(args, "adp phase show [--workspace <name>] <phase-id>")
	if err != nil {
		return err
	}
	store, _, err := a.loadTaskStore(ctx, workspace)
	if err != nil {
		return err
	}
	phase, err := store.GetPhase(ctx, phaseID)
	if err != nil {
		return err
	}
	a.printPhase(phase)
	return nil
}

func (a *App) phaseStart(ctx context.Context, args []string) error {
	workspace, phaseID, err := parsePhaseIDArgs(args, "adp phase start [--workspace <name>] <phase-id>")
	if err != nil {
		return err
	}
	store, _, err := a.loadTaskStore(ctx, workspace)
	if err != nil {
		return err
	}
	phase, err := store.StartPhase(ctx, phaseID)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "phase %s status: %s\n", phase.ID, phase.Status)
	return nil
}

func (a *App) phaseAccept(ctx context.Context, args []string) error {
	opts, err := parsePhaseAcceptArgs(args)
	if err != nil {
		return err
	}
	store, _, err := a.loadTaskStore(ctx, opts.workspace)
	if err != nil {
		return err
	}
	phase, err := store.AcceptPhase(ctx, taskstore.PhaseAcceptRequest{
		ID:       opts.id,
		Commands: opts.commands,
		Result:   opts.result,
		Notes:    opts.notes,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "phase %s accepted: %s\n", phase.ID, phase.Acceptance.Result)
	return nil
}

func (a *App) phaseCommit(ctx context.Context, args []string) error {
	opts, err := parsePhaseCommitArgs(args)
	if err != nil {
		return err
	}
	store, _, err := a.loadTaskStore(ctx, opts.workspace)
	if err != nil {
		return err
	}
	phase, err := store.RecordPhaseCommit(ctx, taskstore.PhaseCommitRequest{
		ID:      opts.id,
		Hash:    opts.hash,
		Message: opts.message,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "phase %s commit: %s\n", phase.ID, phase.Commit.Hash)
	return nil
}

func (a *App) phasePush(ctx context.Context, args []string) error {
	opts, err := parsePhasePushArgs(args)
	if err != nil {
		return err
	}
	store, _, err := a.loadTaskStore(ctx, opts.workspace)
	if err != nil {
		return err
	}
	phase, err := store.RecordPhasePush(ctx, taskstore.PhasePushRequest{
		ID:     opts.id,
		Remote: opts.remote,
		Branch: opts.branch,
		Result: opts.result,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "phase %s push: %s/%s %s\n", phase.ID, phase.Push.Remote, phase.Push.Branch, phase.Push.Result)
	return nil
}

func (a *App) printPhase(phase taskstore.Phase) {
	fmt.Fprintf(a.stdout, "id: %s\n", phase.ID)
	fmt.Fprintf(a.stdout, "title: %s\n", phase.Title)
	fmt.Fprintf(a.stdout, "status: %s\n", phase.Status)
	fmt.Fprintf(a.stdout, "goal: %s\n", valueOrDash(phase.Goal))
	fmt.Fprintf(a.stdout, "acceptance_result: %s\n", valueOrDash(phase.Acceptance.Result))
	fmt.Fprintf(a.stdout, "acceptance_commands: %s\n", valueOrDash(strings.Join(phase.Acceptance.Commands, "; ")))
	fmt.Fprintf(a.stdout, "acceptance_notes: %s\n", valueOrDash(phase.Acceptance.Notes))
	fmt.Fprintf(a.stdout, "commit_hash: %s\n", valueOrDash(phase.Commit.Hash))
	fmt.Fprintf(a.stdout, "commit_message: %s\n", valueOrDash(phase.Commit.Message))
	fmt.Fprintf(a.stdout, "push_remote: %s\n", valueOrDash(phase.Push.Remote))
	fmt.Fprintf(a.stdout, "push_branch: %s\n", valueOrDash(phase.Push.Branch))
	fmt.Fprintf(a.stdout, "push_result: %s\n", valueOrDash(phase.Push.Result))
	fmt.Fprintf(a.stdout, "created_at: %s\n", formatEventTime(phase.CreatedAt))
	fmt.Fprintf(a.stdout, "updated_at: %s\n", formatEventTime(phase.UpdatedAt))
}
