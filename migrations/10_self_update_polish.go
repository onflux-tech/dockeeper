package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// ── action_tokens collection ──────────────────────────────────────
		col := core.NewBaseCollection("action_tokens")
		col.Fields.Add(
			&core.TextField{Name: "token", Required: true, Max: 64},
			&core.TextField{Name: "action", Required: true, Max: 64},
			&core.JSONField{Name: "params", Required: true},
			&core.DateField{Name: "expires_at", Required: true},
			&core.BoolField{Name: "used"},
			&core.DateField{Name: "used_at"},
			&core.TextField{Name: "created_by", Max: 255},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		col.AddIndex("idx_action_tokens_token", true, "token", "")
		col.AddIndex("idx_action_tokens_expires", false, "expires_at", "")
		if err := app.Save(col); err != nil {
			return err
		}

		// ── Add self-update settings ─────────────────────────────────────
		settingsCol, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}

		defaults := []struct {
			key, value string
		}{
			{"auto_update_check", "true"},
			{"update_check_interval", "86400"},
		}

		for _, d := range defaults {
			rec := core.NewRecord(settingsCol)
			rec.Set("key", d.key)
			rec.Set("value", d.value)
			if err := app.Save(rec); err != nil {
				return err
			}
		}

		return nil
	}, nil)
}
