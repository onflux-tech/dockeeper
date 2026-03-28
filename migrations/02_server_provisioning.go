package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		servers, err := app.FindCollectionByNameOrId("servers")
		if err != nil {
			return err
		}

		// Add enrollment_token field for per-server provisioning.
		servers.Fields.Add(&core.TextField{Name: "enrollment_token", Max: 255})

		// Add description field.
		servers.Fields.Add(&core.TextField{Name: "description", Max: 1000})

		// Update status field to include "pending".
		for i, f := range servers.Fields {
			if sf, ok := f.(*core.SelectField); ok && sf.Name == "status" {
				sf.Values = []string{"online", "offline", "error", "pending"}
				servers.Fields[i] = sf
				break
			}
		}

		servers.AddIndex("idx_servers_enrollment_token", true, "enrollment_token", "enrollment_token != ''")

		return app.Save(servers)
	}, func(app core.App) error {
		servers, err := app.FindCollectionByNameOrId("servers")
		if err != nil {
			return err
		}

		servers.Fields.RemoveByName("enrollment_token")
		servers.Fields.RemoveByName("description")

		for i, f := range servers.Fields {
			if sf, ok := f.(*core.SelectField); ok && sf.Name == "status" {
				sf.Values = []string{"online", "offline", "error"}
				servers.Fields[i] = sf
				break
			}
		}

		return app.Save(servers)
	})
}
