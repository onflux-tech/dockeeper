package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(createHostInventory, nil)
}

func createHostInventory(app core.App) error {
	servers, err := app.FindCollectionByNameOrId("servers")
	if err != nil {
		return err
	}

	hostApps := core.NewBaseCollection("host_apps")
	hostApps.Fields.Add(
		&core.RelationField{
			Name:          "server",
			CollectionId:  servers.Id,
			MaxSelect:     1,
			Required:      true,
			CascadeDelete: true,
		},
		&core.SelectField{
			Name:      "source",
			Values:    []string{"systemd", "pm2", "process"},
			MaxSelect: 1,
		},
		&core.TextField{Name: "name", Required: true, Max: 255},
		&core.TextField{Name: "unit", Max: 255},
		&core.TextField{Name: "pm_id", Max: 50},
		&core.SelectField{
			Name:      "status",
			Values:    []string{"running", "stopped", "failed", "unknown"},
			MaxSelect: 1,
		},
		&core.JSONField{Name: "pids"},
		&core.JSONField{Name: "ports"},
		&core.NumberField{Name: "cpu_percent"},
		&core.NumberField{Name: "mem_bytes"},
		&core.NumberField{Name: "disk_read_bytes"},
		&core.NumberField{Name: "disk_write_bytes"},
		&core.NumberField{Name: "restarts", OnlyInt: true},
		&core.NumberField{Name: "uptime_seconds"},
		&core.TextField{Name: "user", Max: 255},
		&core.TextField{Name: "cwd", Max: textFieldMax},
		&core.TextField{Name: "exec_path", Max: textFieldMax},
		&core.TextField{Name: "log_path", Max: textFieldMax},
		&core.DateField{Name: "last_seen"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	hostApps.AddIndex("idx_host_apps_server", false, "server", "")
	hostApps.AddIndex("idx_host_apps_server_source_name", true, "server, source, name", "")
	if err := app.Save(hostApps); err != nil {
		return err
	}

	hostProcesses := core.NewBaseCollection("host_processes")
	hostProcesses.Fields.Add(
		&core.RelationField{
			Name:          "server",
			CollectionId:  servers.Id,
			MaxSelect:     1,
			Required:      true,
			CascadeDelete: true,
		},
		&core.NumberField{Name: "pid", Required: true, OnlyInt: true},
		&core.TextField{Name: "name", Max: 255},
		&core.TextField{Name: "app", Max: 255},
		&core.NumberField{Name: "cpu_percent"},
		&core.NumberField{Name: "mem_bytes"},
		&core.NumberField{Name: "threads", OnlyInt: true},
		&core.DateField{Name: "last_seen"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	hostProcesses.AddIndex("idx_host_processes_server", false, "server", "")
	hostProcesses.AddIndex("idx_host_processes_server_pid", true, "server, pid", "")
	if err := app.Save(hostProcesses); err != nil {
		return err
	}

	hostAppMetricsRaw := core.NewBaseCollection("host_app_metrics_raw")
	hostAppMetricsRaw.Fields.Add(
		&core.RelationField{
			Name:          "server",
			CollectionId:  servers.Id,
			MaxSelect:     1,
			Required:      true,
			CascadeDelete: true,
		},
		&core.TextField{Name: "app", Max: 255},
		&core.TextField{Name: "source", Max: 50},
		&core.DateField{Name: "timestamp", Required: true},
		&core.NumberField{Name: "cpu_percent"},
		&core.NumberField{Name: "mem_bytes"},
		&core.NumberField{Name: "restarts", OnlyInt: true},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	hostAppMetricsRaw.AddIndex("idx_host_app_metrics_raw_server", false, "server", "")
	hostAppMetricsRaw.AddIndex("idx_host_app_metrics_raw_server_ts", false, "server, timestamp", "")
	hostAppMetricsRaw.AddIndex("idx_host_app_metrics_raw_app_ts", false, "app, timestamp", "")
	if err := app.Save(hostAppMetricsRaw); err != nil {
		return err
	}

	orchestrator, ok := servers.Fields.GetByName("orchestrator").(*core.SelectField)
	if ok && !containsString(orchestrator.Values, "none") {
		orchestrator.Values = append(orchestrator.Values, "none")
	}

	return app.Save(servers)
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
