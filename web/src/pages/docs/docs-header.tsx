import { useState } from "react";
import { Link } from "react-router";
import { ArrowLeft, LogIn, Menu } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { DockeeperLogo } from "@/components/dockeeper-logo";
import { useAuthStore } from "@/stores/auth-store";
import { latestRelease } from "@/data/changelog";
import { AuthorizeBadge, AuthorizeDialog } from "./playground-auth";

interface DocsHeaderProps {
	onMobileToggle: () => void;
	onChangelogOpen: () => void;
	historySlot?: React.ReactNode;
}

const latestVersion = latestRelease?.version ?? "0.0.0";

export function DocsHeader({ onMobileToggle, onChangelogOpen, historySlot }: DocsHeaderProps) {
	const [authOpen, setAuthOpen] = useState(false);
	const isAuthenticated = useAuthStore((s) => s.isAuthenticated);

	return (
		<header className="docs-header sticky top-0 z-50 flex h-14 items-center gap-3 border-b border-border bg-background/90 px-4 backdrop-blur-md">
			{/* Mobile menu toggle */}
			<Button
				variant="ghost"
				size="icon-sm"
				onClick={onMobileToggle}
				className="docs-mobile-toggle lg:hidden"
				aria-label="Toggle sidebar"
			>
				<Menu className="h-5 w-5" />
			</Button>

			{/* Brand */}
			<div className="flex items-center gap-2">
				<DockeeperLogo size={32} />
				<span className="hidden text-lg font-semibold sm:inline">Dockeeper</span>
				<Separator orientation="vertical" className="!h-5" />
				<span className="text-sm text-muted-foreground">API Reference</span>
			</div>

			{/* Version badge */}
			<button onClick={onChangelogOpen} className="cursor-pointer" aria-label="Open changelog">
				<Badge variant="outline" className="border-primary/30 text-primary hover:bg-primary/10">
					v{latestVersion}
				</Badge>
			</button>

			<div className="flex-1" />

			{/* Actions */}
			<div className="flex items-center gap-2">
				{historySlot}
				<AuthorizeBadge onClick={() => setAuthOpen(true)} />
				<Button variant="outline" size="sm" asChild>
					<Link to={isAuthenticated ? "/" : "/login"}>
						{isAuthenticated ? (
							<ArrowLeft className="mr-1.5 h-3.5 w-3.5" />
						) : (
							<LogIn className="mr-1.5 h-3.5 w-3.5" />
						)}
						<span className="hidden sm:inline">{isAuthenticated ? "Dashboard" : "Login"}</span>
					</Link>
				</Button>
			</div>

			<AuthorizeDialog open={authOpen} onOpenChange={setAuthOpen} />
		</header>
	);
}
