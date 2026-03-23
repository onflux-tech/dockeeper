import { Button } from "@/components/ui/button";
import { Container, LogOut } from "lucide-react";
import { useAuthStore } from "@/stores/auth-store";

export default function Dashboard() {
	const { admin, logout } = useAuthStore();

	return (
		<div className="flex min-h-screen flex-col">
			<header className="flex items-center justify-between border-b border-border px-6 py-3">
				<div className="flex items-center gap-2">
					<Container className="h-5 w-5 text-primary" />
					<span className="text-lg font-semibold">Dockeeper</span>
				</div>
				<div className="flex items-center gap-3">
					<span className="text-sm text-muted-foreground">{admin?.email}</span>
					<Button variant="ghost" size="icon-sm" onClick={() => void logout()}>
						<LogOut className="h-4 w-4" />
					</Button>
				</div>
			</header>
			<main className="flex flex-1 items-center justify-center">
				<p className="text-muted-foreground">Dashboard — coming soon</p>
			</main>
		</div>
	);
}
