import { Navigate, Outlet } from "react-router";
import { Loader2 } from "lucide-react";
import { useAuthStore } from "@/stores/auth-store";

export default function ProtectedRoute() {
	const { isAuthenticated, isLoading } = useAuthStore();

	if (isLoading) {
		return (
			<div className="flex min-h-screen items-center justify-center">
				<Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
			</div>
		);
	}

	if (!isAuthenticated) {
		return <Navigate to="/login" replace />;
	}

	return <Outlet />;
}
