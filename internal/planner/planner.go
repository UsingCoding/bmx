package planner

import (
	"sort"

	"github.com/UsingCoding/bmx/internal/model"
)

type Plan struct {
	Installs   []model.App
	Uninstalls []model.App
	Keeps      []model.App
}

func Build(desired, current []model.App) Plan {
	desiredByKey := make(map[string]model.App, len(desired))
	for _, app := range desired {
		desiredByKey[app.Key()] = app
	}

	currentByKey := make(map[string]model.App, len(current))
	for _, app := range current {
		currentByKey[app.Key()] = app
	}

	plan := Plan{}
	for key, app := range desiredByKey {
		if _, ok := currentByKey[key]; ok {
			plan.Keeps = append(plan.Keeps, app)
			continue
		}
		plan.Installs = append(plan.Installs, app)
	}

	for key, app := range currentByKey {
		if _, ok := desiredByKey[key]; ok {
			continue
		}
		plan.Uninstalls = append(plan.Uninstalls, app)
	}

	sortApps(plan.Installs)
	sortApps(plan.Uninstalls)
	sortApps(plan.Keeps)

	return plan
}

func (p Plan) Empty() bool {
	return len(p.Installs) == 0 && len(p.Uninstalls) == 0
}

func sortApps(apps []model.App) {
	sort.Slice(apps, func(i, j int) bool {
		return apps[i].Name < apps[j].Name
	})
}
