package usecase

import (
	"context"
	"fmt"
	"io"

	"github.com/UsingCoding/bmx/internal/backend"
	"github.com/UsingCoding/bmx/internal/config"
	"github.com/UsingCoding/bmx/internal/model"
	"github.com/UsingCoding/bmx/internal/planner"
	"github.com/UsingCoding/bmx/internal/state"
)

type PlanInput struct {
	ConfigPath string
	StatePath  string
	ListName   string
}

type PlanResult struct {
	Config     config.File
	State      state.File
	ActiveList string
	Desired    []model.App
	Current    []model.App
	Plan       planner.Plan
}

func BuildPlan(_ context.Context, input PlanInput) (PlanResult, error) {
	cfg, err := config.Load(input.ConfigPath)
	if err != nil {
		return PlanResult{}, err
	}

	st, err := state.Load(input.StatePath)
	if err != nil {
		return PlanResult{}, err
	}

	activeList, err := selectList(cfg, st, input.ListName)
	if err != nil {
		return PlanResult{}, err
	}

	desired, err := config.ResolveList(cfg, activeList)
	if err != nil {
		return PlanResult{}, err
	}
	current := st.InstalledApps()

	return PlanResult{
		Config:     cfg,
		State:      st,
		ActiveList: activeList,
		Desired:    desired,
		Current:    current,
		Plan:       planner.Build(desired, current),
	}, nil
}

func selectList(cfg config.File, st state.File, explicit string) (string, error) {
	if explicit != "" {
		if _, ok := config.FindList(cfg, explicit); !ok {
			return "", fmt.Errorf("unknown list %q", explicit)
		}
		return explicit, nil
	}

	if st.ActiveList != "" {
		if _, ok := config.FindList(cfg, st.ActiveList); ok {
			return st.ActiveList, nil
		}
	}

	if len(cfg.Lists) == 1 {
		return cfg.Lists[0].Name, nil
	}

	return "", fmt.Errorf("list is ambiguous: pass --list or set active_list in state")
}

type ConvergeInput struct {
	PlanInput
	Managers backend.Registry
	In       io.Reader
	Out      io.Writer
}
