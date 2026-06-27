package planner

import (
	"testing"

	"github.com/UsingCoding/bmx/internal/model"
)

func TestBuildPlanInstallUninstallAndKeep(t *testing.T) {
	t.Parallel()

	docker, _ := model.NewApp("brew:docker")
	git, _ := model.NewApp("brew:lazygit")
	gimp, _ := model.NewApp("brew-cask:gimp")

	plan := Build([]model.App{docker, gimp}, []model.App{docker, git})

	if len(plan.Installs) != 1 || plan.Installs[0].Name != "brew-cask:gimp" {
		t.Fatalf("unexpected installs: %+v", plan.Installs)
	}
	if len(plan.Uninstalls) != 1 || plan.Uninstalls[0].Name != "brew:lazygit" {
		t.Fatalf("unexpected uninstalls: %+v", plan.Uninstalls)
	}
	if len(plan.Keeps) != 1 || plan.Keeps[0].Name != "brew:docker" {
		t.Fatalf("unexpected keeps: %+v", plan.Keeps)
	}
}

func TestEmptyPlan(t *testing.T) {
	t.Parallel()

	docker, _ := model.NewApp("brew:docker")
	plan := Build([]model.App{docker}, []model.App{docker})
	if !plan.Empty() {
		t.Fatal("expected empty plan")
	}
}
