package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

func init() {
	m.Register(migrateHostBackupsProxies, nil)
}

func migrateHostBackupsProxies(app core.App) error {
	if err := extendBackupPolicies(app); err != nil {
		return err
	}
	if err := backfillBackupPolicyTargetTypes(app); err != nil {
		return err
	}
	if err := extendBackupRecords(app); err != nil {
		return err
	}
	return createProxyCollections(app)
}

func extendBackupPolicies(app core.App) error {
	policies, err := app.FindCollectionByNameOrId(constants.CollBackupPolicies)
	if err != nil {
		return err
	}

	if field, ok := policies.Fields.GetByName("volume").(*core.TextField); ok {
		field.Required = false
	}

	policies.Fields.Add(
		&core.SelectField{
			Name:      "target_type",
			Values:    []string{"volume", "host_app"},
			MaxSelect: 1,
		},
		&core.SelectField{
			Name:      "app_source",
			Values:    []string{"systemd", "pm2", "process"},
			MaxSelect: 1,
		},
		&core.TextField{Name: "app_name", Max: 255},
		&core.JSONField{Name: "paths"},
		&core.JSONField{Name: "exclude"},
		&core.NumberField{Name: "max_bytes", OnlyInt: true},
	)
	policies.AddIndex("idx_backup_policies_target_type", false, "target_type", "")

	return app.Save(policies)
}

func backfillBackupPolicyTargetTypes(app core.App) error {
	policies, err := app.FindAllRecords(constants.CollBackupPolicies)
	if err != nil {
		return err
	}
	for _, policy := range policies {
		if policy.GetString("target_type") != "" {
			continue
		}
		policy.Set("target_type", "volume")
		if err := app.Save(policy); err != nil {
			return err
		}
	}
	return nil
}

func extendBackupRecords(app core.App) error {
	records, err := app.FindCollectionByNameOrId(constants.CollBackupRecords)
	if err != nil {
		return err
	}

	records.Fields.Add(
		&core.SelectField{
			Name:      "target_type",
			Values:    []string{"volume", "host_app"},
			MaxSelect: 1,
		},
		&core.SelectField{
			Name:      "app_source",
			Values:    []string{"systemd", "pm2", "process"},
			MaxSelect: 1,
		},
		&core.TextField{Name: "app_name", Max: 255},
		&core.TextField{Name: "target_label", Max: 512},
		&core.JSONField{Name: "manifest"},
	)
	records.AddIndex("idx_backup_records_target_type", false, "target_type", "")

	return app.Save(records)
}

func createProxyCollections(app core.App) error {
	servers, err := app.FindCollectionByNameOrId(constants.CollServers)
	if err != nil {
		return err
	}

	instances := core.NewBaseCollection(constants.CollProxyInstances)
	instances.Fields.Add(
		&core.RelationField{
			Name:          "server",
			CollectionId:  servers.Id,
			MaxSelect:     1,
			Required:      true,
			CascadeDelete: true,
		},
		&core.SelectField{
			Name:      "kind",
			Values:    []string{"traefik", "nginx", "caddy", "haproxy", "envoy"},
			MaxSelect: 1,
			Required:  true,
		},
		&core.SelectField{
			Name:      "source",
			Values:    []string{"systemd", "docker", "process"},
			MaxSelect: 1,
			Required:  true,
		},
		&core.TextField{Name: "app", Max: 255},
		&core.TextField{Name: "version", Max: 100},
		&core.TextField{Name: "config_path", Max: 1000},
		&core.TextField{Name: "admin_url", Max: 500},
		&core.SelectField{
			Name:      "status",
			Values:    []string{"healthy", "degraded", "unknown"},
			MaxSelect: 1,
		},
		&core.DateField{Name: "last_seen"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	instances.AddIndex("idx_proxy_instances_server", false, "server", "")
	instances.AddIndex("idx_proxy_instances_kind", false, "kind", "")
	instances.AddIndex("idx_proxy_instances_unique", true, "server, kind, source, app", "")
	if err := app.Save(instances); err != nil {
		return err
	}

	routes := core.NewBaseCollection(constants.CollProxyRoutes)
	routes.Fields.Add(
		&core.RelationField{
			Name:          "instance",
			CollectionId:  instances.Id,
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
		&core.TextField{Name: "name", Max: 500},
		&core.JSONField{Name: "hosts"},
		&core.JSONField{Name: "paths"},
		&core.JSONField{Name: "upstreams"},
		&core.BoolField{Name: "tls"},
		&core.JSONField{Name: "middlewares"},
		&core.SelectField{
			Name:      "status",
			Values:    []string{"healthy", "degraded", "unknown"},
			MaxSelect: 1,
		},
		&core.TextField{Name: "provider", Max: 100},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	routes.AddIndex("idx_proxy_routes_server", false, "server", "")
	routes.AddIndex("idx_proxy_routes_instance", false, "instance", "")

	return app.Save(routes)
}
