package migrations

import (
	"log/slog"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// ── 1. Add new fields to alert_rules ──────────────────
		alertRules, err := app.FindCollectionByNameOrId("alert_rules")
		if err != nil {
			return err
		}
		alertRules.Fields.Add(
			&core.SelectField{
				Name:      "rule_type",
				Values:    []string{"metric", "event"},
				MaxSelect: 1,
				Required:  true,
			},
			&core.TextField{Name: "event_type", Max: 100},
			&core.TextField{Name: "scope", Max: 255},
			&core.NumberField{Name: "active_delay", OnlyInt: true},
			&core.NumberField{Name: "clear_delay", OnlyInt: true},
			&core.NumberField{Name: "notification_cooldown", OnlyInt: true},
			&core.BoolField{Name: "suppress_during_updates"},
			&core.TextField{Name: "template"},
			&core.JSONField{Name: "annotations"},
		)
		if err := app.Save(alertRules); err != nil {
			return err
		}

		// Set defaults for existing rules.
		existingRules, err := app.FindRecordsByFilter("alert_rules", "", "", 0, 0)
		if err == nil {
			for _, r := range existingRules {
				r.Set("rule_type", "metric")
				if r.GetInt("notification_cooldown") == 0 {
					r.Set("notification_cooldown", 300)
				}
				if r.GetString("scope") == "" {
					r.Set("scope", "all")
				}
				if err := app.Save(r); err != nil {
					slog.Warn("failed to set defaults on existing alert rule", "id", r.Id, "error", err)
				}
			}
		}

		// ── 2. Expand notification_channels type field ─────────
		channels, err := app.FindCollectionByNameOrId("notification_channels")
		if err != nil {
			return err
		}
		// Replace the type field with expanded values.
		channels.Fields.Add(
			&core.SelectField{
				Name:      "type",
				Values:    []string{"whatsapp", "telegram", "webhook", "discord", "slack", "smtp", "ntfy"},
				MaxSelect: 1,
				Required:  true,
			},
		)
		if err := app.Save(channels); err != nil {
			return err
		}

		// ── 3. Create blackout_windows collection ──────────────
		blackoutWindows := core.NewBaseCollection("blackout_windows")
		blackoutWindows.Fields.Add(
			&core.TextField{Name: "name", Required: true, Max: 255},
			&core.DateField{Name: "start_time", Required: true},
			&core.DateField{Name: "end_time", Required: true},
			&core.BoolField{Name: "recurring"},
			&core.TextField{Name: "recurrence_rule", Max: 100},
			&core.NumberField{Name: "duration_minutes", OnlyInt: true},
			&core.JSONField{Name: "server_ids"},
			&core.BoolField{Name: "enabled"},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		blackoutWindows.AddIndex("idx_blackout_windows_enabled", false, "enabled", "")
		if err := app.Save(blackoutWindows); err != nil {
			return err
		}

		// ── 4. Seed retention settings ─────────────────────────
		settingsCol, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}

		retentionDefaults := map[string]string{
			"retention_raw_hours":             "168",  // 7 days
			"retention_hourly_days":           "30",
			"retention_daily_days":            "365",
			"retention_events_days":           "30",
			"retention_audit_logs_days":       "90",
			"retention_notification_logs_days": "30",
		}

		for key, val := range retentionDefaults {
			existing, _ := app.FindFirstRecordByFilter("settings", "key = {:key}", dbx.Params{"key": key})
			if existing != nil {
				continue
			}
			rec := core.NewRecord(settingsCol)
			rec.Set("key", key)
			rec.Set("value", val)
			if err := app.Save(rec); err != nil {
				slog.Warn("failed to seed retention setting", "key", key, "error", err)
			}
		}

		return nil
	}, func(app core.App) error {
		// Down migration.
		if col, err := app.FindCollectionByNameOrId("blackout_windows"); err == nil {
			if err := app.Delete(col); err != nil {
				return err
			}
		}

		// Remove retention settings.
		retentionKeys := []string{
			"retention_raw_hours", "retention_hourly_days", "retention_daily_days",
			"retention_events_days", "retention_audit_logs_days", "retention_notification_logs_days",
		}
		for _, key := range retentionKeys {
			rec, _ := app.FindFirstRecordByFilter("settings", "key = {:key}", dbx.Params{"key": key})
			if rec != nil {
				_ = app.Delete(rec)
			}
		}

		return nil
	})
}
