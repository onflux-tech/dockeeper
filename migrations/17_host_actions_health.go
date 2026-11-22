package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

func init() {
	m.Register(migrateHostActionsHealth, nil)
}

func migrateHostActionsHealth(app core.App) error {
	if err := extendHealthChecks(app); err != nil {
		return err
	}
	if err := backfillHealthCheckTargetTypes(app); err != nil {
		return err
	}
	if err := createServerConnectivityRaw(app); err != nil {
		return err
	}
	return extendServersAndAlertRules(app)
}

func extendHealthChecks(app core.App) error {
	healthChecks, err := app.FindCollectionByNameOrId(constants.CollHealthChecks)
	if err != nil {
		return err
	}

	if field, ok := healthChecks.Fields.GetByName("container").(*core.RelationField); ok {
		field.Required = false
	}
	if field, ok := healthChecks.Fields.GetByName("url_path").(*core.TextField); ok {
		field.Required = false
	}
	if field, ok := healthChecks.Fields.GetByName("port").(*core.NumberField); ok {
		field.Required = false
	}

	healthChecks.Fields.Add(
		&core.SelectField{
			Name:      "target_type",
			Values:    []string{"container", "host_app"},
			MaxSelect: 1,
		},
		&core.SelectField{
			Name:      "app_source",
			Values:    []string{"systemd", "pm2", "process"},
			MaxSelect: 1,
		},
		&core.TextField{Name: "app_name", Max: 255},
		&core.SelectField{
			Name:      "check_type",
			Values:    []string{"http", "tcp", "process"},
			MaxSelect: 1,
		},
		&core.TextField{Name: "host", Max: 255},
		&core.SelectField{
			Name:      "scheme",
			Values:    []string{"http", "https"},
			MaxSelect: 1,
		},
		&core.TextField{Name: "method", Max: 10},
		&core.TextField{Name: "expected_body", Max: textFieldMax},
		&core.JSONField{Name: "headers"},
		&core.BoolField{Name: "tls_skip_verify"},
		&core.NumberField{Name: "consecutive_successes", OnlyInt: true},
	)
	healthChecks.AddIndex("idx_health_checks_target_type", false, "target_type", "")
	healthChecks.AddIndex("idx_health_checks_server_app", false, "server, app_source, app_name", "")

	return app.Save(healthChecks)
}

func backfillHealthCheckTargetTypes(app core.App) error {
	records, err := app.FindAllRecords(constants.CollHealthChecks)
	if err != nil {
		return err
	}
	for _, record := range records {
		if record.GetString("target_type") != "" {
			continue
		}
		record.Set("target_type", "container")
		if record.GetString("check_type") == "" {
			record.Set("check_type", "http")
		}
		if err := app.Save(record); err != nil {
			return err
		}
	}
	return nil
}

func createServerConnectivityRaw(app core.App) error {
	servers, err := app.FindCollectionByNameOrId(constants.CollServers)
	if err != nil {
		return err
	}

	raw := core.NewBaseCollection(constants.CollServerConnectivityRaw)
	raw.Fields.Add(
		&core.RelationField{
			Name:          "server",
			CollectionId:  servers.Id,
			MaxSelect:     1,
			Required:      true,
			CascadeDelete: true,
		},
		&core.DateField{Name: "timestamp", Required: true},
		&core.BoolField{Name: "internet_ok"},
		&core.NumberField{Name: "internet_ms", OnlyInt: true},
		&core.BoolField{Name: "dns_ok"},
		&core.NumberField{Name: "dns_ms", OnlyInt: true},
		&core.BoolField{Name: "gateway_ok"},
		&core.NumberField{Name: "gateway_ms", OnlyInt: true},
		&core.TextField{Name: "gateway_method", Max: 20},
		&core.TextField{Name: "error", Max: 4096},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	raw.AddIndex("idx_server_connectivity_raw_server", false, "server", "")
	raw.AddIndex("idx_server_connectivity_raw_server_ts", false, "server, timestamp", "")

	return app.Save(raw)
}

func extendServersAndAlertRules(app core.App) error {
	servers, err := app.FindCollectionByNameOrId(constants.CollServers)
	if err != nil {
		return err
	}
	servers.Fields.Add(
		&core.SelectField{
			Name:      "connectivity_status",
			Values:    []string{"healthy", "degraded", "unknown"},
			MaxSelect: 1,
		},
		&core.DateField{Name: "connectivity_checked_at"},
		&core.NumberField{Name: "connectivity_latency_ms", OnlyInt: true},
		&core.JSONField{Name: "connectivity_details"},
		&core.SelectField{
			Name:      "view_scope",
			Values:    []string{"auto", "server", "docker", "both"},
			MaxSelect: 1,
		},
	)
	if err := app.Save(servers); err != nil {
		return err
	}

	existingServers, err := app.FindAllRecords(constants.CollServers)
	if err != nil {
		return err
	}
	for _, server := range existingServers {
		if server.GetString("view_scope") != "" {
			continue
		}
		server.Set("view_scope", "auto")
		if err := app.Save(server); err != nil {
			return err
		}
	}

	alertRules, err := app.FindCollectionByNameOrId(constants.CollAlertRules)
	if err != nil {
		return err
	}
	if field, ok := alertRules.Fields.GetByName("target_type").(*core.SelectField); ok && !containsString(field.Values, "host_app") {
		field.Values = append(field.Values, "host_app")
	}
	return app.Save(alertRules)
}
