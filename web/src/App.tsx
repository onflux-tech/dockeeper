import { Routes, Route } from "react-router";
import ProtectedRoute from "@/components/layout/protected-route";
import AppLayout from "@/components/layout/app-layout";
import LoginPage from "@/pages/login";
import DashboardPage from "@/pages/dashboard";
import ServersPage from "@/pages/servers";
import ServerDetailPage from "@/pages/server-detail";
import ContainersPage from "@/pages/containers";
import ContainerCreatePage from "@/pages/container-create";
import ContainerDetailPage from "@/pages/container-detail";
import ContainerTerminalPage from "@/pages/container-terminal";
import ImagesPage from "@/pages/images";
import ComposePage from "@/pages/compose";
import NetworksPage from "@/pages/networks";
import VolumesPage from "@/pages/volumes";
import MetricsPage from "@/pages/metrics";
import AlertsPage from "@/pages/alerts";
import BackupsPage from "@/pages/backups";
import SettingsPage from "@/pages/settings";

function App() {
	return (
		<div className="dark min-h-screen bg-background text-foreground">
			<Routes>
				<Route path="/login" element={<LoginPage />} />
				<Route element={<ProtectedRoute />}>
					<Route element={<AppLayout />}>
						<Route index element={<DashboardPage />} />
						<Route path="servers" element={<ServersPage />} />
						<Route path="servers/:id" element={<ServerDetailPage />} />
						<Route path="containers" element={<ContainersPage />} />
						<Route path="containers/create" element={<ContainerCreatePage />} />
						<Route path="containers/:id" element={<ContainerDetailPage />} />
						<Route path="containers/:id/terminal" element={<ContainerTerminalPage />} />
						<Route path="images" element={<ImagesPage />} />
					<Route path="compose" element={<ComposePage />} />
						<Route path="networks" element={<NetworksPage />} />
						<Route path="volumes" element={<VolumesPage />} />
						<Route path="metrics" element={<MetricsPage />} />
						<Route path="alerts" element={<AlertsPage />} />
						<Route path="backups" element={<BackupsPage />} />
						<Route path="settings" element={<SettingsPage />} />
					</Route>
				</Route>
			</Routes>
		</div>
	);
}

export default App;
