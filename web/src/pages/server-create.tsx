import { useState } from "react";
import { Link, useNavigate } from "react-router";
import {
	ArrowLeft,
	ArrowRight,
	Check,
	Copy,
	Download,
	Loader2,
	Monitor,
	Server,
	Terminal,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Textarea } from "@/components/ui/textarea";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import { useProvisionServer, useInstallScript } from "@/hooks/use-servers";

type Step = 0 | 1 | 2;
type Mode = "docker" | "binary";

const STEPS = [
	{ label: "Server Info" },
	{ label: "Installation Method" },
	{ label: "Install Agent" },
];

export default function ServerCreatePage() {
	const navigate = useNavigate();
	const [step, setStep] = useState<Step>(0);
	const [name, setName] = useState("");
	const [description, setDescription] = useState("");
	const [mode, setMode] = useState<Mode>("docker");
	const [serverId, setServerId] = useState("");
	const [copied, setCopied] = useState(false);

	const provision = useProvisionServer();
	const { data: scriptData, isLoading: scriptLoading } = useInstallScript(
		serverId,
		mode,
	);

	const handleProvision = () => {
		provision.mutate(
			{ name: name.trim(), description: description.trim() },
			{
				onSuccess: (data) => {
					setServerId(data.id);
					setStep(1);
				},
			},
		);
	};

	const handleCopy = async () => {
		if (!scriptData?.script) return;
		await navigator.clipboard.writeText(scriptData.script);
		setCopied(true);
		setTimeout(() => setCopied(false), 2000);
	};

	const handleDownload = () => {
		if (!scriptData?.script) return;
		const filename = `dockeeper-install-${name.trim().replace(/\s+/g, "-").toLowerCase()}.sh`;
		const blob = new Blob([scriptData.script], {
			type: "text/x-shellscript",
		});
		const url = URL.createObjectURL(blob);
		const a = document.createElement("a");
		a.href = url;
		a.download = filename;
		a.click();
		URL.revokeObjectURL(url);
	};

	return (
		<div className="flex flex-col gap-6">
			{/* Header */}
			<div className="flex items-center gap-3">
				<Button variant="ghost" size="icon" asChild>
					<Link to="/servers">
						<ArrowLeft className="h-4 w-4" />
					</Link>
				</Button>
				<div className="flex items-center gap-2">
					<Server className="h-5 w-5 text-muted-foreground" />
					<h1 className="text-2xl font-semibold">Add Server</h1>
				</div>
			</div>

			{/* Step Indicator */}
			<nav aria-label="Wizard steps" className="flex items-center gap-1 overflow-x-auto pb-2">
				{STEPS.map((s, i) => {
					const isCompleted = i < step;
					const isCurrent = i === step;

					return (
						<button
							key={s.label}
							type="button"
							onClick={() => {
								if (i <= step) setStep(i as Step);
							}}
							className={cn(
								"flex shrink-0 items-center gap-2 rounded-md px-3 py-2 text-sm transition-colors",
								isCurrent && "bg-primary text-primary-foreground",
								isCompleted && "text-primary hover:bg-accent",
								!isCurrent && !isCompleted && "text-muted-foreground hover:bg-accent",
							)}
						>
							<span
								className={cn(
									"flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-xs font-medium",
									isCurrent && "bg-primary-foreground text-primary",
									isCompleted && "bg-primary text-primary-foreground",
									!isCurrent && !isCompleted && "border border-muted-foreground/40",
								)}
							>
								{isCompleted ? <Check className="h-3.5 w-3.5" /> : i + 1}
							</span>
							<span className="hidden sm:inline">{s.label}</span>
						</button>
					);
				})}
			</nav>

			{/* Card */}
			<Card>
				<CardHeader>
					<CardTitle>{STEPS[step].label}</CardTitle>
					<CardDescription>Step {step + 1} of {STEPS.length}</CardDescription>
				</CardHeader>
				<CardContent>
					{/* Step 1: Name + Description */}
					{step === 0 && (
						<div className="space-y-4 max-w-md">
							<div className="space-y-2">
								<Label htmlFor="server-name">Server Name</Label>
								<Input
									id="server-name"
									placeholder="e.g. production-web-01"
									value={name}
									onChange={(e) => setName(e.target.value)}
									autoFocus
								/>
							</div>
							<div className="space-y-2">
								<Label htmlFor="server-desc">
									Description{" "}
									<span className="text-muted-foreground">
										(optional)
									</span>
								</Label>
								<Input
									id="server-desc"
									placeholder="e.g. Main web server"
									value={description}
									onChange={(e) =>
										setDescription(e.target.value)
									}
								/>
							</div>
						</div>
					)}

					{/* Step 2: Choose method */}
					{step === 1 && (
						<div className="grid gap-3 max-w-lg">
							<button
								type="button"
								className={`flex items-center gap-3 rounded-lg border p-4 text-left transition-colors hover:bg-accent ${
									mode === "docker"
										? "border-primary bg-accent"
										: ""
								}`}
								onClick={() => setMode("docker")}
							>
								<Monitor className="h-5 w-5 shrink-0 text-muted-foreground" />
								<div>
									<p className="font-medium">Docker</p>
									<p className="text-sm text-muted-foreground">
										Run agent as a Docker container.
										Requires Docker installed.
									</p>
								</div>
							</button>
							<button
								type="button"
								className={`flex items-center gap-3 rounded-lg border p-4 text-left transition-colors hover:bg-accent ${
									mode === "binary"
										? "border-primary bg-accent"
										: ""
								}`}
								onClick={() => setMode("binary")}
							>
								<Terminal className="h-5 w-5 shrink-0 text-muted-foreground" />
								<div>
									<p className="font-medium">
										Install Script
									</p>
									<p className="text-sm text-muted-foreground">
										Download binary and install as systemd
										service.
									</p>
								</div>
							</button>
						</div>
					)}

					{/* Step 3: Install command */}
					{step === 2 && (
						<div className="space-y-4">
							{scriptLoading ? (
								<div className="flex items-center justify-center py-8">
									<Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
								</div>
							) : (
								<>
									<div className="flex items-center gap-2">
										<Badge variant="secondary">
											{mode === "docker"
												? "Docker"
												: "Binary"}
										</Badge>
										<span className="text-sm text-muted-foreground">
											{mode === "docker"
												? "Run this command on your target server:"
												: "Download and execute the install script on your target server:"}
										</span>
									</div>
									<Textarea
										readOnly
										value={scriptData?.script ?? ""}
										className="h-56 max-h-56 resize-none font-mono text-xs leading-relaxed"
										spellCheck={false}
									/>
									<div className="flex items-center gap-2">
										<Button
											variant="outline"
											size="sm"
											onClick={handleCopy}
										>
											{copied ? (
												<Check className="mr-1.5 h-3.5 w-3.5" />
											) : (
												<Copy className="mr-1.5 h-3.5 w-3.5" />
											)}
											{copied ? "Copied" : "Copy"}
										</Button>
										<Button
											variant="outline"
											size="sm"
											onClick={handleDownload}
										>
											<Download className="mr-1.5 h-3.5 w-3.5" />
											Download .sh
										</Button>
									</div>
									<p className="text-xs text-muted-foreground">
										After downloading, run on the target
										server:{" "}
										<code className="rounded bg-muted px-1 py-0.5 font-mono">
											chmod +x dockeeper-install-*.sh
											&& sudo ./dockeeper-install-*.sh
										</code>
									</p>
									<p className="text-xs text-muted-foreground">
										The server will appear as{" "}
										<strong>online</strong> once the agent
										connects successfully.
									</p>
								</>
							)}
						</div>
					)}
				</CardContent>
			</Card>

			{/* Navigation */}
			<div className="flex items-center justify-between">
				<Button
					type="button"
					variant="outline"
					onClick={() => {
						if (step === 0) navigate("/servers");
						else setStep((step - 1) as Step);
					}}
					disabled={step === 0}
				>
					<ArrowLeft className="mr-1.5 h-4 w-4" />
					Back
				</Button>

				{step === 2 ? (
					<Button onClick={() => navigate("/servers")}>Done</Button>
				) : step === 0 ? (
					<Button
						onClick={handleProvision}
						disabled={!name.trim() || provision.isPending}
					>
						{provision.isPending ? (
							<Loader2 className="mr-1.5 h-4 w-4 animate-spin" />
						) : null}
						Next
						<ArrowRight className="ml-1.5 h-4 w-4" />
					</Button>
				) : (
					<Button onClick={() => setStep(2)}>
						Next
						<ArrowRight className="ml-1.5 h-4 w-4" />
					</Button>
				)}
			</div>
		</div>
	);
}
