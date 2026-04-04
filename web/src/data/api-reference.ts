import {
	Activity,
	Archive,
	Bell,
	BookOpen,
	Box,
	Cog,
	HardDrive,
	Key,
	Layers,
	Network,
	Database,
	Search,
	Send,
	Server,
	Settings,
	Shield,
	Tag,
	Zap,
} from "lucide-react";

import type { ApiGroup } from "@/types/docs";

// ── Helper ──────────────────────────────────────────────────────────────────

const json = (obj: unknown) => JSON.stringify(obj, null, 2);

// ── Group 1: Getting Started (sidebar meta, 0 endpoints) ───────────────────

const gettingStarted: ApiGroup = {
	id: "getting-started",
	title: "Getting Started",
	description: "Introduction, authentication, rate limiting, RBAC, and response formats.",
	icon: BookOpen,
	endpoints: [],
};

// ── Group 2: Authentication ────────────────────────────────────────────────

const authentication: ApiGroup = {
	id: "authentication",
	title: "Authentication",
	description: "Admin authentication, token management, and initial setup.",
	icon: Key,
	endpoints: [
		{
			id: "auth-status",
			method: "GET",
			path: "/api/dk/auth/status",
			title: "Auth Status",
			description: "Check if the initial admin setup has been completed. Public endpoint.",
			auth: "public",
			responseExample: json({ setupCompleted: true }),
			curlExample: `curl http://localhost:8090/api/dk/auth/status`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/auth/status");
const data = await res.json();`,
		},
		{
			id: "auth-setup",
			method: "POST",
			path: "/api/dk/auth/setup",
			title: "Initial Setup",
			description: "Create the first admin user. Only works if no admin exists yet.",
			auth: "public",
			requestBody: {
				contentType: "application/json",
				fields: [
					{ name: "email", type: "string", required: true, description: "Admin email address." },
					{
						name: "password",
						type: "string",
						required: true,
						description: "Admin password (min 8 chars).",
					},
				],
			},
			curlExample: `curl -X POST http://localhost:8090/api/dk/auth/setup \\
  -H "Content-Type: application/json" \\
  -d '{"email": "admin@example.com", "password": "securepass"}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/auth/setup", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    email: "admin@example.com",
    password: "securepass",
  }),
});`,
		},
		{
			id: "auth-login",
			method: "POST",
			path: "/api/dk/auth/login",
			title: "Login",
			description:
				"Authenticate with email and password. Returns an access token (15 min) and a refresh token (7 days).",
			auth: "public",
			requestBody: {
				contentType: "application/json",
				fields: [
					{
						name: "email",
						type: "string",
						required: true,
						description: "Admin email address.",
						example: "admin@example.com",
					},
					{
						name: "password",
						type: "string",
						required: true,
						description: "Admin password.",
						example: "your-password",
					},
				],
			},
			responseExample: json({
				token: "eyJhbGciOiJIUzI1NiIs...",
				refresh: "eyJhbGciOiJIUzI1NiIs...",
				admin: { id: "abc123", email: "admin@example.com" },
			}),
			curlExample: `curl -X POST http://localhost:8090/api/dk/auth/login \\
  -H "Content-Type: application/json" \\
  -d '{"email": "admin@example.com", "password": "your-password"}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/auth/login", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    email: "admin@example.com",
    password: "your-password",
  }),
});
const { token, refresh } = await res.json();`,
		},
		{
			id: "auth-refresh",
			method: "POST",
			path: "/api/dk/auth/refresh",
			title: "Refresh Token",
			description: "Exchange a valid refresh token for a new access/refresh token pair.",
			auth: "superuser",
			curlExample: `curl -X POST http://localhost:8090/api/dk/auth/refresh \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/auth/refresh", {
  method: "POST",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const { token, refresh } = await res.json();`,
		},
		{
			id: "auth-logout",
			method: "POST",
			path: "/api/dk/auth/logout",
			title: "Logout",
			description: "Invalidate the current session.",
			auth: "superuser",
			curlExample: `curl -X POST http://localhost:8090/api/dk/auth/logout \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/auth/logout", {
  method: "POST",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
		{
			id: "auth-password",
			method: "POST",
			path: "/api/dk/auth/password",
			title: "Change Password",
			description: "Update the authenticated admin's password.",
			auth: "superuser",
			requestBody: {
				contentType: "application/json",
				fields: [
					{
						name: "current_password",
						type: "string",
						required: true,
						description: "Current password.",
					},
					{
						name: "new_password",
						type: "string",
						required: true,
						description: "New password (min 8 chars).",
					},
				],
			},
			curlExample: `curl -X POST http://localhost:8090/api/dk/auth/password \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"current_password": "old", "new_password": "new"}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/auth/password", {
  method: "POST",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    current_password: "old",
    new_password: "new",
  }),
});`,
		},
		{
			id: "auth-me",
			method: "GET",
			path: "/api/dk/auth/me",
			title: "Get Current User",
			description: "Returns the authenticated admin's profile data.",
			auth: "superuser",
			responseExample: json({
				id: "abc123",
				email: "admin@example.com",
				created: "2026-01-01 00:00:00.000Z",
				updated: "2026-01-01 00:00:00.000Z",
			}),
			curlExample: `curl http://localhost:8090/api/dk/auth/me \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/auth/me", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const user = await res.json();`,
		},
	],
};

// ── Group 3: Servers ───────────────────────────────────────────────────────

const servers: ApiGroup = {
	id: "servers",
	title: "Servers",
	description: "Server provisioning, management, and enrollment token operations.",
	icon: Server,
	endpoints: [
		{
			id: "servers-list",
			method: "GET",
			path: "/api/dk/servers",
			title: "List Servers",
			description:
				"Retrieve all registered servers with their current status, metrics, and agent info.",
			auth: "required",
			capability: "servers.view",
			responseExample: json([
				{
					id: "srv_01",
					hostname: "prod-1",
					status: "online",
					ip: "10.0.1.10",
					agent_version: "0.1.0",
				},
			]),
			curlExample: `curl http://localhost:8090/api/dk/servers \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/servers", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const servers = await res.json();`,
		},
		{
			id: "servers-provision",
			method: "POST",
			path: "/api/dk/servers/provision",
			title: "Provision Server",
			description:
				"Create a new server record and generate a one-time enrollment token for agent installation.",
			auth: "required",
			capability: "servers.manage",
			requestBody: {
				contentType: "application/json",
				fields: [
					{
						name: "name",
						type: "string",
						required: true,
						description: "Display name for the server.",
					},
					{
						name: "description",
						type: "string",
						required: false,
						description: "Optional server description.",
					},
				],
			},
			curlExample: `curl -X POST http://localhost:8090/api/dk/servers/provision \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"name": "prod-2"}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/servers/provision", {
  method: "POST",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ name: "prod-2" }),
});
const server = await res.json();`,
		},
		{
			id: "servers-install-script",
			method: "GET",
			path: "/api/dk/servers/{id}/install-script",
			title: "Get Install Script",
			description:
				"Download the agent installation script for a specific server. Includes the enrollment token.",
			auth: "required",
			capability: "servers.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Server record ID." },
			],
			curlExample: `curl http://localhost:8090/api/dk/servers/SERVER_ID/install-script \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/servers/SERVER_ID/install-script", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const script = await res.text();`,
		},
		{
			id: "servers-update",
			method: "PATCH",
			path: "/api/dk/servers/{id}",
			title: "Update Server",
			description: "Update a server's display name or description.",
			auth: "required",
			capability: "servers.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Server record ID." },
			],
			requestBody: {
				contentType: "application/json",
				fields: [
					{ name: "name", type: "string", required: true, description: "New display name." },
					{ name: "description", type: "string", required: false, description: "New description." },
				],
			},
			curlExample: `curl -X PATCH http://localhost:8090/api/dk/servers/SERVER_ID \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"name": "prod-1-renamed"}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/servers/SERVER_ID", {
  method: "PATCH",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ name: "prod-1-renamed" }),
});`,
		},
		{
			id: "servers-delete",
			method: "DELETE",
			path: "/api/dk/servers/{id}",
			title: "Delete Server",
			description:
				"Remove a server and all associated records (containers, metrics, backups). Triggers cascade delete.",
			auth: "required",
			capability: "servers.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Server record ID." },
			],
			curlExample: `curl -X DELETE http://localhost:8090/api/dk/servers/SERVER_ID \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/servers/SERVER_ID", {
  method: "DELETE",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
		{
			id: "servers-quick-install",
			method: "GET",
			path: "/api/dk/servers/quick-install",
			title: "Quick Install Command",
			description:
				"Get a one-line install command that provisions a new server and installs the agent.",
			auth: "required",
			capability: "servers.manage",
			curlExample: `curl http://localhost:8090/api/dk/servers/quick-install \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/servers/quick-install", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const { command } = await res.json();`,
		},
		{
			id: "enrollment-token-get",
			method: "GET",
			path: "/api/dk/enrollment-token",
			title: "Get Enrollment Token",
			description: "Retrieve the current global enrollment token used for agent registration.",
			auth: "required",
			capability: "servers.manage",
			curlExample: `curl http://localhost:8090/api/dk/enrollment-token \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/enrollment-token", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const { token } = await res.json();`,
		},
		{
			id: "enrollment-token-regenerate",
			method: "POST",
			path: "/api/dk/enrollment-token/regenerate",
			title: "Regenerate Enrollment Token",
			description: "Generate a new enrollment token, invalidating the previous one.",
			auth: "required",
			capability: "servers.manage",
			curlExample: `curl -X POST http://localhost:8090/api/dk/enrollment-token/regenerate \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/enrollment-token/regenerate", {
  method: "POST",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const { token } = await res.json();`,
		},
	],
};

// ── Group 4: Containers ────────────────────────────────────────────────────

const containers: ApiGroup = {
	id: "containers",
	title: "Containers",
	description:
		"Container lifecycle management, inspection, resource limits, and environment variables.",
	icon: Box,
	endpoints: [
		{
			id: "containers-create",
			method: "POST",
			path: "/api/dk/containers/create",
			title: "Create Container",
			description:
				"Create and optionally start a new container on a target server. Auto-pulls the image if not present.",
			auth: "required",
			capability: "containers.manage",
			requestBody: {
				contentType: "application/json",
				fields: [
					{ name: "server_id", type: "string", required: true, description: "Target server ID." },
					{ name: "name", type: "string", required: true, description: "Container name." },
					{
						name: "image",
						type: "string",
						required: true,
						description: "Docker image reference.",
						example: "nginx:latest",
					},
					{
						name: "ports",
						type: "object[]",
						required: false,
						description: "Port mappings: [{host_port, container_port, protocol}].",
					},
					{
						name: "env",
						type: "object",
						required: false,
						description: 'Environment variables as key-value map: {"KEY": "VALUE"}.',
					},
					{
						name: "volumes",
						type: "object[]",
						required: false,
						description: "Volume mounts: [{source, target, read_only}].",
					},
					{
						name: "labels",
						type: "object",
						required: false,
						description: 'Container labels as key-value map: {"key": "value"}.',
					},
					{
						name: "network",
						type: "string",
						required: false,
						description: "Docker network to connect.",
					},
					{
						name: "cpu_limit",
						type: "number",
						required: false,
						description: "CPU limit in cores (e.g. 0.5, 1, 2).",
					},
					{
						name: "mem_limit",
						type: "number",
						required: false,
						description: "Memory limit in bytes.",
					},
					{
						name: "pids_limit",
						type: "number",
						required: false,
						description: "Max number of PIDs.",
					},
					{
						name: "restart_policy",
						type: "string",
						required: false,
						description: "Restart policy (no, always, unless-stopped, on-failure).",
					},
					{
						name: "command",
						type: "string[]",
						required: false,
						description: "Override container command.",
					},
					{
						name: "entrypoint",
						type: "string[]",
						required: false,
						description: "Override container entrypoint.",
					},
					{
						name: "auto_start",
						type: "boolean",
						required: false,
						description: "Start the container after creation (default: true).",
					},
				],
			},
			responseExample: json({ id: "ctr_abc123", name: "my-nginx", status: "running" }),
			curlExample: `curl -X POST http://localhost:8090/api/dk/containers/create \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "server_id": "SERVER_ID",
    "name": "my-nginx",
    "image": "nginx:latest",
    "ports": [{"host_port": "8080", "container_port": "80", "protocol": "tcp"}],
    "auto_start": true
  }'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/containers/create", {
  method: "POST",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    server_id: "SERVER_ID",
    name: "my-nginx",
    image: "nginx:latest",
    ports: [{ host_port: "8080", container_port: "80", protocol: "tcp" }],
    auto_start: true,
  }),
});`,
		},
		{
			id: "containers-inspect",
			method: "GET",
			path: "/api/dk/containers/{id}/inspect",
			title: "Inspect Container",
			description:
				"Retrieve detailed information about a container including config, state, and network settings.",
			auth: "required",
			capability: "containers.view",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Container record ID." },
			],
			curlExample: `curl http://localhost:8090/api/dk/containers/CONTAINER_ID/inspect \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/containers/CONTAINER_ID/inspect", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const info = await res.json();`,
		},
		{
			id: "containers-start",
			method: "POST",
			path: "/api/dk/containers/{id}/start",
			title: "Start Container",
			description: "Start a stopped container.",
			auth: "required",
			capability: "containers.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Container record ID." },
			],
			curlExample: `curl -X POST http://localhost:8090/api/dk/containers/CONTAINER_ID/start \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/containers/CONTAINER_ID/start", {
  method: "POST",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
		{
			id: "containers-stop",
			method: "POST",
			path: "/api/dk/containers/{id}/stop",
			title: "Stop Container",
			description: "Gracefully stop a running container.",
			auth: "required",
			capability: "containers.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Container record ID." },
			],
			curlExample: `curl -X POST http://localhost:8090/api/dk/containers/CONTAINER_ID/stop \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/containers/CONTAINER_ID/stop", {
  method: "POST",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
		{
			id: "containers-restart",
			method: "POST",
			path: "/api/dk/containers/{id}/restart",
			title: "Restart Container",
			description: "Restart a container (stop + start).",
			auth: "required",
			capability: "containers.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Container record ID." },
			],
			curlExample: `curl -X POST http://localhost:8090/api/dk/containers/CONTAINER_ID/restart \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/containers/CONTAINER_ID/restart", {
  method: "POST",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
		{
			id: "containers-remove",
			method: "POST",
			path: "/api/dk/containers/{id}/remove",
			title: "Remove Container",
			description:
				"Remove a container. The container must be stopped first unless force is applied.",
			auth: "required",
			capability: "containers.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Container record ID." },
			],
			curlExample: `curl -X POST http://localhost:8090/api/dk/containers/CONTAINER_ID/remove \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/containers/CONTAINER_ID/remove", {
  method: "POST",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
		{
			id: "containers-pause",
			method: "POST",
			path: "/api/dk/containers/{id}/pause",
			title: "Pause Container",
			description: "Pause all processes within a running container.",
			auth: "required",
			capability: "containers.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Container record ID." },
			],
			curlExample: `curl -X POST http://localhost:8090/api/dk/containers/CONTAINER_ID/pause \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/containers/CONTAINER_ID/pause", {
  method: "POST",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
		{
			id: "containers-unpause",
			method: "POST",
			path: "/api/dk/containers/{id}/unpause",
			title: "Unpause Container",
			description: "Resume a paused container.",
			auth: "required",
			capability: "containers.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Container record ID." },
			],
			curlExample: `curl -X POST http://localhost:8090/api/dk/containers/CONTAINER_ID/unpause \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/containers/CONTAINER_ID/unpause", {
  method: "POST",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
		{
			id: "containers-desired-state",
			method: "PUT",
			path: "/api/dk/containers/{id}/desired-state",
			title: "Set Desired State",
			description:
				"Set the desired state for a container (running or stopped). The agent will reconcile automatically.",
			auth: "required",
			capability: "containers.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Container record ID." },
			],
			requestBody: {
				contentType: "application/json",
				fields: [
					{
						name: "desired_state",
						type: "string",
						required: true,
						description: 'Target state: "running" or "stopped".',
					},
				],
			},
			curlExample: `curl -X PUT http://localhost:8090/api/dk/containers/CONTAINER_ID/desired-state \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"desired_state": "running"}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/containers/CONTAINER_ID/desired-state", {
  method: "PUT",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ desired_state: "running" }),
});`,
		},
		{
			id: "containers-resources-get",
			method: "GET",
			path: "/api/dk/containers/{id}/resources",
			title: "Get Resource Limits",
			description: "Retrieve the current CPU and memory resource limits for a container.",
			auth: "required",
			capability: "containers.view",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Container record ID." },
			],
			curlExample: `curl http://localhost:8090/api/dk/containers/CONTAINER_ID/resources \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/containers/CONTAINER_ID/resources", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const limits = await res.json();`,
		},
		{
			id: "containers-resources-update",
			method: "PUT",
			path: "/api/dk/containers/{id}/resources",
			title: "Update Resource Limits",
			description: "Update CPU and memory limits for a running container.",
			auth: "required",
			capability: "containers.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Container record ID." },
			],
			requestBody: {
				contentType: "application/json",
				fields: [
					{
						name: "cpu_limit",
						type: "number",
						required: false,
						description: "CPU limit in cores (e.g. 0.5, 1, 2).",
					},
					{
						name: "mem_limit",
						type: "number",
						required: false,
						description: "Memory limit in bytes.",
					},
					{
						name: "pids_limit",
						type: "number",
						required: false,
						description: "Max number of PIDs (0 = unlimited).",
					},
				],
			},
			curlExample: `curl -X PUT http://localhost:8090/api/dk/containers/CONTAINER_ID/resources \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"cpu_limit": 1, "mem_limit": 536870912, "pids_limit": 200}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/containers/CONTAINER_ID/resources", {
  method: "PUT",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ cpu_limit: 1, mem_limit: 536870912, pids_limit: 200 }),
});`,
		},
		{
			id: "containers-env-get",
			method: "GET",
			path: "/api/dk/containers/{id}/env",
			title: "Get Environment Variables",
			description: "Retrieve the current environment variables of a container.",
			auth: "required",
			capability: "containers.view",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Container record ID." },
			],
			curlExample: `curl http://localhost:8090/api/dk/containers/CONTAINER_ID/env \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/containers/CONTAINER_ID/env", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const { env } = await res.json();`,
		},
		{
			id: "containers-env-update",
			method: "PUT",
			path: "/api/dk/containers/{id}/env",
			title: "Update Environment Variables",
			description: "Update environment variables. Requires container recreation to apply changes.",
			auth: "required",
			capability: "containers.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Container record ID." },
			],
			requestBody: {
				contentType: "application/json",
				fields: [
					{
						name: "env",
						type: "object",
						required: true,
						description: "Environment variables as key-value map.",
						example: '{"NODE_ENV": "production", "PORT": "3000"}',
					},
				],
			},
			curlExample: `curl -X PUT http://localhost:8090/api/dk/containers/CONTAINER_ID/env \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"env": {"NODE_ENV": "production", "PORT": "3000"}}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/containers/CONTAINER_ID/env", {
  method: "PUT",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ env: { NODE_ENV: "production", PORT: "3000" } }),
});`,
		},
	],
};

// ── Group 5: Images ────────────────────────────────────────────────────────

const images: ApiGroup = {
	id: "images",
	title: "Images",
	description: "Docker image listing, removal, and cleanup across servers.",
	icon: HardDrive,
	endpoints: [
		{
			id: "images-list",
			method: "GET",
			path: "/api/dk/images",
			title: "List Images",
			description: "Retrieve all Docker images across all servers.",
			auth: "required",
			capability: "images.view",
			curlExample: `curl http://localhost:8090/api/dk/images \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/images", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const images = await res.json();`,
		},
		{
			id: "images-remove",
			method: "DELETE",
			path: "/api/dk/images/{id}",
			title: "Remove Image",
			description:
				"Remove a Docker image from its server. Fails if the image is in use by a container.",
			auth: "required",
			capability: "images.manage",
			parameters: [{ name: "id", type: "string", required: true, description: "Image record ID." }],
			curlExample: `curl -X DELETE http://localhost:8090/api/dk/images/IMAGE_ID \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/images/IMAGE_ID", {
  method: "DELETE",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
		{
			id: "images-prune",
			method: "POST",
			path: "/api/dk/images/prune",
			title: "Prune Unused Images",
			description: "Remove all dangling (unused) images from a target server to free disk space.",
			auth: "required",
			capability: "images.manage",
			requestBody: {
				contentType: "application/json",
				fields: [
					{ name: "server_id", type: "string", required: true, description: "Target server ID." },
					{
						name: "all",
						type: "boolean",
						required: false,
						description: "Remove all unused images, not just dangling ones (default: false).",
					},
				],
			},
			curlExample: `curl -X POST http://localhost:8090/api/dk/images/prune \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"server_id": "SERVER_ID", "all": true}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/images/prune", {
  method: "POST",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ server_id: "SERVER_ID", all: true }),
});
const { spaceReclaimed } = await res.json();`,
		},
	],
};

// ── Group 6: Compose ───────────────────────────────────────────────────────

const compose: ApiGroup = {
	id: "compose",
	title: "Compose Stacks",
	description: "Docker Compose stack CRUD operations.",
	icon: Layers,
	endpoints: [
		{
			id: "compose-list",
			method: "GET",
			path: "/api/dk/compose",
			title: "List Stacks",
			description: "Retrieve all compose stacks with their status and linked containers.",
			auth: "required",
			capability: "compose.view",
			curlExample: `curl http://localhost:8090/api/dk/compose \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/compose", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const stacks = await res.json();`,
		},
		{
			id: "compose-get",
			method: "GET",
			path: "/api/dk/compose/{id}",
			title: "Get Stack",
			description: "Retrieve a specific compose stack with its YAML content and linked containers.",
			auth: "required",
			capability: "compose.view",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Compose stack record ID." },
			],
			curlExample: `curl http://localhost:8090/api/dk/compose/STACK_ID \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/compose/STACK_ID", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const stack = await res.json();`,
		},
		{
			id: "compose-create",
			method: "POST",
			path: "/api/dk/compose",
			title: "Create Stack",
			description:
				"Create a new compose stack with YAML content. Does not deploy — use the Up endpoint to deploy.",
			auth: "required",
			capability: "compose.manage",
			requestBody: {
				contentType: "application/json",
				fields: [
					{ name: "server_id", type: "string", required: true, description: "Target server ID." },
					{ name: "name", type: "string", required: true, description: "Stack name." },
					{
						name: "yaml_content",
						type: "string",
						required: true,
						description: "Docker Compose YAML content.",
					},
				],
			},
			curlExample: `curl -X POST http://localhost:8090/api/dk/compose \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"server_id": "SERVER_ID", "name": "my-stack", "yaml_content": "version: \\"3\\"\\nservices:\\n  web:\\n    image: nginx"}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/compose", {
  method: "POST",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    server_id: "SERVER_ID",
    name: "my-stack",
    yaml_content: "version: \\"3\\"\\nservices:\\n  web:\\n    image: nginx",
  }),
});`,
		},
		{
			id: "compose-update",
			method: "PUT",
			path: "/api/dk/compose/{id}",
			title: "Update Stack",
			description: "Update a compose stack's YAML content or name.",
			auth: "required",
			capability: "compose.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Compose stack record ID." },
			],
			requestBody: {
				contentType: "application/json",
				fields: [
					{
						name: "yaml_content",
						type: "string",
						required: true,
						description: "Updated YAML content.",
					},
				],
			},
			curlExample: `curl -X PUT http://localhost:8090/api/dk/compose/STACK_ID \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"yaml_content": "version: \\"3\\"\\nservices:\\n  web:\\n    image: nginx:alpine"}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/compose/STACK_ID", {
  method: "PUT",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    yaml_content: "version: \\"3\\"\\nservices:\\n  web:\\n    image: nginx:alpine",
  }),
});`,
		},
		{
			id: "compose-delete",
			method: "DELETE",
			path: "/api/dk/compose/{id}",
			title: "Delete Stack",
			description:
				"Delete a compose stack record. Does not stop running containers — use Down first.",
			auth: "required",
			capability: "compose.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Compose stack record ID." },
			],
			curlExample: `curl -X DELETE http://localhost:8090/api/dk/compose/STACK_ID \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/compose/STACK_ID", {
  method: "DELETE",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
	],
};

// ── Group 7: Networks ──────────────────────────────────────────────────────

const networks: ApiGroup = {
	id: "networks",
	title: "Networks",
	description: "Docker network creation, removal, and container connectivity management.",
	icon: Network,
	endpoints: [
		{
			id: "networks-create",
			method: "POST",
			path: "/api/dk/networks",
			title: "Create Network",
			description: "Create a new Docker network on a target server.",
			auth: "required",
			capability: "networks.manage",
			requestBody: {
				contentType: "application/json",
				fields: [
					{ name: "server_id", type: "string", required: true, description: "Target server ID." },
					{ name: "name", type: "string", required: true, description: "Network name." },
					{
						name: "driver",
						type: "string",
						required: false,
						description: "Network driver (default: bridge).",
					},
					{
						name: "subnet",
						type: "string",
						required: false,
						description: "Subnet in CIDR format (e.g. 172.20.0.0/16).",
					},
					{
						name: "gateway",
						type: "string",
						required: false,
						description: "Gateway IP address (e.g. 172.20.0.1).",
					},
					{
						name: "internal",
						type: "boolean",
						required: false,
						description: "Restrict external access to the network (default: false).",
					},
				],
			},
			curlExample: `curl -X POST http://localhost:8090/api/dk/networks \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"server_id": "SERVER_ID", "name": "my-network", "driver": "bridge", "subnet": "172.20.0.0/16", "gateway": "172.20.0.1"}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/networks", {
  method: "POST",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    server_id: "SERVER_ID",
    name: "my-network",
    driver: "bridge",
    subnet: "172.20.0.0/16",
    gateway: "172.20.0.1",
  }),
});`,
		},
		{
			id: "networks-remove",
			method: "DELETE",
			path: "/api/dk/networks/{id}",
			title: "Remove Network",
			description: "Remove a Docker network. Fails if containers are still connected.",
			auth: "required",
			capability: "networks.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Network record ID." },
			],
			curlExample: `curl -X DELETE http://localhost:8090/api/dk/networks/NETWORK_ID \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/networks/NETWORK_ID", {
  method: "DELETE",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
		{
			id: "networks-connect",
			method: "POST",
			path: "/api/dk/networks/{id}/connect",
			title: "Connect Container",
			description: "Connect a container to this network.",
			auth: "required",
			capability: "networks.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Network record ID." },
			],
			requestBody: {
				contentType: "application/json",
				fields: [
					{
						name: "container_id",
						type: "string",
						required: true,
						description: "Container record ID to connect.",
					},
				],
			},
			curlExample: `curl -X POST http://localhost:8090/api/dk/networks/NETWORK_ID/connect \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"container_id": "CONTAINER_ID"}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/networks/NETWORK_ID/connect", {
  method: "POST",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ container_id: "CONTAINER_ID" }),
});`,
		},
		{
			id: "networks-disconnect",
			method: "POST",
			path: "/api/dk/networks/{id}/disconnect",
			title: "Disconnect Container",
			description: "Disconnect a container from this network.",
			auth: "required",
			capability: "networks.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Network record ID." },
			],
			requestBody: {
				contentType: "application/json",
				fields: [
					{
						name: "container_id",
						type: "string",
						required: true,
						description: "Container record ID to disconnect.",
					},
					{
						name: "force",
						type: "boolean",
						required: false,
						description: "Force disconnection even if the container is running (default: false).",
					},
				],
			},
			curlExample: `curl -X POST http://localhost:8090/api/dk/networks/NETWORK_ID/disconnect \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"container_id": "CONTAINER_ID", "force": true}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/networks/NETWORK_ID/disconnect", {
  method: "POST",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ container_id: "CONTAINER_ID", force: true }),
});`,
		},
	],
};

// ── Group 8: Volumes ───────────────────────────────────────────────────────

const volumes: ApiGroup = {
	id: "volumes",
	title: "Volumes",
	description: "Docker volume creation, removal, and inspection.",
	icon: Database,
	endpoints: [
		{
			id: "volumes-create",
			method: "POST",
			path: "/api/dk/volumes",
			title: "Create Volume",
			description: "Create a new Docker volume on a target server.",
			auth: "required",
			capability: "volumes.manage",
			requestBody: {
				contentType: "application/json",
				fields: [
					{ name: "server_id", type: "string", required: true, description: "Target server ID." },
					{ name: "name", type: "string", required: true, description: "Volume name." },
					{
						name: "driver",
						type: "string",
						required: false,
						description: "Volume driver (default: local).",
					},
				],
			},
			curlExample: `curl -X POST http://localhost:8090/api/dk/volumes \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"server_id": "SERVER_ID", "name": "my-volume"}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/volumes", {
  method: "POST",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ server_id: "SERVER_ID", name: "my-volume" }),
});`,
		},
		{
			id: "volumes-remove",
			method: "DELETE",
			path: "/api/dk/volumes/{id}",
			title: "Remove Volume",
			description: "Remove a Docker volume. Fails if the volume is currently in use.",
			auth: "required",
			capability: "volumes.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Volume record ID." },
			],
			curlExample: `curl -X DELETE http://localhost:8090/api/dk/volumes/VOLUME_ID \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/volumes/VOLUME_ID", {
  method: "DELETE",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
		{
			id: "volumes-inspect",
			method: "GET",
			path: "/api/dk/volumes/{id}/inspect",
			title: "Inspect Volume",
			description: "Retrieve detailed information about a volume including mount point and usage.",
			auth: "required",
			capability: "containers.view",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Volume record ID." },
			],
			curlExample: `curl http://localhost:8090/api/dk/volumes/VOLUME_ID/inspect \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/volumes/VOLUME_ID/inspect", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const info = await res.json();`,
		},
	],
};

// ── Group 9: Backups ───────────────────────────────────────────────────────

const backups: ApiGroup = {
	id: "backups",
	title: "Backups",
	description: "Backup policy management, manual triggers, records, and restore operations.",
	icon: Archive,
	endpoints: [
		{
			id: "backups-policies-list",
			method: "GET",
			path: "/api/dk/backups/policies",
			title: "List Backup Policies",
			description: "Retrieve all backup policies with their schedules and retention settings.",
			auth: "required",
			capability: "backups.view",
			curlExample: `curl http://localhost:8090/api/dk/backups/policies \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/backups/policies", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const policies = await res.json();`,
		},
		{
			id: "backups-policies-create",
			method: "POST",
			path: "/api/dk/backups/policies",
			title: "Create Backup Policy",
			description: "Create a new backup policy with a cron schedule and retention period.",
			auth: "required",
			capability: "backups.manage",
			requestBody: {
				contentType: "application/json",
				fields: [
					{ name: "server_id", type: "string", required: true, description: "Target server ID." },
					{
						name: "volume",
						type: "string",
						required: true,
						description: "Volume name to back up.",
					},
					{
						name: "schedule",
						type: "string",
						required: true,
						description: "Cron schedule expression.",
						example: "0 2 * * *",
					},
					{
						name: "retention_days",
						type: "number",
						required: false,
						description: "Days to keep backups (default: 30).",
					},
					{
						name: "s3_bucket",
						type: "string",
						required: false,
						description: "S3 bucket for remote storage.",
					},
					{
						name: "s3_prefix",
						type: "string",
						required: false,
						description: "S3 key prefix for backups.",
					},
					{
						name: "enabled",
						type: "boolean",
						required: false,
						description: "Whether the policy is active.",
					},
				],
			},
			curlExample: `curl -X POST http://localhost:8090/api/dk/backups/policies \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"server_id": "SERVER_ID", "volume": "my-data", "schedule": "0 2 * * *", "retention_days": 30}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/backups/policies", {
  method: "POST",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    server_id: "SERVER_ID",
    volume: "my-data",
    schedule: "0 2 * * *",
    retention_days: 30,
  }),
});`,
		},
		{
			id: "backups-policies-update",
			method: "PUT",
			path: "/api/dk/backups/policies/{id}",
			title: "Update Backup Policy",
			description: "Update an existing backup policy's schedule, retention, or storage settings.",
			auth: "required",
			capability: "backups.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Policy record ID." },
			],
			curlExample: `curl -X PUT http://localhost:8090/api/dk/backups/policies/POLICY_ID \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"cron": "0 3 * * *", "retention_days": 60}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/backups/policies/POLICY_ID", {
  method: "PUT",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ cron: "0 3 * * *", retention_days: 60 }),
});`,
		},
		{
			id: "backups-policies-delete",
			method: "DELETE",
			path: "/api/dk/backups/policies/{id}",
			title: "Delete Backup Policy",
			description: "Delete a backup policy. Existing backup records are preserved.",
			auth: "required",
			capability: "backups.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Policy record ID." },
			],
			curlExample: `curl -X DELETE http://localhost:8090/api/dk/backups/policies/POLICY_ID \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/backups/policies/POLICY_ID", {
  method: "DELETE",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
		{
			id: "backups-policies-trigger",
			method: "POST",
			path: "/api/dk/backups/policies/{id}/trigger",
			title: "Trigger Backup",
			description:
				"Manually trigger an immediate backup for a policy, bypassing the cron schedule.",
			auth: "required",
			capability: "backups.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Policy record ID." },
			],
			curlExample: `curl -X POST http://localhost:8090/api/dk/backups/policies/POLICY_ID/trigger \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/backups/policies/POLICY_ID/trigger", {
  method: "POST",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
		{
			id: "backups-records-list",
			method: "GET",
			path: "/api/dk/backups/records",
			title: "List Backup Records",
			description: "Retrieve all backup records with their status, size, and policy info.",
			auth: "required",
			capability: "backups.view",
			responseExample: json([
				{
					id: "bkp_01",
					policy_id: "pol_01",
					status: "completed",
					size: 52428800,
					created: "2026-03-29T02:00:00Z",
				},
			]),
			curlExample: `curl http://localhost:8090/api/dk/backups/records \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/backups/records", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const records = await res.json();`,
		},
		{
			id: "backups-records-restore",
			method: "POST",
			path: "/api/dk/backups/records/{id}/restore",
			title: "Restore Backup",
			description:
				"Restore a container from a specific backup record. The container will be stopped during restore.",
			auth: "required",
			capability: "backups.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Backup record ID." },
			],
			curlExample: `curl -X POST http://localhost:8090/api/dk/backups/records/BACKUP_ID/restore \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/backups/records/BACKUP_ID/restore", {
  method: "POST",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
		{
			id: "backups-cleanup",
			method: "POST",
			path: "/api/dk/backups/cleanup",
			title: "Cleanup Expired Backups",
			description:
				"Manually trigger cleanup of backup records that exceed their policy's retention period.",
			auth: "required",
			capability: "backups.manage",
			curlExample: `curl -X POST http://localhost:8090/api/dk/backups/cleanup \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/backups/cleanup", {
  method: "POST",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const { deleted } = await res.json();`,
		},
	],
};

// ── Export ──────────────────────────────────────────────────────────────────

// ── Group 10: Alerts & Blackout ─────────────────────────────────────────────

const alertsBlackout: ApiGroup = {
	id: "alerts-blackout",
	title: "Alerts & Blackout",
	description: "Alert rule management, toggle, and blackout window scheduling.",
	icon: Bell,
	endpoints: [
		{
			id: "alerts-list",
			method: "GET",
			path: "/api/dk/alerts",
			title: "List Alert Rules",
			description:
				"Retrieve all configured alert rules with their current state (OK, FIRING, RESOLVED).",
			auth: "required",
			capability: "alerts.view",
			responseExample: json([
				{
					id: "alr_01",
					name: "High CPU",
					metric: "cpu_percent",
					threshold: 90,
					state: "OK",
					enabled: true,
				},
			]),
			curlExample: `curl http://localhost:8090/api/dk/alerts \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/alerts", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const rules = await res.json();`,
		},
		{
			id: "alerts-create",
			method: "POST",
			path: "/api/dk/alerts",
			title: "Create Alert Rule",
			description: "Create a new alert rule with metric, threshold, and notification channel.",
			auth: "required",
			capability: "alerts.manage",
			requestBody: {
				contentType: "application/json",
				fields: [
					{ name: "name", type: "string", required: true, description: "Alert rule name." },
					{
						name: "rule_type",
						type: "string",
						required: false,
						description: 'Rule type: "metric" or "event" (default: "metric").',
					},
					{
						name: "metric",
						type: "string",
						required: false,
						description:
							"Metric to monitor (required for metric rules: cpu_percent, memory_percent, disk_percent).",
					},
					{
						name: "operator",
						type: "string",
						required: false,
						description: "Comparison operator: gt, lt, gte, lte, eq (required for metric rules).",
					},
					{
						name: "threshold",
						type: "number",
						required: false,
						description: "Threshold value to trigger alert (required for metric rules).",
					},
					{
						name: "duration",
						type: "number",
						required: false,
						description: "Seconds the condition must hold before firing (default: 60).",
					},
					{
						name: "target_type",
						type: "string",
						required: false,
						description: 'Target scope: "server" or "container".',
					},
					{
						name: "target_id",
						type: "string",
						required: false,
						description: "Target server or container ID.",
					},
					{
						name: "event_type",
						type: "string",
						required: false,
						description: "Event type to watch (required for event rules).",
					},
					{
						name: "scope",
						type: "string",
						required: false,
						description: 'Event scope (default: "all").',
					},
					{
						name: "active_delay",
						type: "number",
						required: false,
						description: "Seconds to wait before activating (default: 0).",
					},
					{
						name: "clear_delay",
						type: "number",
						required: false,
						description: "Seconds after condition clears to resolve (default: 0).",
					},
					{
						name: "notification_cooldown",
						type: "number",
						required: false,
						description: "Min seconds between notifications (default: 300).",
					},
					{
						name: "suppress_during_updates",
						type: "boolean",
						required: false,
						description: "Suppress alerts during container updates.",
					},
					{
						name: "template",
						type: "string",
						required: false,
						description: "Custom notification message template.",
					},
					{
						name: "channels",
						type: "string[]",
						required: false,
						description: "Notification channel IDs.",
					},
					{
						name: "enabled",
						type: "boolean",
						required: false,
						description: "Whether the rule is enabled (default: true).",
					},
				],
			},
			curlExample: `curl -X POST http://localhost:8090/api/dk/alerts \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"name": "High CPU", "rule_type": "metric", "metric": "cpu_percent", "operator": "gt", "threshold": 90, "duration": 60}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/alerts", {
  method: "POST",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    name: "High CPU",
    rule_type: "metric",
    metric: "cpu_percent",
    operator: "gt",
    threshold: 90,
    duration: 60,
  }),
});`,
		},
		{
			id: "alerts-update",
			method: "PUT",
			path: "/api/dk/alerts/{id}",
			title: "Update Alert Rule",
			description: "Update an existing alert rule's metric, threshold, or channels.",
			auth: "required",
			capability: "alerts.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Alert rule record ID." },
			],
			curlExample: `curl -X PUT http://localhost:8090/api/dk/alerts/ALERT_ID \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"threshold": 95}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/alerts/ALERT_ID", {
  method: "PUT",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ threshold: 95 }),
});`,
		},
		{
			id: "alerts-delete",
			method: "DELETE",
			path: "/api/dk/alerts/{id}",
			title: "Delete Alert Rule",
			description: "Delete an alert rule permanently.",
			auth: "required",
			capability: "alerts.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Alert rule record ID." },
			],
			curlExample: `curl -X DELETE http://localhost:8090/api/dk/alerts/ALERT_ID \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/alerts/ALERT_ID", {
  method: "DELETE",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
		{
			id: "alerts-toggle",
			method: "POST",
			path: "/api/dk/alerts/{id}/toggle",
			title: "Toggle Alert Rule",
			description: "Enable or disable an alert rule without deleting it.",
			auth: "required",
			capability: "alerts.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Alert rule record ID." },
			],
			curlExample: `curl -X POST http://localhost:8090/api/dk/alerts/ALERT_ID/toggle \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/alerts/ALERT_ID/toggle", {
  method: "POST",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
		{
			id: "blackout-list",
			method: "GET",
			path: "/api/dk/blackout-windows",
			title: "List Blackout Windows",
			description: "Retrieve all blackout windows. During blackout, alerts are suppressed.",
			auth: "required",
			capability: "alerts.view",
			curlExample: `curl http://localhost:8090/api/dk/blackout-windows \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/blackout-windows", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const windows = await res.json();`,
		},
		{
			id: "blackout-create",
			method: "POST",
			path: "/api/dk/blackout-windows",
			title: "Create Blackout Window",
			description: "Create a blackout window to suppress alerts during maintenance.",
			auth: "required",
			capability: "alerts.manage",
			requestBody: {
				contentType: "application/json",
				fields: [
					{ name: "name", type: "string", required: true, description: "Blackout window name." },
					{
						name: "start_time",
						type: "string",
						required: true,
						description: "Start time (ISO 8601).",
					},
					{ name: "end_time", type: "string", required: true, description: "End time (ISO 8601)." },
					{
						name: "recurring",
						type: "boolean",
						required: false,
						description: "Whether this window recurs (default: false).",
					},
					{
						name: "recurrence_rule",
						type: "string",
						required: false,
						description: "Recurrence rule (e.g. RRULE:FREQ=WEEKLY;BYDAY=SA).",
					},
					{
						name: "duration_minutes",
						type: "number",
						required: false,
						description: "Duration in minutes (used with recurring windows).",
					},
					{
						name: "server_ids",
						type: "string[]",
						required: false,
						description: "Affected servers (omit for all).",
					},
					{
						name: "enabled",
						type: "boolean",
						required: false,
						description: "Whether this window is active (default: true).",
					},
				],
			},
			curlExample: `curl -X POST http://localhost:8090/api/dk/blackout-windows \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"name": "Deploy window", "start_time": "2026-04-01T02:00:00Z", "end_time": "2026-04-01T04:00:00Z", "enabled": true}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/blackout-windows", {
  method: "POST",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    name: "Deploy window",
    start_time: "2026-04-01T02:00:00Z",
    end_time: "2026-04-01T04:00:00Z",
    enabled: true,
  }),
});`,
		},
		{
			id: "blackout-update",
			method: "PUT",
			path: "/api/dk/blackout-windows/{id}",
			title: "Update Blackout Window",
			description: "Update the schedule or scope of a blackout window.",
			auth: "required",
			capability: "alerts.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Blackout window record ID." },
			],
			curlExample: `curl -X PUT http://localhost:8090/api/dk/blackout-windows/BLACKOUT_ID \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"end": "2026-04-01T06:00:00Z"}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/blackout-windows/BLACKOUT_ID", {
  method: "PUT",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ end: "2026-04-01T06:00:00Z" }),
});`,
		},
		{
			id: "blackout-delete",
			method: "DELETE",
			path: "/api/dk/blackout-windows/{id}",
			title: "Delete Blackout Window",
			description: "Delete a blackout window.",
			auth: "required",
			capability: "alerts.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Blackout window record ID." },
			],
			curlExample: `curl -X DELETE http://localhost:8090/api/dk/blackout-windows/BLACKOUT_ID \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/blackout-windows/BLACKOUT_ID", {
  method: "DELETE",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
	],
};

// ── Group 11: Notifications ────────────────────────────────────────────────

const notifications: ApiGroup = {
	id: "notifications",
	title: "Notifications",
	description: "Notification channel management (WhatsApp, Telegram, Webhook) and delivery logs.",
	icon: Send,
	endpoints: [
		{
			id: "notifications-channels-list",
			method: "GET",
			path: "/api/dk/notifications/channels",
			title: "List Channels",
			description: "Retrieve all configured notification channels.",
			auth: "required",
			capability: "notifications.view",
			curlExample: `curl http://localhost:8090/api/dk/notifications/channels \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/notifications/channels", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const channels = await res.json();`,
		},
		{
			id: "notifications-channels-create",
			method: "POST",
			path: "/api/dk/notifications/channels",
			title: "Create Channel",
			description:
				"Create a new notification channel (WhatsApp, Telegram, Webhook, Discord, Slack, SMTP, or Ntfy).",
			auth: "required",
			capability: "notifications.manage",
			requestBody: {
				contentType: "application/json",
				fields: [
					{ name: "name", type: "string", required: true, description: "Channel display name." },
					{
						name: "type",
						type: "string",
						required: true,
						description:
							"Channel type: whatsapp, telegram, webhook, discord, slack, smtp, or ntfy.",
					},
					{
						name: "config",
						type: "object",
						required: true,
						description: "Type-specific configuration (e.g. phone, bot_token, url).",
					},
					{
						name: "enabled",
						type: "boolean",
						required: false,
						description: "Whether the channel is active (default: true).",
					},
				],
			},
			curlExample: `curl -X POST http://localhost:8090/api/dk/notifications/channels \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"name": "Ops Webhook", "type": "webhook", "config": {"url": "https://hooks.example.com/alert"}, "enabled": true}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/notifications/channels", {
  method: "POST",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    name: "Ops Webhook",
    type: "webhook",
    config: { url: "https://hooks.example.com/alert" },
    enabled: true,
  }),
});`,
		},
		{
			id: "notifications-channels-update",
			method: "PUT",
			path: "/api/dk/notifications/channels/{id}",
			title: "Update Channel",
			description: "Update a notification channel's name or configuration.",
			auth: "required",
			capability: "notifications.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Channel record ID." },
			],
			curlExample: `curl -X PUT http://localhost:8090/api/dk/notifications/channels/CHANNEL_ID \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"name": "Ops Webhook v2"}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/notifications/channels/CHANNEL_ID", {
  method: "PUT",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ name: "Ops Webhook v2" }),
});`,
		},
		{
			id: "notifications-channels-delete",
			method: "DELETE",
			path: "/api/dk/notifications/channels/{id}",
			title: "Delete Channel",
			description:
				"Delete a notification channel. Active alert rules referencing this channel will stop receiving notifications.",
			auth: "required",
			capability: "notifications.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Channel record ID." },
			],
			curlExample: `curl -X DELETE http://localhost:8090/api/dk/notifications/channels/CHANNEL_ID \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/notifications/channels/CHANNEL_ID", {
  method: "DELETE",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
		{
			id: "notifications-channels-test",
			method: "POST",
			path: "/api/dk/notifications/channels/{id}/test",
			title: "Test Channel",
			description: "Send a test notification to verify channel configuration.",
			auth: "required",
			capability: "notifications.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Channel record ID." },
			],
			curlExample: `curl -X POST http://localhost:8090/api/dk/notifications/channels/CHANNEL_ID/test \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/notifications/channels/CHANNEL_ID/test", {
  method: "POST",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
		{
			id: "notifications-logs",
			method: "GET",
			path: "/api/dk/notifications/logs",
			title: "Notification Logs",
			description: "Retrieve notification delivery logs with status, channel, and timestamp.",
			auth: "required",
			capability: "notifications.view",
			responseExample: json([
				{
					id: "nl_01",
					channel_id: "ch_01",
					alert_rule_id: "alr_01",
					status: "delivered",
					sent_at: "2026-03-29T02:00:00Z",
				},
			]),
			curlExample: `curl http://localhost:8090/api/dk/notifications/logs \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/notifications/logs", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const logs = await res.json();`,
		},
	],
};

// ── Group 12: Health Checks ────────────────────────────────────────────────

const healthChecks: ApiGroup = {
	id: "health-checks",
	title: "Health Checks",
	description: "HTTP/TCP health check configuration, execution logs, and manual triggers.",
	icon: Activity,
	endpoints: [
		{
			id: "health-checks-list",
			method: "GET",
			path: "/api/dk/health-checks",
			title: "List Health Checks",
			description: "Retrieve all health check configurations with their last status.",
			auth: "required",
			capability: "health.view",
			curlExample: `curl http://localhost:8090/api/dk/health-checks \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/health-checks", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const checks = await res.json();`,
		},
		{
			id: "health-checks-create",
			method: "POST",
			path: "/api/dk/health-checks",
			title: "Create Health Check",
			description: "Create a new health check with URL, interval, and expected status.",
			auth: "required",
			capability: "health.manage",
			requestBody: {
				contentType: "application/json",
				fields: [
					{ name: "name", type: "string", required: true, description: "Health check name." },
					{
						name: "container",
						type: "string",
						required: true,
						description: "Container ID to monitor.",
					},
					{
						name: "server",
						type: "string",
						required: true,
						description: "Server ID where the container runs.",
					},
					{
						name: "url_path",
						type: "string",
						required: false,
						description: "HTTP path to check (e.g. /health).",
					},
					{
						name: "port",
						type: "number",
						required: true,
						description: "Port to check.",
					},
					{
						name: "expected_status",
						type: "number",
						required: false,
						description: "Expected HTTP status (default: 200).",
					},
					{
						name: "interval_seconds",
						type: "number",
						required: false,
						description: "Check interval in seconds (default: 60).",
					},
					{
						name: "timeout_seconds",
						type: "number",
						required: false,
						description: "Request timeout (default: 10).",
					},
					{
						name: "failure_threshold",
						type: "number",
						required: false,
						description: "Consecutive failures before marking unhealthy.",
					},
					{
						name: "success_threshold",
						type: "number",
						required: false,
						description: "Consecutive successes before marking healthy.",
					},
					{
						name: "auto_restart",
						type: "boolean",
						required: false,
						description: "Auto-restart container on failure.",
					},
					{
						name: "max_restarts",
						type: "number",
						required: false,
						description: "Max auto-restarts allowed.",
					},
					{
						name: "restart_window_minutes",
						type: "number",
						required: false,
						description: "Time window for max restart count.",
					},
					{
						name: "enabled",
						type: "boolean",
						required: false,
						description: "Whether the check is active.",
					},
				],
			},
			curlExample: `curl -X POST http://localhost:8090/api/dk/health-checks \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"name": "API Health", "container": "CTR_ID", "server": "SRV_ID", "port": 8080, "url_path": "/health"}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/health-checks", {
  method: "POST",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    name: "API Health",
    container: "CTR_ID",
    server: "SRV_ID",
    port: 8080,
    url_path: "/health",
  }),
});`,
		},
		{
			id: "health-checks-update",
			method: "PUT",
			path: "/api/dk/health-checks/{id}",
			title: "Update Health Check",
			description: "Update a health check's URL, interval, or expected status.",
			auth: "required",
			capability: "health.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Health check record ID." },
			],
			curlExample: `curl -X PUT http://localhost:8090/api/dk/health-checks/CHECK_ID \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"interval_seconds": 15}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/health-checks/CHECK_ID", {
  method: "PUT",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ interval_seconds: 15 }),
});`,
		},
		{
			id: "health-checks-delete",
			method: "DELETE",
			path: "/api/dk/health-checks/{id}",
			title: "Delete Health Check",
			description: "Delete a health check and its execution logs.",
			auth: "required",
			capability: "health.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Health check record ID." },
			],
			curlExample: `curl -X DELETE http://localhost:8090/api/dk/health-checks/CHECK_ID \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/health-checks/CHECK_ID", {
  method: "DELETE",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
		{
			id: "health-checks-logs",
			method: "GET",
			path: "/api/dk/health-checks/{id}/logs",
			title: "Get Execution Logs",
			description: "Retrieve execution history for a specific health check.",
			auth: "required",
			capability: "health.view",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Health check record ID." },
			],
			curlExample: `curl http://localhost:8090/api/dk/health-checks/CHECK_ID/logs \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/health-checks/CHECK_ID/logs", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const logs = await res.json();`,
		},
		{
			id: "health-checks-trigger",
			method: "POST",
			path: "/api/dk/health-checks/{id}/trigger",
			title: "Trigger Health Check",
			description: "Manually trigger a health check execution, bypassing the interval schedule.",
			auth: "required",
			capability: "health.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Health check record ID." },
			],
			curlExample: `curl -X POST http://localhost:8090/api/dk/health-checks/CHECK_ID/trigger \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/health-checks/CHECK_ID/trigger", {
  method: "POST",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
	],
};

// ── Group 13: Users & Access ───────────────────────────────────────────────

const usersAccess: ApiGroup = {
	id: "users-access",
	title: "Users & Access",
	description: "User management, RBAC groups, group membership, and API key operations.",
	icon: Shield,
	endpoints: [
		{
			id: "users-list",
			method: "GET",
			path: "/api/dk/users",
			title: "List Users",
			description: "Retrieve all user accounts with their group memberships.",
			auth: "required",
			capability: "users.view",
			curlExample: `curl http://localhost:8090/api/dk/users \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/users", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const users = await res.json();`,
		},
		{
			id: "users-create",
			method: "POST",
			path: "/api/dk/users",
			title: "Create User",
			description: "Create a new user account.",
			auth: "required",
			capability: "users.manage",
			requestBody: {
				contentType: "application/json",
				fields: [
					{ name: "email", type: "string", required: true, description: "User email address." },
					{
						name: "password",
						type: "string",
						required: true,
						description: "User password (min 8 chars).",
					},
					{
						name: "name",
						type: "string",
						required: false,
						description: "User display name.",
					},
				],
			},
			curlExample: `curl -X POST http://localhost:8090/api/dk/users \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"email": "user@example.com", "password": "securepass", "name": "John"}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/users", {
  method: "POST",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    email: "user@example.com",
    password: "securepass",
    name: "John",
  }),
});`,
		},
		{
			id: "users-update",
			method: "PATCH",
			path: "/api/dk/users/{id}",
			title: "Update User",
			description: "Update a user's email, name, password, or enabled status.",
			auth: "required",
			capability: "users.manage",
			parameters: [{ name: "id", type: "string", required: true, description: "User record ID." }],
			requestBody: {
				contentType: "application/json",
				fields: [
					{ name: "email", type: "string", required: false, description: "New email address." },
					{ name: "name", type: "string", required: false, description: "New display name." },
					{ name: "password", type: "string", required: false, description: "New password." },
					{
						name: "enabled",
						type: "boolean",
						required: false,
						description: "Enable or disable user.",
					},
				],
			},
			curlExample: `curl -X PATCH http://localhost:8090/api/dk/users/USER_ID \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"enabled": true}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/users/USER_ID", {
  method: "PATCH",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ enabled: true }),
});`,
		},
		{
			id: "users-delete",
			method: "DELETE",
			path: "/api/dk/users/{id}",
			title: "Delete User",
			description: "Delete a user account and remove all group memberships.",
			auth: "required",
			capability: "users.manage",
			parameters: [{ name: "id", type: "string", required: true, description: "User record ID." }],
			curlExample: `curl -X DELETE http://localhost:8090/api/dk/users/USER_ID \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/users/USER_ID", {
  method: "DELETE",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
		{
			id: "groups-list",
			method: "GET",
			path: "/api/dk/groups",
			title: "List Groups",
			description: "Retrieve all RBAC groups with their capabilities.",
			auth: "required",
			capability: "users.view",
			responseExample: json([
				{
					id: "grp_01",
					name: "Operators",
					capabilities: ["servers.view", "containers.view", "containers.manage"],
				},
			]),
			curlExample: `curl http://localhost:8090/api/dk/groups \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/groups", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const groups = await res.json();`,
		},
		{
			id: "groups-create",
			method: "POST",
			path: "/api/dk/groups",
			title: "Create Group",
			description: "Create a new RBAC group with a set of capabilities.",
			auth: "required",
			capability: "users.manage",
			requestBody: {
				contentType: "application/json",
				fields: [
					{ name: "name", type: "string", required: true, description: "Group name." },
					{
						name: "description",
						type: "string",
						required: false,
						description: "Group description.",
					},
					{
						name: "capabilities",
						type: "string[]",
						required: false,
						description: "List of capability strings.",
					},
				],
			},
			curlExample: `curl -X POST http://localhost:8090/api/dk/groups \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"name": "Read Only", "capabilities": ["servers.view", "containers.view"]}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/groups", {
  method: "POST",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    name: "Read Only",
    capabilities: ["servers.view", "containers.view"],
  }),
});`,
		},
		{
			id: "groups-update",
			method: "PUT",
			path: "/api/dk/groups/{id}",
			title: "Update Group",
			description: "Update a group's name or capabilities.",
			auth: "required",
			capability: "users.manage",
			parameters: [{ name: "id", type: "string", required: true, description: "Group record ID." }],
			curlExample: `curl -X PUT http://localhost:8090/api/dk/groups/GROUP_ID \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"capabilities": ["servers.view", "containers.view", "images.view"]}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/groups/GROUP_ID", {
  method: "PUT",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    capabilities: ["servers.view", "containers.view", "images.view"],
  }),
});`,
		},
		{
			id: "groups-delete",
			method: "DELETE",
			path: "/api/dk/groups/{id}",
			title: "Delete Group",
			description: "Delete a group and remove all member associations.",
			auth: "required",
			capability: "users.manage",
			parameters: [{ name: "id", type: "string", required: true, description: "Group record ID." }],
			curlExample: `curl -X DELETE http://localhost:8090/api/dk/groups/GROUP_ID \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/groups/GROUP_ID", {
  method: "DELETE",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
		{
			id: "groups-add-member",
			method: "POST",
			path: "/api/dk/groups/{id}/members",
			title: "Add Group Member",
			description: "Add a user to a group, granting them the group's capabilities.",
			auth: "required",
			capability: "users.manage",
			parameters: [{ name: "id", type: "string", required: true, description: "Group record ID." }],
			requestBody: {
				contentType: "application/json",
				fields: [
					{
						name: "user_id",
						type: "string",
						required: true,
						description: "User record ID to add.",
					},
				],
			},
			curlExample: `curl -X POST http://localhost:8090/api/dk/groups/GROUP_ID/members \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"user_id": "USER_ID"}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/groups/GROUP_ID/members", {
  method: "POST",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ user_id: "USER_ID" }),
});`,
		},
		{
			id: "groups-remove-member",
			method: "DELETE",
			path: "/api/dk/groups/{id}/members/{userId}",
			title: "Remove Group Member",
			description: "Remove a user from a group, revoking the group's capabilities.",
			auth: "required",
			capability: "users.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Group record ID." },
				{
					name: "userId",
					type: "string",
					required: true,
					description: "User record ID to remove.",
				},
			],
			curlExample: `curl -X DELETE http://localhost:8090/api/dk/groups/GROUP_ID/members/USER_ID \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/groups/GROUP_ID/members/USER_ID", {
  method: "DELETE",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
		{
			id: "api-keys-list",
			method: "GET",
			path: "/api/dk/api-keys",
			title: "List API Keys",
			description: "Retrieve all API keys (hashed, not the raw key value).",
			auth: "required",
			capability: "users.view",
			curlExample: `curl http://localhost:8090/api/dk/api-keys \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/api-keys", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const keys = await res.json();`,
		},
		{
			id: "api-keys-create",
			method: "POST",
			path: "/api/dk/api-keys",
			title: "Create API Key",
			description:
				"Generate a new API key with specific capabilities. The raw key is returned only once.",
			auth: "required",
			capability: "users.manage",
			requestBody: {
				contentType: "application/json",
				fields: [
					{ name: "name", type: "string", required: true, description: "Key display name." },
					{
						name: "capabilities",
						type: "string[]",
						required: false,
						description: "List of capability strings the key grants.",
					},
					{
						name: "expires_at",
						type: "string",
						required: false,
						description: "Expiration date in RFC3339 format.",
					},
				],
			},
			responseExample: json({
				id: "ak_01",
				name: "CI Pipeline",
				key: "dk_abc123...",
				capabilities: ["containers.view", "containers.manage"],
			}),
			curlExample: `curl -X POST http://localhost:8090/api/dk/api-keys \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"name": "CI Pipeline", "capabilities": ["containers.view", "containers.manage"]}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/api-keys", {
  method: "POST",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ name: "CI Pipeline", capabilities: ["containers.view", "containers.manage"] }),
});
const { key } = await res.json(); // Save this — shown only once`,
		},
		{
			id: "api-keys-delete",
			method: "DELETE",
			path: "/api/dk/api-keys/{id}",
			title: "Delete API Key",
			description: "Revoke and delete an API key permanently.",
			auth: "required",
			capability: "users.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "API key record ID." },
			],
			curlExample: `curl -X DELETE http://localhost:8090/api/dk/api-keys/KEY_ID \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/api-keys/KEY_ID", {
  method: "DELETE",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
	],
};

// ── Group 14: Tags ─────────────────────────────────────────────────────────

const tags: ApiGroup = {
	id: "tags",
	title: "Tags",
	description: "Container tagging for organization, filtering, and batch operations.",
	icon: Tag,
	endpoints: [
		{
			id: "tags-list",
			method: "GET",
			path: "/api/dk/tags",
			title: "List Tag Assignments",
			description: "Retrieve all tag-to-container assignments.",
			auth: "required",
			capability: "containers.view",
			curlExample: `curl http://localhost:8090/api/dk/tags \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/tags", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const tags = await res.json();`,
		},
		{
			id: "tags-names",
			method: "GET",
			path: "/api/dk/tags/names",
			title: "List Tag Names",
			description: "Retrieve all unique tag names for autocomplete and filtering.",
			auth: "required",
			capability: "containers.view",
			curlExample: `curl http://localhost:8090/api/dk/tags/names \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/tags/names", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const names = await res.json();`,
		},
		{
			id: "tags-create",
			method: "POST",
			path: "/api/dk/tags",
			title: "Assign Tag",
			description: "Assign a tag to a resource (container or server).",
			auth: "required",
			capability: "containers.manage",
			requestBody: {
				contentType: "application/json",
				fields: [
					{ name: "name", type: "string", required: true, description: "Tag name." },
					{ name: "color", type: "string", required: false, description: "Tag color (hex)." },
					{
						name: "resource_type",
						type: "string",
						required: true,
						description: 'Resource type: "container" or "server".',
					},
					{
						name: "resource_id",
						type: "string",
						required: true,
						description: "Resource record ID.",
					},
				],
			},
			curlExample: `curl -X POST http://localhost:8090/api/dk/tags \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"name": "production", "resource_type": "container", "resource_id": "CONTAINER_ID"}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/tags", {
  method: "POST",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ name: "production", resource_type: "container", resource_id: "CONTAINER_ID" }),
});`,
		},
		{
			id: "tags-delete",
			method: "DELETE",
			path: "/api/dk/tags/{id}",
			title: "Remove Tag",
			description: "Remove a tag assignment from a container.",
			auth: "required",
			capability: "containers.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Tag assignment record ID." },
			],
			curlExample: `curl -X DELETE http://localhost:8090/api/dk/tags/TAG_ID \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/tags/TAG_ID", {
  method: "DELETE",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
		{
			id: "tags-bulk-assign",
			method: "POST",
			path: "/api/dk/tags/bulk",
			title: "Bulk Assign Tags",
			description: "Assign a tag to multiple resources at once.",
			auth: "required",
			capability: "containers.manage",
			requestBody: {
				contentType: "application/json",
				fields: [
					{ name: "tag", type: "string", required: true, description: "Tag name to assign." },
					{ name: "color", type: "string", required: false, description: "Tag color (hex)." },
					{
						name: "resource_type",
						type: "string",
						required: true,
						description: 'Resource type: "container" or "server".',
					},
					{
						name: "resource_ids",
						type: "string[]",
						required: true,
						description: "Resource record IDs.",
					},
				],
			},
			curlExample: `curl -X POST http://localhost:8090/api/dk/tags/bulk \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"tag": "staging", "resource_type": "container", "resource_ids": ["CTR_1", "CTR_2"]}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/tags/bulk", {
  method: "POST",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ tag: "staging", resource_type: "container", resource_ids: ["CTR_1", "CTR_2"] }),
});`,
		},
		{
			id: "tags-bulk-remove",
			method: "DELETE",
			path: "/api/dk/tags/bulk",
			title: "Bulk Remove Tags",
			description: "Remove a tag from multiple resources at once.",
			auth: "required",
			capability: "containers.manage",
			requestBody: {
				contentType: "application/json",
				fields: [
					{ name: "tag", type: "string", required: true, description: "Tag name to remove." },
					{
						name: "resource_type",
						type: "string",
						required: true,
						description: 'Resource type: "container" or "server".',
					},
					{
						name: "resource_ids",
						type: "string[]",
						required: true,
						description: "Resource record IDs.",
					},
				],
			},
			curlExample: `curl -X DELETE http://localhost:8090/api/dk/tags/bulk \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"tag": "staging", "resource_type": "container", "resource_ids": ["CTR_1", "CTR_2"]}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/tags/bulk", {
  method: "DELETE",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ tag: "staging", resource_type: "container", resource_ids: ["CTR_1", "CTR_2"] }),
});`,
		},
	],
};

// ── Group 15: Batch Operations ─────────────────────────────────────────────

const batchOperations: ApiGroup = {
	id: "batch-operations",
	title: "Batch Operations",
	description: "Bulk container actions (start, stop, restart, remove) with job tracking.",
	icon: Zap,
	endpoints: [
		{
			id: "batch-submit",
			method: "POST",
			path: "/api/dk/batch",
			title: "Submit Batch Job",
			description: "Submit a batch operation to run an action on multiple containers.",
			auth: "required",
			capability: "containers.manage",
			requestBody: {
				contentType: "application/json",
				fields: [
					{
						name: "action",
						type: "string",
						required: true,
						description: "Action: start, stop, restart, or remove.",
					},
					{
						name: "container_ids",
						type: "string[]",
						required: true,
						description: "Container record IDs.",
					},
					{
						name: "tag_filter",
						type: "string",
						required: false,
						description: "Alternative: target all containers with this tag.",
					},
				],
			},
			curlExample: `curl -X POST http://localhost:8090/api/dk/batch \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"action": "restart", "container_ids": ["CTR_1", "CTR_2", "CTR_3"]}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/batch", {
  method: "POST",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    action: "restart",
    container_ids: ["CTR_1", "CTR_2", "CTR_3"],
  }),
});
const { job_id } = await res.json();`,
		},
		{
			id: "batch-list",
			method: "GET",
			path: "/api/dk/batch",
			title: "List Batch Jobs",
			description: "Retrieve all batch jobs with their status and progress.",
			auth: "required",
			capability: "containers.view",
			curlExample: `curl http://localhost:8090/api/dk/batch \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/batch", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const jobs = await res.json();`,
		},
		{
			id: "batch-get",
			method: "GET",
			path: "/api/dk/batch/{id}",
			title: "Get Batch Job",
			description: "Retrieve a specific batch job with detailed per-container results.",
			auth: "required",
			capability: "containers.view",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Batch job record ID." },
			],
			responseExample: json({
				id: "job_01",
				action: "restart",
				status: "completed",
				total: 3,
				succeeded: 3,
				failed: 0,
			}),
			curlExample: `curl http://localhost:8090/api/dk/batch/JOB_ID \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/batch/JOB_ID", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const job = await res.json();`,
		},
		{
			id: "batch-progress",
			method: "GET",
			path: "/api/dk/batch/{id}/progress",
			title: "Get Batch Progress",
			description: "Get real-time progress updates for a running batch job.",
			streaming: true,
			auth: "required",
			capability: "containers.view",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Batch job record ID." },
			],
			curlExample: `curl http://localhost:8090/api/dk/batch/JOB_ID/progress \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/batch/JOB_ID/progress", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const progress = await res.json();`,
		},
		{
			id: "batch-cancel",
			method: "POST",
			path: "/api/dk/batch/{id}/cancel",
			title: "Cancel Batch Job",
			description: "Cancel a running batch job. Already-completed actions are not reverted.",
			auth: "required",
			capability: "containers.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Batch job record ID." },
			],
			curlExample: `curl -X POST http://localhost:8090/api/dk/batch/JOB_ID/cancel \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/batch/JOB_ID/cancel", {
  method: "POST",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
	],
};

// ── Group 16: Monitoring ───────────────────────────────────────────────────

const monitoring: ApiGroup = {
	id: "monitoring",
	title: "Monitoring",
	description: "Traefik routes, global search, and audit log access.",
	icon: Search,
	endpoints: [
		{
			id: "traefik-routes",
			method: "GET",
			path: "/api/dk/traefik/routes",
			title: "List Traefik Routes",
			description: "Retrieve all discovered Traefik routes (HTTP routers, services, middleware).",
			auth: "required",
			capability: "traefik.view",
			curlExample: `curl http://localhost:8090/api/dk/traefik/routes \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/traefik/routes", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const routes = await res.json();`,
		},
		{
			id: "global-search",
			method: "GET",
			path: "/api/dk/search",
			title: "Global Search",
			description: "Search across servers, containers, images, and compose stacks.",
			auth: "required",
			capability: "containers.view",
			queryParams: [
				{ name: "q", type: "string", required: true, description: "Search query (min 2 chars)." },
			],
			curlExample: `curl "http://localhost:8090/api/dk/search?q=nginx" \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/search?q=nginx", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const results = await res.json();`,
		},
		{
			id: "audit-logs",
			method: "GET",
			path: "/api/dk/audit-logs",
			title: "List Audit Logs",
			description:
				"Retrieve audit log entries with pagination and filtering. Supports filter by collection, action, and user.",
			auth: "required",
			capability: "audit.view",
			queryParams: [
				{ name: "page", type: "number", required: false, description: "Page number (default: 1)." },
				{
					name: "perPage",
					type: "number",
					required: false,
					description: "Items per page (default: 20, max: 100).",
				},
				{
					name: "collection",
					type: "string",
					required: false,
					description: "Filter by collection name.",
				},
				{
					name: "action",
					type: "string",
					required: false,
					description: "Filter by action (create, update, delete).",
				},
			],
			curlExample: `curl "http://localhost:8090/api/dk/audit-logs?page=1&perPage=20" \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/audit-logs?page=1&perPage=20", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const logs = await res.json();`,
		},
	],
};

// ── Group 17: Settings ─────────────────────────────────────────────────────

const settings: ApiGroup = {
	id: "settings",
	title: "Settings",
	description: "Data retention policies and Docker registry credential management.",
	icon: Settings,
	endpoints: [
		{
			id: "settings-retention-get",
			method: "GET",
			path: "/api/dk/settings/retention",
			title: "Get Retention Settings",
			description: "Retrieve current data retention periods for raw, hourly, and daily metrics.",
			auth: "required",
			capability: "settings.view",
			responseExample: json({ raw_hours: 48, hourly_days: 30, daily_days: 365 }),
			curlExample: `curl http://localhost:8090/api/dk/settings/retention \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/settings/retention", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const retention = await res.json();`,
		},
		{
			id: "settings-retention-update",
			method: "PUT",
			path: "/api/dk/settings/retention",
			title: "Update Retention Settings",
			description: "Update data retention periods. Changes take effect on the next cleanup cycle.",
			auth: "required",
			capability: "settings.manage",
			requestBody: {
				contentType: "application/json",
				fields: [
					{
						name: "retention_raw_hours",
						type: "number",
						required: false,
						description: "Hours to keep raw metrics (default: 48).",
					},
					{
						name: "retention_hourly_days",
						type: "number",
						required: false,
						description: "Days to keep hourly aggregates (default: 30).",
					},
					{
						name: "retention_daily_days",
						type: "number",
						required: false,
						description: "Days to keep daily aggregates (default: 365).",
					},
					{
						name: "retention_events_days",
						type: "number",
						required: false,
						description: "Days to keep event records (default: 90).",
					},
					{
						name: "retention_audit_logs_days",
						type: "number",
						required: false,
						description: "Days to keep audit logs (default: 90).",
					},
					{
						name: "retention_notification_logs_days",
						type: "number",
						required: false,
						description: "Days to keep notification logs (default: 90).",
					},
				],
			},
			curlExample: `curl -X PUT http://localhost:8090/api/dk/settings/retention \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"retention_raw_hours": 72, "retention_hourly_days": 60, "retention_audit_logs_days": 180}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/settings/retention", {
  method: "PUT",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    retention_raw_hours: 72,
    retention_hourly_days: 60,
    retention_audit_logs_days: 180,
  }),
});`,
		},
		{
			id: "registry-credentials-list",
			method: "GET",
			path: "/api/dk/registry-credentials",
			title: "List Registry Credentials",
			description: "Retrieve all stored Docker registry credentials (passwords are masked).",
			auth: "required",
			capability: "settings.view",
			curlExample: `curl http://localhost:8090/api/dk/registry-credentials \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/registry-credentials", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const creds = await res.json();`,
		},
		{
			id: "registry-credentials-create",
			method: "POST",
			path: "/api/dk/registry-credentials",
			title: "Add Registry Credential",
			description:
				"Store a Docker registry credential. The password is encrypted with AES-256-GCM.",
			auth: "required",
			capability: "settings.manage",
			requestBody: {
				contentType: "application/json",
				fields: [
					{
						name: "name",
						type: "string",
						required: true,
						description: "Display name for this credential.",
						example: "GitHub Container Registry",
					},
					{
						name: "registry_host",
						type: "string",
						required: true,
						description: "Registry host URL.",
						example: "ghcr.io",
					},
					{ name: "username", type: "string", required: true, description: "Registry username." },
					{
						name: "password",
						type: "string",
						required: true,
						description: "Registry password or access token.",
					},
				],
			},
			curlExample: `curl -X POST http://localhost:8090/api/dk/registry-credentials \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"name": "GitHub CR", "registry_host": "ghcr.io", "username": "user", "password": "ghp_token"}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/registry-credentials", {
  method: "POST",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    name: "GitHub CR",
    registry_host: "ghcr.io",
    username: "user",
    password: "ghp_token",
  }),
});`,
		},
		{
			id: "registry-credentials-update",
			method: "PUT",
			path: "/api/dk/registry-credentials/{id}",
			title: "Update Registry Credential",
			description: "Update an existing registry credential's username or password.",
			auth: "required",
			capability: "settings.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Credential record ID." },
			],
			curlExample: `curl -X PUT http://localhost:8090/api/dk/registry-credentials/CRED_ID \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"password": "new_ghp_token"}'`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/registry-credentials/CRED_ID", {
  method: "PUT",
  headers: {
    "X-API-Key": "YOUR_API_KEY",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ password: "new_ghp_token" }),
});`,
		},
		{
			id: "registry-credentials-delete",
			method: "DELETE",
			path: "/api/dk/registry-credentials/{id}",
			title: "Delete Registry Credential",
			description: "Delete a stored registry credential.",
			auth: "required",
			capability: "settings.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Credential record ID." },
			],
			curlExample: `curl -X DELETE http://localhost:8090/api/dk/registry-credentials/CRED_ID \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/registry-credentials/CRED_ID", {
  method: "DELETE",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
	],
};

// ── Group 18: System ───────────────────────────────────────────────────────

const system: ApiGroup = {
	id: "system",
	title: "System",
	description:
		"Version info, update management, cascade previews, and dashboard layout persistence.",
	icon: Cog,
	endpoints: [
		{
			id: "system-version",
			method: "GET",
			path: "/api/dk/system/version",
			title: "Get Version",
			description: "Retrieve the manager's version and build commit hash. Public endpoint.",
			auth: "public",
			responseExample: json({ version: "0.1.0", commit: "abc1234" }),
			curlExample: `curl http://localhost:8090/api/dk/system/version`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/system/version");
const { version, commit } = await res.json();`,
		},
		{
			id: "system-update-check",
			method: "GET",
			path: "/api/dk/system/update/check",
			title: "Check for Updates",
			description: "Check if newer versions are available for the manager or agents.",
			auth: "required",
			capability: "settings.view",
			curlExample: `curl http://localhost:8090/api/dk/system/update/check \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/system/update/check", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const updates = await res.json();`,
		},
		{
			id: "system-update-manager",
			method: "POST",
			path: "/api/dk/system/update/manager",
			title: "Update Manager",
			description: "Trigger a self-update of the manager binary. The process will restart.",
			streaming: true,
			auth: "required",
			capability: "settings.manage",
			curlExample: `curl -X POST http://localhost:8090/api/dk/system/update/manager \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/system/update/manager", {
  method: "POST",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
		},
		{
			id: "system-update-agent",
			method: "POST",
			path: "/api/dk/system/update/agent/{id}",
			title: "Update Agent",
			description:
				"Push an update to a specific agent. Uses the saga pattern (snapshot → replace → verify → rollback on failure).",
			auth: "required",
			capability: "settings.manage",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Server record ID." },
			],
			curlExample: `curl -X POST http://localhost:8090/api/dk/system/update/agent/SERVER_ID \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/system/update/agent/SERVER_ID", {
  method: "POST",
  headers: { "X-API-Key": "YOUR_API_KEY" },
});`,
			streaming: true,
		},
		{
			id: "cascade-preview-container",
			method: "GET",
			path: "/api/dk/containers/{id}/cascade-preview",
			title: "Container Cascade Preview",
			description:
				"Preview what will be deleted when removing a container (linked tags, backups, metrics).",
			auth: "required",
			capability: "containers.view",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Container record ID." },
			],
			curlExample: `curl http://localhost:8090/api/dk/containers/CONTAINER_ID/cascade-preview \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/containers/CONTAINER_ID/cascade-preview", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const preview = await res.json();`,
		},
		{
			id: "cascade-preview-volume",
			method: "GET",
			path: "/api/dk/volumes/{id}/cascade-preview",
			title: "Volume Cascade Preview",
			description: "Preview what will be affected when removing a volume.",
			auth: "required",
			capability: "containers.view",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Volume record ID." },
			],
			curlExample: `curl http://localhost:8090/api/dk/volumes/VOLUME_ID/cascade-preview \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/volumes/VOLUME_ID/cascade-preview", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const preview = await res.json();`,
		},
		{
			id: "cascade-preview-server",
			method: "GET",
			path: "/api/dk/servers/{id}/cascade-preview",
			title: "Server Cascade Preview",
			description: "Preview all resources that will be deleted when removing a server.",
			auth: "required",
			capability: "servers.view",
			parameters: [
				{ name: "id", type: "string", required: true, description: "Server record ID." },
			],
			responseExample: json({ containers: 5, images: 12, volumes: 3, networks: 2, backups: 8 }),
			curlExample: `curl http://localhost:8090/api/dk/servers/SERVER_ID/cascade-preview \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/servers/SERVER_ID/cascade-preview", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const preview = await res.json();`,
		},
		{
			id: "dashboard-layout-get",
			method: "GET",
			path: "/api/dk/dashboard/layout",
			title: "Get Dashboard Layout",
			description: "Retrieve the saved dashboard widget layout for the current user.",
			auth: "required",
			capability: "containers.view",
			curlExample: `curl http://localhost:8090/api/dk/dashboard/layout \\
  -H "X-API-Key: YOUR_API_KEY"`,
			jsExample: `const res = await fetch("http://localhost:8090/api/dk/dashboard/layout", {
  headers: { "X-API-Key": "YOUR_API_KEY" },
});
const layout = await res.json();`,
		},
	],
};

export const apiGroups: ApiGroup[] = [
	gettingStarted,
	authentication,
	servers,
	containers,
	images,
	compose,
	networks,
	volumes,
	backups,
	alertsBlackout,
	notifications,
	healthChecks,
	usersAccess,
	tags,
	batchOperations,
	monitoring,
	settings,
	system,
];
