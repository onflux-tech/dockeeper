package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// ── Add hardware + live metrics fields to servers ─────────────
		servers, err := app.FindCollectionByNameOrId("servers")
		if err != nil {
			return err
		}
		servers.Fields.Add(
			&core.TextField{Name: "cpu_model", Max: 255},
			&core.NumberField{Name: "cpu_cores"},
			&core.NumberField{Name: "mem_total"},
			&core.NumberField{Name: "current_cpu"},
			&core.NumberField{Name: "current_ram"},
			&core.NumberField{Name: "current_disk"},
			&core.NumberField{Name: "uptime_seconds"},
		)
		if err := app.Save(servers); err != nil {
			return err
		}

		// ── Add live metrics fields to containers ────────────────────
		containers, err := app.FindCollectionByNameOrId("containers")
		if err != nil {
			return err
		}
		containers.Fields.Add(
			&core.NumberField{Name: "current_cpu"},
			&core.NumberField{Name: "current_ram"},
			&core.NumberField{Name: "current_ram_limit"},
		)
		if err := app.Save(containers); err != nil {
			return err
		}

		// ── Add enhanced fields to server_metrics_raw ────────────────
		raw, err := app.FindCollectionByNameOrId("server_metrics_raw")
		if err != nil {
			return err
		}
		raw.Fields.Add(
			&core.NumberField{Name: "disk_io_read"},
			&core.NumberField{Name: "disk_io_write"},
			&core.NumberField{Name: "swap_used"},
			&core.NumberField{Name: "swap_total"},
			&core.NumberField{Name: "mem_available"},
			&core.JSONField{Name: "partitions"},
			&core.JSONField{Name: "temperatures"},
			&core.NumberField{Name: "cpu_cores"},
			&core.NumberField{Name: "uptime_seconds"},
		)
		if err := app.Save(raw); err != nil {
			return err
		}

		return nil
	}, nil)
}
