package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("backup_policies")
		if err != nil {
			return nil // collection doesn't exist yet, 01_collections will create it correctly
		}

		// Remove the old relation field if it exists.
		if f := col.Fields.GetByName("volume"); f != nil {
			col.Fields.RemoveById(f.GetId())
		}

		// Add the volume field as a plain text field (stores the Docker volume name).
		col.Fields.Add(&core.TextField{Name: "volume", Required: true, Max: 255})

		return app.Save(col)
	}, func(app core.App) error {
		// Down: no-op — the field type change is non-reversible in a meaningful way.
		return nil
	})
}
