import { Badge } from "@/components/ui/badge";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { CodeBlock } from "./code-block";

// ── RBAC capabilities data ──────────────────────────────────────────────────

const CAPABILITIES: { domain: string; view: string | null; manage: string | null }[] = [
	{ domain: "Servers", view: "servers.view", manage: "servers.manage" },
	{ domain: "Containers", view: "containers.view", manage: "containers.manage" },
	{ domain: "Images", view: "images.view", manage: "images.manage" },
	{ domain: "Compose", view: "compose.view", manage: "compose.manage" },
	{ domain: "Networks", view: null, manage: "networks.manage" },
	{ domain: "Volumes", view: null, manage: "volumes.manage" },
	{ domain: "Backups", view: "backups.view", manage: "backups.manage" },
	{ domain: "Alerts", view: "alerts.view", manage: "alerts.manage" },
	{ domain: "Notifications", view: "notifications.view", manage: "notifications.manage" },
	{ domain: "Settings", view: "settings.view", manage: "settings.manage" },
	{ domain: "Users", view: "users.view", manage: "users.manage" },
	{ domain: "Audit", view: "audit.view", manage: null },
	{ domain: "Traefik", view: "traefik.view", manage: null },
	{ domain: "Events", view: "events.view", manage: null },
	{ domain: "Health Checks", view: "health.view", manage: "health.manage" },
	{ domain: "Batch Ops", view: "batch.view", manage: "batch.manage" },
	{ domain: "Tags", view: "tags.view", manage: "tags.manage" },
];

// ── Component ───────────────────────────────────────────────────────────────

export function GettingStarted() {
	return (
		<section id="getting-started" className="scroll-mt-24">
			{/* ── Introduction ─────────────────────────────────────────────── */}
			<div className="mb-12">
				<h2 className="text-2xl font-bold mb-4">Getting Started</h2>
				<p className="text-sm text-muted-foreground leading-relaxed mb-4">
					The Dockeeper REST API provides full programmatic access to your container infrastructure.
					All endpoints use the{" "}
					<code className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs">/api/dk/</code> base
					path and return JSON responses with appropriate HTTP status codes.
				</p>
				<div className="rounded-lg bg-muted/50 px-3 py-2">
					<code className="font-mono text-sm text-muted-foreground">
						Base URL: http://localhost:8090/api/dk/
					</code>
				</div>
			</div>

			{/* ── Authentication ───────────────────────────────────────────── */}
			<div id="auth-jwt" className="mb-12 scroll-mt-24">
				<h3 className="text-lg font-bold mb-3">Authentication — JWT Tokens</h3>
				<p className="text-sm text-muted-foreground leading-relaxed mb-4">
					Authenticate with email and password via{" "}
					<code className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs">
						POST /api/dk/auth/login
					</code>
					. The response includes an access token (15 min) and a refresh token (7 days). Include the
					access token in subsequent requests.
				</p>
				<CodeBlock
					curlCode={`curl -X POST http://localhost:8090/api/dk/auth/login \\
  -H "Content-Type: application/json" \\
  -d '{"email": "admin@example.com", "password": "your-password"}'

# Then use the token:
curl http://localhost:8090/api/dk/servers \\
  -H "Authorization: Bearer YOUR_TOKEN"`}
					jsCode={`const login = await fetch("http://localhost:8090/api/dk/auth/login", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    email: "admin@example.com",
    password: "your-password",
  }),
});
const { token } = await login.json();

// Then use the token:
const res = await fetch("http://localhost:8090/api/dk/servers", {
  headers: { Authorization: \`Bearer \${token}\` },
});`}
				/>
			</div>

			<div id="auth-api-key" className="mb-12 scroll-mt-24">
				<h3 className="text-lg font-bold mb-3">Authentication — API Keys</h3>
				<p className="text-sm text-muted-foreground leading-relaxed mb-4">
					Create API keys via the UI at{" "}
					<code className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs">
						/users → API Keys
					</code>{" "}
					tab. Keys are prefixed with{" "}
					<code className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs">dk_</code> and carry
					the capabilities of the group they belong to.
				</p>
				<CodeBlock
					curlCode={`curl http://localhost:8090/api/dk/servers \\
  -H "X-API-Key: dk_your_api_key_here"`}
					jsCode={`const res = await fetch("http://localhost:8090/api/dk/servers", {
  headers: { "X-API-Key": "dk_your_api_key_here" },
});
const data = await res.json();`}
				/>
			</div>

			<div id="auth-pocketbase" className="mb-12 scroll-mt-24">
				<h3 className="text-lg font-bold mb-3">Authentication — PocketBase Auth</h3>
				<p className="text-sm text-muted-foreground leading-relaxed mb-4">
					Users authenticated via PocketBase&apos;s built-in auth system inherit capabilities from
					their assigned groups. Superusers bypass all capability checks. The middleware evaluates
					access in order: superuser → API key → PocketBase session → 401.
				</p>
			</div>

			{/* ── Rate Limiting ────────────────────────────────────────────── */}
			<div id="rate-limiting" className="mb-12 scroll-mt-24">
				<h3 className="text-lg font-bold mb-3">Rate Limiting</h3>
				<p className="text-sm text-muted-foreground leading-relaxed mb-4">
					All API requests are subject to rate limiting by category. Limits are evaluated per IP
					address. When exceeded, the API returns{" "}
					<code className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs">
						429 Too Many Requests
					</code>
					.
				</p>
				<div className="rounded-lg border border-border overflow-hidden">
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead className="text-xs">Category</TableHead>
								<TableHead className="text-xs">Limit</TableHead>
								<TableHead className="text-xs">Window</TableHead>
								<TableHead className="text-xs">Applies To</TableHead>
							</TableRow>
						</TableHeader>
						<TableBody>
							<TableRow>
								<TableCell className="font-mono text-xs">auth</TableCell>
								<TableCell className="text-xs">5 requests</TableCell>
								<TableCell className="text-xs">1 minute</TableCell>
								<TableCell className="text-xs text-muted-foreground">Login, setup</TableCell>
							</TableRow>
							<TableRow>
								<TableCell className="font-mono text-xs">read</TableCell>
								<TableCell className="text-xs">120 requests</TableCell>
								<TableCell className="text-xs">1 minute</TableCell>
								<TableCell className="text-xs text-muted-foreground">GET endpoints</TableCell>
							</TableRow>
							<TableRow>
								<TableCell className="font-mono text-xs">write</TableCell>
								<TableCell className="text-xs">30 requests</TableCell>
								<TableCell className="text-xs">1 minute</TableCell>
								<TableCell className="text-xs text-muted-foreground">POST, PUT, PATCH</TableCell>
							</TableRow>
							<TableRow>
								<TableCell className="font-mono text-xs">destructive</TableCell>
								<TableCell className="text-xs">10 requests</TableCell>
								<TableCell className="text-xs">1 minute</TableCell>
								<TableCell className="text-xs text-muted-foreground">DELETE endpoints</TableCell>
							</TableRow>
						</TableBody>
					</Table>
				</div>
			</div>

			{/* ── RBAC Reference ───────────────────────────────────────────── */}
			<div id="rbac" className="mb-12 scroll-mt-24">
				<h3 className="text-lg font-bold mb-3">RBAC Capabilities</h3>
				<p className="text-sm text-muted-foreground leading-relaxed mb-4">
					Access control is capability-based. Users inherit capabilities from their groups. Each
					endpoint requires a specific capability — listed in the endpoint documentation below.
				</p>
				<div className="rounded-lg border border-border overflow-hidden">
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead className="text-xs">Domain</TableHead>
								<TableHead className="text-xs">View</TableHead>
								<TableHead className="text-xs">Manage</TableHead>
							</TableRow>
						</TableHeader>
						<TableBody>
							{CAPABILITIES.map((cap) => (
								<TableRow key={cap.domain}>
									<TableCell className="text-xs font-medium">{cap.domain}</TableCell>
									<TableCell>
										{cap.view ? (
											<Badge variant="outline" className="font-mono text-[10px]">
												{cap.view}
											</Badge>
										) : (
											<span className="text-xs text-muted-foreground">—</span>
										)}
									</TableCell>
									<TableCell>
										{cap.manage ? (
											<Badge variant="outline" className="font-mono text-[10px]">
												{cap.manage}
											</Badge>
										) : (
											<span className="text-xs text-muted-foreground">—</span>
										)}
									</TableCell>
								</TableRow>
							))}
						</TableBody>
					</Table>
				</div>
			</div>

			{/* ── Response Format ──────────────────────────────────────────── */}
			<div id="response-format" className="mb-12 scroll-mt-24">
				<h3 className="text-lg font-bold mb-3">Response Format</h3>
				<p className="text-sm text-muted-foreground leading-relaxed mb-4">
					All responses use JSON. Successful responses return the resource directly. Errors return a
					consistent format with an appropriate HTTP status code.
				</p>
				<div className="space-y-3">
					<div>
						<h4 className="text-xs font-bold uppercase tracking-widest text-muted-foreground mb-2">
							Error Response
						</h4>
						<pre className="rounded-lg bg-muted/50 p-3 overflow-x-auto text-xs font-mono leading-relaxed text-muted-foreground">
							<code>{`{
  "error": "Descriptive error message"
}`}</code>
						</pre>
					</div>
					<div>
						<h4 className="text-xs font-bold uppercase tracking-widest text-muted-foreground mb-2">
							Common Status Codes
						</h4>
						<div className="rounded-lg border border-border overflow-hidden">
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead className="text-xs">Code</TableHead>
										<TableHead className="text-xs">Meaning</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{[
										["200", "Success"],
										["201", "Created"],
										["400", "Bad request / validation error"],
										["401", "Unauthorized — missing or invalid token"],
										["403", "Forbidden — insufficient capabilities"],
										["404", "Resource not found"],
										["429", "Rate limit exceeded"],
										["500", "Internal server error"],
									].map(([code, desc]) => (
										<TableRow key={code}>
											<TableCell className="font-mono text-xs font-semibold">{code}</TableCell>
											<TableCell className="text-xs text-muted-foreground">{desc}</TableCell>
										</TableRow>
									))}
								</TableBody>
							</Table>
						</div>
					</div>
				</div>
			</div>
		</section>
	);
}
