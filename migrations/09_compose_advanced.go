package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// ── Add desired_state field to containers ──────────────
		containers, err := app.FindCollectionByNameOrId("containers")
		if err != nil {
			return err
		}
		containers.Fields.Add(&core.SelectField{
			Name:      "desired_state",
			Values:    []string{"none", "running", "stopped"},
			MaxSelect: 1,
		})
		if err := app.Save(containers); err != nil {
			return err
		}

		// ── Add source field to compose_stacks ────────────────
		stacks, err := app.FindCollectionByNameOrId("compose_stacks")
		if err != nil {
			return err
		}
		stacks.Fields.Add(&core.SelectField{
			Name:      "source",
			Values:    []string{"managed", "imported"},
			MaxSelect: 1,
		})
		if err := app.Save(stacks); err != nil {
			return err
		}

		return nil
	}, nil)
}
