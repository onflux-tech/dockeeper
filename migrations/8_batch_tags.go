package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// ── batch_jobs collection ───────────────────────────────
		batchJobs := core.NewBaseCollection("batch_jobs")
		batchJobs.Fields.Add(
			&core.TextField{Name: "action", Required: true, Max: 50},
			&core.JSONField{Name: "items", MaxSize: 5_000_000},
			&core.SelectField{
				Name:      "status",
				Values:    []string{"pending", "running", "completed", "cancelled"},
				MaxSelect: 1,
				Required:  true,
			},
			&core.NumberField{Name: "total"},
			&core.NumberField{Name: "succeeded"},
			&core.NumberField{Name: "failed"},
			&core.NumberField{Name: "cancelled"},
			&core.TextField{Name: "created_by", Max: 255},
			&core.DateField{Name: "started_at"},
			&core.DateField{Name: "completed_at"},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		batchJobs.AddIndex("idx_batch_jobs_status", false, "status", "")
		batchJobs.AddIndex("idx_batch_jobs_created_by", false, "created_by", "")
		if err := app.Save(batchJobs); err != nil {
			return err
		}

		// ── tags collection ────────────────────────────────────
		tags := core.NewBaseCollection("tags")
		tags.Fields.Add(
			&core.TextField{Name: "name", Required: true, Max: 255},
			&core.TextField{Name: "color", Max: 20},
			&core.SelectField{
				Name:      "resource_type",
				Values:    []string{"container", "server"},
				MaxSelect: 1,
				Required:  true,
			},
			&core.TextField{Name: "resource_id", Required: true, Max: 255},
			&core.AutodateField{Name: "created", OnCreate: true},
		)
		tags.AddIndex("idx_tags_name", false, "name", "")
		tags.AddIndex("idx_tags_resource", false, "resource_type, resource_id", "")
		tags.AddIndex("idx_tags_unique", true, "name, resource_type, resource_id", "")
		if err := app.Save(tags); err != nil {
			return err
		}

		// ── Add tags JSON field to containers ──────────────────
		containers, err := app.FindCollectionByNameOrId("containers")
		if err != nil {
			return err
		}
		containers.Fields.Add(&core.JSONField{Name: "tags", MaxSize: 50_000})
		if err := app.Save(containers); err != nil {
			return err
		}

		// ── Add tags JSON field to servers ─────────────────────
		servers, err := app.FindCollectionByNameOrId("servers")
		if err != nil {
			return err
		}
		servers.Fields.Add(&core.JSONField{Name: "tags", MaxSize: 50_000})
		if err := app.Save(servers); err != nil {
			return err
		}

		return nil
	}, nil)
}
