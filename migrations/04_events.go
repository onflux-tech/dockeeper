package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		col := core.NewBaseCollection("events")
		col.Fields.Add(
			&core.TextField{Name: "event_type", Required: true, Max: 100},
			&core.SelectField{
				Name:      "severity",
				Values:    []string{"info", "warning", "critical"},
				MaxSelect: 1,
				Required:  true,
			},
			&core.TextField{Name: "server_id", Max: 50},
			&core.TextField{Name: "server_name", Max: 255},
			&core.TextField{Name: "container_id", Max: 50},
			&core.TextField{Name: "container_name", Max: 255},
			&core.JSONField{Name: "details"},
			&core.DateField{Name: "timestamp"},
			&core.AutodateField{Name: "created", OnCreate: true},
		)
		col.AddIndex("idx_events_type", false, "event_type", "")
		col.AddIndex("idx_events_severity", false, "severity", "")
		col.AddIndex("idx_events_server_id", false, "server_id", "")
		col.AddIndex("idx_events_timestamp", false, "timestamp", "")
		col.AddIndex("idx_events_type_severity", false, "event_type, severity", "")

		return app.Save(col)
	}, nil)
}
