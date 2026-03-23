import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router";
import { QueryClientProvider } from "@tanstack/react-query";
import { TooltipProvider } from "@/components/ui/tooltip";
import { useAuthStore } from "@/stores/auth-store";
import queryClient from "@/lib/query-client";
import App from "./App";
import "./index.css";

// Initialize auth state on app load.
useAuthStore.getState().checkAuth();

createRoot(document.getElementById("root")!).render(
	<StrictMode>
		<QueryClientProvider client={queryClient}>
			<BrowserRouter>
				<TooltipProvider>
					<App />
				</TooltipProvider>
			</BrowserRouter>
		</QueryClientProvider>
	</StrictMode>,
);
