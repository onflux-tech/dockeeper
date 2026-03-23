import { Routes, Route } from "react-router";
import ProtectedRoute from "@/components/layout/protected-route";
import LoginPage from "@/pages/login";
import Dashboard from "@/pages/dashboard";

function App() {
	return (
		<div className="dark min-h-screen bg-background text-foreground">
			<Routes>
				<Route path="/login" element={<LoginPage />} />
				<Route element={<ProtectedRoute />}>
					<Route path="/" element={<Dashboard />} />
				</Route>
			</Routes>
		</div>
	);
}

export default App;
