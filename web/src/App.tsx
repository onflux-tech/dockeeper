import { Suspense, lazy } from "react";
import { Loader2 } from "lucide-react";
import { Routes, Route } from "react-router";
import { RequireCapability } from "@/components/layout/require-capability";

const ProtectedRoute = lazy(() => import("@/components/layout/protected-route"));
const AppLayout = lazy(() => import("@/components/layout/app-layout"));
const LoginPage = lazy(() => import("@/pages/login"));
const DashboardPage = lazy(() => import("@/pages/dashboard"));
const ServersPage = lazy(() => import("@/pages/servers"));
const ServerDetailPage = lazy(() => import("@/pages/server-detail"));
const ContainersPage = lazy(() => import("@/pages/containers"));
const ContainerCreatePage = lazy(() => import("@/pages/container-create"));
const ContainerDetailPage = lazy(() => import("@/pages/container-detail"));
const ContainerTerminalPage = lazy(() => import("@/pages/container-terminal"));
const ImagesPage = lazy(() => import("@/pages/images"));
const ComposePage = lazy(() => import("@/pages/compose"));
const ComposeCreatePage = lazy(() => import("@/pages/compose-create"));
const NetworksPage = lazy(() => import("@/pages/networks"));
const VolumesPage = lazy(() => import("@/pages/volumes"));
const AlertsPage = lazy(() => import("@/pages/alerts"));
const BackupsPage = lazy(() => import("@/pages/backups"));
const SettingsPage = lazy(() => import("@/pages/settings"));
const TraefikPage = lazy(() => import("@/pages/traefik"));
const ServerCreatePage = lazy(() => import("@/pages/server-create"));
const AuditLogsPage = lazy(() => import("@/pages/audit-logs"));
const UsersPage = lazy(() => import("@/pages/users"));
const EventsPage = lazy(() => import("@/pages/events"));
const HealthChecksPage = lazy(() => import("@/pages/health-checks"));
const DocsPage = lazy(() => import("@/pages/docs"));

function RouteFallback() {
	return (
		<div className="flex min-h-screen items-center justify-center">
			<Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
		</div>
	);
}

/** Helper to wrap a page with a capability check. */
function Cap({ cap, children }: { cap: string; children: React.ReactNode }) {
	return <RequireCapability capability={cap}>{children}</RequireCapability>;
}

function App() {
	return (
		<div className="min-h-screen bg-background text-foreground">
			<Suspense fallback={<RouteFallback />}>
				<Routes>
					<Route path="/login" element={<LoginPage />} />
					<Route path="/docs" element={<DocsPage />} />
					<Route element={<ProtectedRoute />}>
						<Route element={<AppLayout />}>
							<Route index element={<DashboardPage />} />
							<Route path="servers" element={<Cap cap="servers.view"><ServersPage /></Cap>} />
							<Route path="servers/create" element={<Cap cap="servers.manage"><ServerCreatePage /></Cap>} />
							<Route path="servers/:id" element={<Cap cap="servers.view"><ServerDetailPage /></Cap>} />
							<Route path="containers" element={<Cap cap="containers.view"><ContainersPage /></Cap>} />
							<Route path="containers/create" element={<Cap cap="containers.manage"><ContainerCreatePage /></Cap>} />
							<Route path="containers/:id" element={<Cap cap="containers.view"><ContainerDetailPage /></Cap>} />
							<Route path="containers/:id/terminal" element={<Cap cap="containers.manage"><ContainerTerminalPage /></Cap>} />
							<Route path="images" element={<Cap cap="images.view"><ImagesPage /></Cap>} />
							<Route path="compose" element={<Cap cap="compose.view"><ComposePage /></Cap>} />
							<Route path="compose/create" element={<Cap cap="compose.manage"><ComposeCreatePage /></Cap>} />
							<Route path="networks" element={<Cap cap="networks.manage"><NetworksPage /></Cap>} />
							<Route path="volumes" element={<Cap cap="volumes.manage"><VolumesPage /></Cap>} />
							<Route path="alerts" element={<Cap cap="alerts.view"><AlertsPage /></Cap>} />
							<Route path="health-checks" element={<Cap cap="health.view"><HealthChecksPage /></Cap>} />
							<Route path="backups" element={<Cap cap="backups.view"><BackupsPage /></Cap>} />
							<Route path="traefik" element={<Cap cap="traefik.view"><TraefikPage /></Cap>} />
							<Route path="settings" element={<Cap cap="settings.view"><SettingsPage /></Cap>} />
							<Route path="audit-logs" element={<Cap cap="audit.view"><AuditLogsPage /></Cap>} />
							<Route path="users" element={<Cap cap="users.view"><UsersPage /></Cap>} />
							<Route path="events" element={<Cap cap="events.view"><EventsPage /></Cap>} />
						</Route>
					</Route>
				</Routes>
			</Suspense>
		</div>
	);
}

export default App;
