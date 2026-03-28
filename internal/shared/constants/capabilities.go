package constants

// Capability domain constants for RBAC middleware.
const (
	CapServersView         = "servers.view"
	CapServersManage       = "servers.manage"
	CapContainersView      = "containers.view"
	CapContainersManage    = "containers.manage"
	CapImagesView          = "images.view"
	CapImagesManage        = "images.manage"
	CapComposeView         = "compose.view"
	CapComposeManage       = "compose.manage"
	CapNetworksManage      = "networks.manage"
	CapVolumesManage       = "volumes.manage"
	CapBackupsView         = "backups.view"
	CapBackupsManage       = "backups.manage"
	CapAlertsView          = "alerts.view"
	CapAlertsManage        = "alerts.manage"
	CapNotificationsView   = "notifications.view"
	CapNotificationsManage = "notifications.manage"
	CapSettingsView        = "settings.view"
	CapSettingsManage      = "settings.manage"
	CapUsersView           = "users.view"
	CapUsersManage         = "users.manage"
	CapAuditView           = "audit.view"
	CapTraefikView         = "traefik.view"
	CapEventsView          = "events.view"
	CapHealthView          = "health.view"
	CapHealthManage        = "health.manage"
	CapBatchView           = "batch.view"
	CapBatchManage         = "batch.manage"
	CapTagsView            = "tags.view"
	CapTagsManage          = "tags.manage"
)

// AllCapabilities is the complete set of capabilities for Sprint 13.
var AllCapabilities = []string{
	CapServersView, CapServersManage,
	CapContainersView, CapContainersManage,
	CapImagesView, CapImagesManage,
	CapComposeView, CapComposeManage,
	CapNetworksManage, CapVolumesManage,
	CapBackupsView, CapBackupsManage,
	CapAlertsView, CapAlertsManage,
	CapNotificationsView, CapNotificationsManage,
	CapSettingsView, CapSettingsManage,
	CapUsersView, CapUsersManage,
	CapAuditView, CapTraefikView,
	CapEventsView,
	CapHealthView, CapHealthManage,
	CapBatchView, CapBatchManage,
	CapTagsView, CapTagsManage,
}
