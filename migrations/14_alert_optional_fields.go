package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		alertRules, err := app.FindCollectionByNameOrId("alert_rules")
		if err != nil {
			return err
		}

		// Make metric-specific fields optional so event-type rules can be created.
		alertRules.Fields.Add(
			&core.TextField{Name: "metric", Max: 100},
			&core.SelectField{
				Name:      "operator",
				Values:    []string{"gt", "lt", "gte", "lte", "eq"},
				MaxSelect: 1,
			},
			&core.NumberField{Name: "threshold"},
			&core.SelectField{
				Name:      "target_type",
				Values:    []string{"server", "container"},
				MaxSelect: 1,
			},
		)

		return app.Save(alertRules)
	}, func(app core.App) error {
		alertRules, err := app.FindCollectionByNameOrId("alert_rules")
		if err != nil {
			return err
		}

		// Restore Required on metric-specific fields.
		alertRules.Fields.Add(
			&core.TextField{Name: "metric", Required: true, Max: 100},
			&core.SelectField{
				Name:      "operator",
				Values:    []string{"gt", "lt", "gte", "lte", "eq"},
				MaxSelect: 1,
				Required:  true,
			},
			&core.NumberField{Name: "threshold", Required: true},
			&core.SelectField{
				Name:      "target_type",
				Values:    []string{"server", "container"},
				MaxSelect: 1,
				Required:  true,
			},
		)

		return app.Save(alertRules)
	})
}
