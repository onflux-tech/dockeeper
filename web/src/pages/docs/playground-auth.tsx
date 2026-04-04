import { useState, type ReactNode } from "react";
import { Link } from "react-router";
import { Eye, EyeOff, KeyRound, Loader2, ShieldCheck, ShieldOff } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import {
	PlaygroundAuthContext,
	usePlaygroundAuth,
	usePlaygroundAuthValue,
} from "./use-playground-auth";

// ── Provider ───────────────────────────────────────────────────────────────

export function PlaygroundAuthProvider({ children }: { children: ReactNode }) {
	const value = usePlaygroundAuthValue();
	return <PlaygroundAuthContext.Provider value={value}>{children}</PlaygroundAuthContext.Provider>;
}

// ── Authorize Dialog ───────────────────────────────────────────────────────

interface AuthorizeDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
}

export function AuthorizeDialog({ open, onOpenChange }: AuthorizeDialogProps) {
	const { apiKey, isAuthorized, authorize, deauthorize } = usePlaygroundAuth();
	const [draft, setDraft] = useState("");
	const [showKey, setShowKey] = useState(false);
	const [isValidating, setIsValidating] = useState(false);
	const [error, setError] = useState("");

	const handleAuthorize = async () => {
		if (!draft.trim()) return;
		setIsValidating(true);
		setError("");
		const result = await authorize(draft.trim());
		setIsValidating(false);
		if (result.ok) {
			setDraft("");
			setShowKey(false);
			onOpenChange(false);
		} else {
			setError(result.error ?? "Invalid API key");
		}
	};

	const handleDeauthorize = () => {
		deauthorize();
		setDraft("");
		setShowKey(false);
		setError("");
	};

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="sm:max-w-md">
				<DialogHeader>
					<DialogTitle className="flex items-center gap-2">
						<KeyRound className="h-4 w-4" />
						API Authorization
					</DialogTitle>
					<DialogDescription>
						Enter your API key to test endpoints. The key is stored in this tab only and cleared
						when you close it.
					</DialogDescription>
				</DialogHeader>

				<div className="space-y-4 py-2">
					{isAuthorized ? (
						<div className="space-y-3">
							<div className="flex items-center gap-2 rounded-lg border border-green-400/30 bg-green-400/5 px-3 py-2.5">
								<ShieldCheck className="h-4 w-4 text-green-400" />
								<span className="text-sm font-medium text-green-400">Authorized</span>
								<span className="ml-auto font-mono text-xs text-muted-foreground">
									{showKey ? apiKey : `${"•".repeat(Math.min(apiKey.length, 20))}`}
								</span>
								<Button
									variant="ghost"
									size="icon-sm"
									onClick={() => setShowKey(!showKey)}
									aria-label={showKey ? "Hide key" : "Show key"}
								>
									{showKey ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
								</Button>
							</div>
							<Button
								variant="destructive"
								size="sm"
								className="w-full"
								onClick={handleDeauthorize}
							>
								<ShieldOff className="mr-1.5 h-3.5 w-3.5" />
								Deauthorize
							</Button>
						</div>
					) : (
						<div className="space-y-3">
							<div className="relative">
								<Input
									type={showKey ? "text" : "password"}
									placeholder="dk_..."
									value={draft}
									onChange={(e) => {
										setDraft(e.target.value);
										if (error) setError("");
									}}
									onKeyDown={(e) => e.key === "Enter" && handleAuthorize()}
									className={`pr-10 font-mono ${error ? "border-destructive" : ""}`}
									autoFocus
									disabled={isValidating}
								/>
								<Button
									variant="ghost"
									size="icon-sm"
									className="absolute right-1 top-1/2 -translate-y-1/2"
									onClick={() => setShowKey(!showKey)}
									aria-label={showKey ? "Hide key" : "Show key"}
								>
									{showKey ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
								</Button>
							</div>
							{error && <p className="text-xs text-destructive">{error}</p>}
							<Button
								size="sm"
								className="w-full"
								disabled={!draft.trim() || isValidating}
								onClick={handleAuthorize}
							>
								{isValidating ? (
									<Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />
								) : (
									<ShieldCheck className="mr-1.5 h-3.5 w-3.5" />
								)}
								{isValidating ? "Validating..." : "Authorize"}
							</Button>
						</div>
					)}
				</div>

				<DialogFooter className="sm:justify-start">
					<p className="text-xs text-muted-foreground">
						Don't have an API key?{" "}
						<Link
							to="/users?tab=api-keys"
							className="text-primary underline underline-offset-2 hover:text-primary/80"
						>
							Create one
						</Link>
					</p>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}

// ── Header badge button ────────────────────────────────────────────────────

interface AuthorizeBadgeProps {
	onClick: () => void;
}

export function AuthorizeBadge({ onClick }: AuthorizeBadgeProps) {
	const { isAuthorized } = usePlaygroundAuth();

	return (
		<Button
			variant="outline"
			size="sm"
			onClick={onClick}
			className="gap-1.5"
			aria-label={isAuthorized ? "API authorized — click to manage" : "Authorize API key"}
		>
			<span
				className={`h-2 w-2 rounded-full ${isAuthorized ? "bg-green-400" : "bg-muted-foreground/50"}`}
			/>
			<KeyRound className="h-3.5 w-3.5" />
			<span className="hidden sm:inline">{isAuthorized ? "Authorized" : "Authorize"}</span>
		</Button>
	);
}
