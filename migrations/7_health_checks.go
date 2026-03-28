package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// ── health_checks ────────────────────────────────────────
		hc := core.NewBaseCollection("health_checks")
		hc.Fields.Add(
			&core.TextField{Name: "name", Required: true, Max: 255},
			&core.RelationField{
				Name:          "container",
				CollectionId:  getCollectionID(app, "containers"),
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.RelationField{
				Name:          "server",
				CollectionId:  getCollectionID(app, "servers"),
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.TextField{Name: "url_path", Required: true, Max: 2048},
			&core.NumberField{Name: "port", Required: true},
			&core.NumberField{Name: "expected_status"},
			&core.NumberField{Name: "interval_seconds"},
			&core.NumberField{Name: "timeout_seconds"},
			&core.NumberField{Name: "failure_threshold"},
			&core.NumberField{Name: "success_threshold"},
			&core.BoolField{Name: "auto_restart"},
			&core.NumberField{Name: "max_restarts"},
			&core.NumberField{Name: "restart_window_minutes"},
			&core.BoolField{Name: "enabled"},
			&core.SelectField{
				Name:      "status",
				Values:    []string{"healthy", "unhealthy", "unknown"},
				MaxSelect: 1,
			},
			&core.NumberField{Name: "consecutive_failures"},
			&core.DateField{Name: "last_check_at"},
			&core.DateField{Name: "next_check_at"},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		hc.AddIndex("idx_health_checks_container", false, "container", "")
		hc.AddIndex("idx_health_checks_server", false, "server", "")
		hc.AddIndex("idx_health_checks_enabled", false, "enabled", "")
		hc.AddIndex("idx_health_checks_next_check_at", false, "next_check_at", "")

		if err := app.Save(hc); err != nil {
			return err
		}

		// ── health_check_logs ────────────────────────────────────
		hcl := core.NewBaseCollection("health_check_logs")
		hcl.Fields.Add(
			&core.RelationField{
				Name:          "health_check",
				CollectionId:  hc.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.NumberField{Name: "status_code"},
			&core.NumberField{Name: "response_time_ms"},
			&core.BoolField{Name: "success", Required: true},
			&core.TextField{Name: "error", Max: 4096},
			&core.DateField{Name: "checked_at", Required: true},
			&core.AutodateField{Name: "created", OnCreate: true},
		)
		hcl.AddIndex("idx_health_check_logs_check", false, "health_check", "")
		hcl.AddIndex("idx_health_check_logs_checked_at", false, "checked_at", "")

		return app.Save(hcl)
	}, nil)
}

func getCollectionID(app core.App, name string) string {
	col, err := app.FindCollectionByNameOrId(name)
	if err != nil {
		return ""
	}
	return col.Id
}
