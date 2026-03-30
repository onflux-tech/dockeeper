package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Create dk_users auth collection for regular (non-superuser) users.
		col := core.NewAuthCollection("dk_users")
		col.Fields.Add(
			&core.TextField{Name: "name", Max: 255},
			&core.BoolField{Name: "enabled"},
			&core.AutodateField{Name: "created", OnCreate: true, System: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true, System: true},
		)
		col.AddIndex("idx_dk_users_email", true, "email", "")
		col.AddIndex("idx_dk_users_enabled", false, "enabled", "")

		// Set auth options.
		col.AuthToken.Duration = 900 // 15 minutes

		return app.Save(col)
	}, func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("dk_users")
		if err != nil {
			return nil
		}
		return app.Delete(col)
	})
}
