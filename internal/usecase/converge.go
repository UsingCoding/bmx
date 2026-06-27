package usecase

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/UsingCoding/bmx/internal/state"
)

func Converge(ctx context.Context, input ConvergeInput) error {
	planResult, err := BuildPlan(ctx, input.PlanInput)
	if err != nil {
		return err
	}

	renderPlan(input.Out, planResult)
	if planResult.Plan.Empty() {
		_, err := fmt.Fprintln(input.Out, "No changes.")
		return err
	}

	approved, err := confirmApply(input.In, input.Out)
	if err != nil {
		return err
	}
	if !approved {
		_, err := fmt.Fprintln(input.Out, "Aborted.")
		return err
	}

	for _, app := range planResult.Plan.Uninstalls {
		manager, err := input.Managers.For(app)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(input.Out, "Uninstalling %s\n", app.Name); err != nil {
			return err
		}
		if err := manager.Uninstall(ctx, app); err != nil {
			return err
		}
	}

	for _, app := range planResult.Plan.Installs {
		manager, err := input.Managers.For(app)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(input.Out, "Installing %s\n", app.Name); err != nil {
			return err
		}
		if err := manager.Install(ctx, app); err != nil {
			return err
		}
	}

	if err := state.Write(input.StatePath, state.FromApps(planResult.ActiveList, planResult.Desired)); err != nil {
		return err
	}

	_, err = fmt.Fprintln(input.Out, "Converge complete.")
	return err
}

func renderPlan(out io.Writer, result PlanResult) {
	fmt.Fprintf(out, "List: %s\n", result.ActiveList)
	if len(result.Plan.Installs) > 0 {
		fmt.Fprintln(out, "Install:")
		for _, app := range result.Plan.Installs {
			fmt.Fprintf(out, "  + %s\n", app.Name)
		}
	}
	if len(result.Plan.Uninstalls) > 0 {
		fmt.Fprintln(out, "Uninstall:")
		for _, app := range result.Plan.Uninstalls {
			fmt.Fprintf(out, "  - %s\n", app.Name)
		}
	}
	if len(result.Plan.Keeps) > 0 {
		fmt.Fprintln(out, "Keep:")
		for _, app := range result.Plan.Keeps {
			fmt.Fprintf(out, "    %s\n", app.Name)
		}
	}
}

func confirmApply(in io.Reader, out io.Writer) (bool, error) {
	if _, err := fmt.Fprint(out, "Apply plan? [y/N]: "); err != nil {
		return false, err
	}

	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}

	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}
