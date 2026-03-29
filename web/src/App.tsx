import { Suspense, lazy } from "react";
import { Loader2 } from "lucide-react";
import { Routes, Route } from "react-router";

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
							<Route path="servers" element={<ServersPage />} />
							<Route path="servers/create" element={<ServerCreatePage />} />
							<Route path="servers/:id" element={<ServerDetailPage />} />
							<Route path="containers" element={<ContainersPage />} />
							<Route path="containers/create" element={<ContainerCreatePage />} />
							<Route path="containers/:id" element={<ContainerDetailPage />} />
							<Route path="containers/:id/terminal" element={<ContainerTerminalPage />} />
							<Route path="images" element={<ImagesPage />} />
							<Route path="compose" element={<ComposePage />} />
							<Route path="compose/create" element={<ComposeCreatePage />} />
							<Route path="networks" element={<NetworksPage />} />
							<Route path="volumes" element={<VolumesPage />} />
							<Route path="alerts" element={<AlertsPage />} />
							<Route path="health-checks" element={<HealthChecksPage />} />
							<Route path="backups" element={<BackupsPage />} />
							<Route path="traefik" element={<TraefikPage />} />
							<Route path="settings" element={<SettingsPage />} />
							<Route path="audit-logs" element={<AuditLogsPage />} />
							<Route path="users" element={<UsersPage />} />
							<Route path="events" element={<EventsPage />} />
						</Route>
					</Route>
				</Routes>
			</Suspense>
		</div>
	);
}

export default App;
