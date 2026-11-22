package routes

import (
	"net/http"
	"strconv"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/manager/services"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// RegisterHealthCheckRoutes adds CRUD and trigger endpoints for health checks.
func RegisterHealthCheckRoutes(se *core.ServeEvent, app core.App, healthStore *store.HealthStore, scheduler *services.HealthScheduler, rbac *middleware.RBAC) {
	g := se.Router.Group("/api/dk/health-checks")

	// GET /api/dk/health-checks — list all checks.
	g.GET("", func(e *core.RequestEvent) error {
		serverID := e.Request.URL.Query().Get("server_id")
		records, err := healthStore.GetAllChecks(serverID)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return e.JSON(http.StatusOK, records)
	}).BindFunc(rbac.RequireCapability(constants.CapHealthView))

	// POST /api/dk/health-checks — create a new check.
	g.POST("", func(e *core.RequestEvent) error {
		var body struct {
			Name                 string            `json:"name"`
			Container            string            `json:"container"`
			Server               string            `json:"server"`
			TargetType           string            `json:"target_type"`
			AppSource            string            `json:"app_source"`
			AppName              string            `json:"app_name"`
			CheckType            string            `json:"check_type"`
			Host                 string            `json:"host"`
			Scheme               string            `json:"scheme"`
			Method               string            `json:"method"`
			URLPath              string            `json:"url_path"`
			Port                 int               `json:"port"`
			ExpectedStatus       int               `json:"expected_status"`
			ExpectedBody         string            `json:"expected_body"`
			Headers              map[string]string `json:"headers"`
			TLSSkipVerify        bool              `json:"tls_skip_verify"`
			IntervalSeconds      int               `json:"interval_seconds"`
			TimeoutSeconds       int               `json:"timeout_seconds"`
			FailureThreshold     int               `json:"failure_threshold"`
			SuccessThreshold     int               `json:"success_threshold"`
			AutoRestart          bool              `json:"auto_restart"`
			MaxRestarts          int               `json:"max_restarts"`
			RestartWindowMinutes int               `json:"restart_window_minutes"`
			Enabled              bool              `json:"enabled"`
		}
		if err := e.BindBody(&body); err != nil || body.Name == "" || body.Server == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "name and server are required"})
		}

		target := healthCheckTarget{
			targetType:  body.TargetType,
			container:   body.Container,
			appSource:   body.AppSource,
			appName:     body.AppName,
			checkType:   body.CheckType,
			scheme:      body.Scheme,
			host:        body.Host,
			port:        body.Port,
			autoRestart: body.AutoRestart,
		}
		if message := target.validate(); message != "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": message})
		}

		fields := map[string]any{
			"name":                   body.Name,
			"container":              body.Container,
			"server":                 body.Server,
			"target_type":            target.normalizedTargetType(),
			"app_source":             body.AppSource,
			"app_name":               body.AppName,
			"check_type":             target.normalizedCheckType(),
			"host":                   body.Host,
			"scheme":                 body.Scheme,
			"method":                 body.Method,
			"url_path":               body.URLPath,
			"port":                   body.Port,
			"expected_status":        body.ExpectedStatus,
			"expected_body":          body.ExpectedBody,
			"headers":                body.Headers,
			"tls_skip_verify":        body.TLSSkipVerify,
			"interval_seconds":       body.IntervalSeconds,
			"timeout_seconds":        body.TimeoutSeconds,
			"failure_threshold":      body.FailureThreshold,
			"success_threshold":      body.SuccessThreshold,
			"auto_restart":           body.AutoRestart,
			"max_restarts":           body.MaxRestarts,
			"restart_window_minutes": body.RestartWindowMinutes,
			"enabled":                body.Enabled,
		}

		id, err := healthStore.CreateCheck(fields)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		if body.Enabled {
			scheduler.AddCheck(id)
		}

		return e.JSON(http.StatusCreated, map[string]string{"id": id})
	}).BindFunc(rbac.RequireCapability(constants.CapHealthManage))

	// PUT /api/dk/health-checks/{id} — update a check.
	g.PUT("/{id}", func(e *core.RequestEvent) error {
		checkID := e.Request.PathValue("id")
		var body map[string]any
		if err := e.BindBody(&body); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
		}

		allowed := map[string]bool{
			"name": true, "url_path": true, "port": true,
			"target_type": true, "app_source": true, "app_name": true,
			"check_type": true, "host": true, "scheme": true, "method": true,
			"expected_status": true, "expected_body": true, "headers": true, "tls_skip_verify": true,
			"interval_seconds": true, "timeout_seconds": true,
			"failure_threshold": true, "success_threshold": true,
			"auto_restart": true, "max_restarts": true, "restart_window_minutes": true,
			"enabled": true,
		}
		fields := make(map[string]any)
		for k, v := range body {
			if allowed[k] {
				fields[k] = v
			}
		}

		existing, err := healthStore.GetCheckByID(checkID)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "health check not found"})
		}

		target := healthCheckTarget{
			targetType:  existing.GetString("target_type"),
			container:   existing.GetString("container"),
			appSource:   existing.GetString("app_source"),
			appName:     existing.GetString("app_name"),
			checkType:   existing.GetString("check_type"),
			scheme:      existing.GetString("scheme"),
			host:        existing.GetString("host"),
			port:        existing.GetInt("port"),
			autoRestart: existing.GetBool("auto_restart"),
		}
		if v, ok := fields["target_type"].(string); ok {
			target.targetType = v
		}
		if v, ok := fields["container"].(string); ok {
			target.container = v
		}
		if v, ok := fields["app_source"].(string); ok {
			target.appSource = v
		}
		if v, ok := fields["app_name"].(string); ok {
			target.appName = v
		}
		if v, ok := fields["check_type"].(string); ok {
			target.checkType = v
		}
		if v, ok := fields["scheme"].(string); ok {
			target.scheme = v
		}
		if v, ok := fields["host"].(string); ok {
			target.host = v
		}
		if v, ok := fields["port"].(float64); ok {
			target.port = int(v)
		}
		if v, ok := fields["auto_restart"].(bool); ok {
			target.autoRestart = v
		}
		if message := target.validate(); message != "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": message})
		}

		if err := healthStore.UpdateCheck(checkID, fields); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		// Reload scheduler to pick up changes.
		scheduler.RemoveCheck(checkID)
		if enabled, ok := fields["enabled"].(bool); ok && enabled {
			scheduler.AddCheck(checkID)
		} else if !ok {
			// If enabled wasn't changed, check current state.
			rec, err := healthStore.GetCheckByID(checkID)
			if err == nil && rec.GetBool("enabled") {
				scheduler.AddCheck(checkID)
			}
		}

		return e.JSON(http.StatusOK, map[string]string{"status": "updated"})
	}).BindFunc(rbac.RequireCapability(constants.CapHealthManage))

	// DELETE /api/dk/health-checks/{id} — delete a check.
	g.DELETE("/{id}", func(e *core.RequestEvent) error {
		checkID := e.Request.PathValue("id")
		scheduler.RemoveCheck(checkID)
		if err := healthStore.DeleteCheck(checkID); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return e.JSON(http.StatusOK, map[string]string{"status": "deleted"})
	}).BindFunc(rbac.RequireCapability(constants.CapHealthManage))

	// GET /api/dk/health-checks/{id}/logs — get check logs.
	g.GET("/{id}/logs", func(e *core.RequestEvent) error {
		checkID := e.Request.PathValue("id")
		page, _ := strconv.Atoi(e.Request.URL.Query().Get("page"))
		perPage, _ := strconv.Atoi(e.Request.URL.Query().Get("per_page"))
		if page < 1 {
			page = 1
		}
		if perPage < 1 {
			perPage = 50
		}

		records, total, err := healthStore.GetCheckLogs(checkID, page, perPage)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusOK, map[string]any{
			"items":       records,
			"total_items": total,
			"page":        page,
			"per_page":    perPage,
		})
	}).BindFunc(rbac.RequireCapability(constants.CapHealthView))

	// POST /api/dk/health-checks/{id}/trigger — trigger immediate check.
	g.POST("/{id}/trigger", func(e *core.RequestEvent) error {
		checkID := e.Request.PathValue("id")
		scheduler.TriggerCheck(checkID)
		return e.JSON(http.StatusOK, map[string]string{"status": "triggered"})
	}).BindFunc(rbac.RequireCapability(constants.CapHealthManage))
}

type healthCheckTarget struct {
	targetType  string
	container   string
	appSource   string
	appName     string
	checkType   string
	scheme      string
	host        string
	port        int
	autoRestart bool
}

func (t healthCheckTarget) normalizedTargetType() string {
	if t.targetType == "" {
		return "container"
	}
	return t.targetType
}

func (t healthCheckTarget) normalizedCheckType() string {
	if t.checkType == "" {
		return "http"
	}
	return t.checkType
}

func (t healthCheckTarget) validate() string {
	if t.autoRestart && t.normalizedTargetType() == "host_app" {
		return "auto_restart is not supported for host apps"
	}

	switch t.normalizedTargetType() {
	case "container":
		if t.container == "" {
			return "container is required"
		}
	case "host_app":
		if !validAppSource(t.appSource) {
			return "app_source is required"
		}
		if t.appName == "" {
			return "app_name is required"
		}
	default:
		return "invalid target_type"
	}

	switch t.normalizedCheckType() {
	case "http", "tcp", "process":
	default:
		return "invalid check_type"
	}

	if t.normalizedCheckType() != "process" && (t.port <= 0 || t.port > 65535) {
		return "port is required for http and tcp checks"
	}
	if t.scheme != "" && t.scheme != "http" && t.scheme != "https" {
		return "invalid scheme"
	}
	if t.host != "" && !validHealthCheckHost(t.host) {
		return "invalid host"
	}

	return ""
}

func validAppSource(source string) bool {
	return source == "systemd" || source == "pm2" || source == "process"
}

func validHealthCheckHost(host string) bool {
	for _, char := range host {
		switch {
		case char >= 'a' && char <= 'z', char >= 'A' && char <= 'Z', char >= '0' && char <= '9',
			char == '.', char == '-', char == '_', char == ':':
		default:
			return false
		}
	}
	return true
}
