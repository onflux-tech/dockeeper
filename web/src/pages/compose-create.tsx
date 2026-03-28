import { useState } from "react";
import { Link, useNavigate } from "react-router";
import {
	ArrowLeft,
	ArrowRight,
	Check,
	CheckCircle,
	AlertTriangle,
	Loader2,
	Rocket,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { useCollection } from "@/hooks/use-collection";
import {
	useCreateStack,
	useValidateCompose,
	streamComposeOp,
	type ComposeOutputEvent,
} from "@/hooks/use-compose";
import type { ServerRecord } from "@/types/pocketbase";

type Step = 0 | 1;

const STEPS = [
	{ label: "Configuration" },
	{ label: "Review & Create" },
];

const YAML_PLACEHOLDER = `version: "3.8"
services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"`;

export default function ComposeCreatePage() {
	const navigate = useNavigate();
	const [step, setStep] = useState<Step>(0);

	// Form state
	const [serverID, setServerID] = useState("");
	const [name, setName] = useState("");
	const [yaml, setYaml] = useState("");
	const [deployAfter, setDeployAfter] = useState(false);

	// Validation state
	const [validationValid, setValidationValid] = useState<boolean | null>(null);
	const [validationErrors, setValidationErrors] = useState<
		{ line: number; column: number; message: string; field: string }[]
	>([]);

	// Deploy logs
	const [deployLogs, setDeployLogs] = useState<string[]>([]);
	const [deploying, setDeploying] = useState(false);

	const { data: serversResult } = useCollection("servers", { perPage: 100 });
	const servers = (serversResult?.items ?? []) as ServerRecord[];
	const onlineServers = servers.filter((s) => s.status === "online");

	const validateMut = useValidateCompose();
	const createMut = useCreateStack();

	const selectedServer = servers.find((s) => s.id === serverID);

	const canProceed = serverID && name.trim() && yaml.trim() && validationValid === true;

	const handleValidate = async () => {
		if (!yaml.trim()) return;
		const result = await validateMut.mutateAsync(yaml);
		setValidationErrors(result.errors ?? []);
		setValidationValid(result.valid);
	};

	const handleCreate = async () => {
		if (!serverID || !name.trim() || !yaml.trim()) return;

		const stack = await createMut.mutateAsync({
			server_id: serverID,
			name: name.trim(),
			yaml_content: yaml,
		});

		if (deployAfter && stack.id) {
			setDeploying(true);
			setDeployLogs([">>> compose up"]);
			try {
				await streamComposeOp(stack.id, "up", (ev: ComposeOutputEvent) => {
					if (ev.line) setDeployLogs((prev) => [...prev, ev.line!]);
					if (ev.error) setDeployLogs((prev) => [...prev, `ERROR: ${ev.error}`]);
					if (ev.done) {
						const code = ev.exit_code ?? 0;
						setDeployLogs((prev) => [
							...prev,
							code === 0
								? "✓ compose up completed"
								: `✗ compose up exited with code ${code}`,
						]);
					}
				});
			} catch (err) {
				setDeployLogs((prev) => [...prev, `FATAL: ${(err as Error).message}`]);
			} finally {
				setDeploying(false);
			}
		}

		if (!deployAfter) {
			void navigate("/compose");
		}
	};

	return (
		<div className="flex flex-col gap-6">
			{/* Header */}
			<div className="flex items-center gap-3">
				<Button variant="ghost" size="icon" asChild>
					<Link to="/compose">
						<ArrowLeft className="h-4 w-4" />
					</Link>
				</Button>
				<div className="flex items-center gap-2">
					<Rocket className="h-5 w-5 text-muted-foreground" />
					<h1 className="text-2xl font-semibold">Create Stack</h1>
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
					<CardDescription>
						Step {step + 1} of {STEPS.length}
					</CardDescription>
				</CardHeader>
				<CardContent>
					{/* Step 1: Configuration */}
					{step === 0 && (
						<div className="space-y-6">
							<div className="grid gap-6 md:grid-cols-2">
								<div className="space-y-2">
									<Label htmlFor="stack-server">Server</Label>
									<Select value={serverID} onValueChange={setServerID}>
										<SelectTrigger id="stack-server">
											<SelectValue placeholder="Select a server..." />
										</SelectTrigger>
										<SelectContent>
											{onlineServers.map((s) => (
												<SelectItem key={s.id} value={s.id}>
													{s.hostname}
												</SelectItem>
											))}
										</SelectContent>
									</Select>
									{onlineServers.length === 0 && (
										<p className="text-xs text-muted-foreground">
											No online servers available.
										</p>
									)}
								</div>

								<div className="space-y-2">
									<Label htmlFor="stack-name">Stack Name</Label>
									<Input
										id="stack-name"
										placeholder="e.g. my-app"
										value={name}
										onChange={(e) => setName(e.target.value)}
									/>
								</div>
							</div>

							<div className="space-y-2">
								<div className="flex items-center justify-between">
									<Label htmlFor="stack-yaml">Compose YAML</Label>
									<Button
										variant="outline"
										size="sm"
										onClick={handleValidate}
										disabled={!yaml.trim() || validateMut.isPending}
									>
										{validateMut.isPending ? (
											<Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />
										) : validationValid === true ? (
											<CheckCircle className="mr-1.5 h-3.5 w-3.5 text-success" />
										) : validationValid === false ? (
											<AlertTriangle className="mr-1.5 h-3.5 w-3.5 text-destructive" />
										) : null}
										Validate
									</Button>
								</div>
								<Textarea
									id="stack-yaml"
									placeholder={YAML_PLACEHOLDER}
									value={yaml}
									onChange={(e) => {
										setYaml(e.target.value);
										setValidationValid(null);
										setValidationErrors([]);
									}}
									className="min-h-96 font-mono text-xs leading-relaxed"
									spellCheck={false}
								/>
							</div>

							{/* Validation Errors */}
							{validationErrors.length > 0 && (
								<div className="rounded-md border border-destructive/50 bg-destructive/5 p-3">
									<p className="mb-1.5 text-xs font-medium text-destructive">
										Validation Errors
									</p>
									<ul className="space-y-1">
										{validationErrors.map((err, i) => (
											<li key={i} className="text-xs text-destructive/80">
												{err.line > 0 && (
													<span className="font-mono text-destructive/60">
														Line {err.line}:{" "}
													</span>
												)}
												{err.message}
											</li>
										))}
									</ul>
								</div>
							)}
						</div>
					)}

					{/* Step 2: Review & Create */}
					{step === 1 && (
						<div className="space-y-6">
							{/* Summary grid */}
							<div className="grid gap-4 sm:grid-cols-2">
								<div className="space-y-1">
									<span className="text-xs text-muted-foreground">Server</span>
									<p className="text-sm font-medium">
										{selectedServer?.hostname ?? "—"}
									</p>
								</div>
								<div className="space-y-1">
									<span className="text-xs text-muted-foreground">Stack Name</span>
									<p className="text-sm font-medium">{name}</p>
								</div>
								<div className="space-y-1">
									<span className="text-xs text-muted-foreground">Validation</span>
									<div>
										<Badge variant="default" className="bg-success/15 text-success">
											<CheckCircle className="mr-1 h-3 w-3" />
											Valid
										</Badge>
									</div>
								</div>
							</div>

							{/* YAML preview */}
							<div className="space-y-2">
								<Label>Compose YAML Preview</Label>
								<pre className="max-h-64 overflow-auto rounded-lg border bg-muted p-4 font-mono text-xs leading-relaxed">
									{yaml}
								</pre>
							</div>

							{/* Deploy option */}
							<div className="flex items-center gap-3 rounded-lg border p-4">
								<Checkbox
									id="deploy-after"
									checked={deployAfter}
									onCheckedChange={(v) => setDeployAfter(v === true)}
								/>
								<div>
									<label htmlFor="deploy-after" className="cursor-pointer text-sm font-medium">
										Deploy immediately after creation
									</label>
									<p className="text-xs text-muted-foreground">
										Runs <code className="font-mono">docker compose up -d</code> right after saving
										the stack.
									</p>
								</div>
							</div>

							{/* Deploy logs */}
							{deployLogs.length > 0 && (
								<div className="space-y-2">
									<Label>Deploy Output</Label>
									<div className="max-h-48 overflow-y-auto rounded-lg border bg-muted p-4 font-mono text-xs leading-relaxed">
										{deployLogs.map((line, i) => (
											<div key={i}>{line}</div>
										))}
									</div>
								</div>
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
						if (step === 0) void navigate("/compose");
						else setStep(0);
					}}
					disabled={createMut.isPending || deploying}
				>
					<ArrowLeft className="mr-1.5 h-4 w-4" />
					Back
				</Button>

				{step === 0 ? (
					<Button onClick={() => setStep(1)} disabled={!canProceed}>
						Next
						<ArrowRight className="ml-1.5 h-4 w-4" />
					</Button>
				) : createMut.isSuccess && !deploying ? (
					<Button onClick={() => navigate("/compose")}>Done</Button>
				) : (
					<Button onClick={handleCreate} disabled={createMut.isPending || deploying}>
						{createMut.isPending ? (
							<Loader2 className="mr-1.5 h-4 w-4 animate-spin" />
						) : deploying ? (
							<Loader2 className="mr-1.5 h-4 w-4 animate-spin" />
						) : (
							<Rocket className="mr-1.5 h-4 w-4" />
						)}
						{createMut.isPending
							? "Creating..."
							: deploying
								? "Deploying..."
								: deployAfter
									? "Create & Deploy"
									: "Create Stack"}
					</Button>
				)}
			</div>
		</div>
	);
}
