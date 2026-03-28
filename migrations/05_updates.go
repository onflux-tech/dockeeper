package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// ── 1. Add new fields to "containers" ───────────────
		containers, err := app.FindCollectionByNameOrId("containers")
		if err != nil {
			return err
		}
		containers.Fields.Add(
			&core.TextField{Name: "compose_project", Max: 255},
			&core.TextField{Name: "compose_service", Max: 255},
			&core.TextField{Name: "update_digest", Max: 255},
			&core.DateField{Name: "last_update_check"},
		)
		if err := app.Save(containers); err != nil {
			return err
		}

		// ── 2. Add new fields to "update_policies" ──────────
		updatePolicies, err := app.FindCollectionByNameOrId("update_policies")
		if err != nil {
			return err
		}
		updatePolicies.Fields.Add(
			&core.SelectField{
				Name:      "mode",
				Values:    []string{"monitor", "notify", "auto"},
				MaxSelect: 1,
				Required:  true,
			},
			&core.NumberField{Name: "max_concurrent", OnlyInt: true},
		)
		if err := app.Save(updatePolicies); err != nil {
			return err
		}

		// ── 3. Create "registry_credentials" ────────────────
		servers, err := app.FindCollectionByNameOrId("servers")
		if err != nil {
			return err
		}

		regCreds := core.NewBaseCollection("registry_credentials")
		regCreds.Fields.Add(
			&core.TextField{Name: "name", Required: true, Max: 255},
			&core.TextField{Name: "registry_host", Required: true, Max: 255},
			&core.TextField{Name: "config", Required: true}, // AES-256-GCM encrypted JSON
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		regCreds.AddIndex("idx_registry_credentials_host", true, "registry_host", "")
		if err := app.Save(regCreds); err != nil {
			return err
		}

		// ── 4. Create "update_history" ──────────────────────
		updateHistory := core.NewBaseCollection("update_history")
		updateHistory.Fields.Add(
			&core.RelationField{
				Name:          "container",
				CollectionId:  containers.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.RelationField{
				Name:          "server",
				CollectionId:  servers.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.TextField{Name: "old_image", Required: true, Max: 500},
			&core.TextField{Name: "new_image", Required: true, Max: 500},
			&core.SelectField{
				Name:      "status",
				Values:    []string{"pending", "pulling", "replacing", "verifying", "success", "failed", "rolledback"},
				MaxSelect: 1,
				Required:  true,
			},
			&core.DateField{Name: "started_at", Required: true},
			&core.DateField{Name: "completed_at"},
			&core.TextField{Name: "error"},
			&core.AutodateField{Name: "created", OnCreate: true},
		)
		updateHistory.AddIndex("idx_update_history_container", false, "container", "")
		updateHistory.AddIndex("idx_update_history_status", false, "status", "")
		updateHistory.AddIndex("idx_update_history_server", false, "server", "")
		if err := app.Save(updateHistory); err != nil {
			return err
		}

		return nil
	}, nil)
}
