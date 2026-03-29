import {
  Archive,
  BookOpen,
  Box,
  HardDrive,
  Key,
  Layers,
  Network,
  Database,
  Server,
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
          { name: "password", type: "string", required: true, description: "Admin password (min 8 chars)." },
          { name: "passwordConfirm", type: "string", required: true, description: "Password confirmation." },
        ],
      },
      curlExample: `curl -X POST http://localhost:8090/api/dk/auth/setup \\
  -H "Content-Type: application/json" \\
  -d '{"email": "admin@example.com", "password": "securepass", "passwordConfirm": "securepass"}'`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/auth/setup", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    email: "admin@example.com",
    password: "securepass",
    passwordConfirm: "securepass",
  }),
});`,
    },
    {
      id: "auth-login",
      method: "POST",
      path: "/api/dk/auth/login",
      title: "Login",
      description: "Authenticate with email and password. Returns an access token (15 min) and a refresh token (7 days).",
      auth: "public",
      requestBody: {
        contentType: "application/json",
        fields: [
          { name: "email", type: "string", required: true, description: "Admin email address.", example: "admin@example.com" },
          { name: "password", type: "string", required: true, description: "Admin password.", example: "your-password" },
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
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/auth/refresh", {
  method: "POST",
  headers: { Authorization: "Bearer YOUR_TOKEN" },
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
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/auth/logout", {
  method: "POST",
  headers: { Authorization: "Bearer YOUR_TOKEN" },
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
          { name: "oldPassword", type: "string", required: true, description: "Current password." },
          { name: "password", type: "string", required: true, description: "New password (min 8 chars)." },
          { name: "passwordConfirm", type: "string", required: true, description: "New password confirmation." },
        ],
      },
      curlExample: `curl -X POST http://localhost:8090/api/dk/auth/password \\
  -H "Authorization: Bearer YOUR_TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{"oldPassword": "old", "password": "new", "passwordConfirm": "new"}'`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/auth/password", {
  method: "POST",
  headers: {
    Authorization: "Bearer YOUR_TOKEN",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    oldPassword: "old",
    password: "new",
    passwordConfirm: "new",
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
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/auth/me", {
  headers: { Authorization: "Bearer YOUR_TOKEN" },
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
      description: "Retrieve all registered servers with their current status, metrics, and agent info.",
      auth: "required",
      capability: "servers.view",
      responseExample: json([
        { id: "srv_01", hostname: "prod-1", status: "online", ip: "10.0.1.10", agent_version: "0.1.0" },
      ]),
      curlExample: `curl http://localhost:8090/api/dk/servers \\
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/servers", {
  headers: { Authorization: "Bearer YOUR_TOKEN" },
});
const servers = await res.json();`,
    },
    {
      id: "servers-provision",
      method: "POST",
      path: "/api/dk/servers/provision",
      title: "Provision Server",
      description: "Create a new server record and generate a one-time enrollment token for agent installation.",
      auth: "required",
      capability: "servers.manage",
      requestBody: {
        contentType: "application/json",
        fields: [
          { name: "hostname", type: "string", required: true, description: "Display name for the server." },
          { name: "description", type: "string", required: false, description: "Optional server description." },
        ],
      },
      curlExample: `curl -X POST http://localhost:8090/api/dk/servers/provision \\
  -H "Authorization: Bearer YOUR_TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{"hostname": "prod-2"}'`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/servers/provision", {
  method: "POST",
  headers: {
    Authorization: "Bearer YOUR_TOKEN",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ hostname: "prod-2" }),
});
const server = await res.json();`,
    },
    {
      id: "servers-install-script",
      method: "GET",
      path: "/api/dk/servers/{id}/install-script",
      title: "Get Install Script",
      description: "Download the agent installation script for a specific server. Includes the enrollment token.",
      auth: "required",
      capability: "servers.manage",
      parameters: [
        { name: "id", type: "string", required: true, description: "Server record ID." },
      ],
      curlExample: `curl http://localhost:8090/api/dk/servers/SERVER_ID/install-script \\
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/servers/SERVER_ID/install-script", {
  headers: { Authorization: "Bearer YOUR_TOKEN" },
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
          { name: "hostname", type: "string", required: false, description: "New display name." },
          { name: "description", type: "string", required: false, description: "New description." },
        ],
      },
      curlExample: `curl -X PATCH http://localhost:8090/api/dk/servers/SERVER_ID \\
  -H "Authorization: Bearer YOUR_TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{"hostname": "prod-1-renamed"}'`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/servers/SERVER_ID", {
  method: "PATCH",
  headers: {
    Authorization: "Bearer YOUR_TOKEN",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ hostname: "prod-1-renamed" }),
});`,
    },
    {
      id: "servers-delete",
      method: "DELETE",
      path: "/api/dk/servers/{id}",
      title: "Delete Server",
      description: "Remove a server and all associated records (containers, metrics, backups). Triggers cascade delete.",
      auth: "required",
      capability: "servers.manage",
      parameters: [
        { name: "id", type: "string", required: true, description: "Server record ID." },
      ],
      curlExample: `curl -X DELETE http://localhost:8090/api/dk/servers/SERVER_ID \\
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/servers/SERVER_ID", {
  method: "DELETE",
  headers: { Authorization: "Bearer YOUR_TOKEN" },
});`,
    },
    {
      id: "servers-quick-install",
      method: "GET",
      path: "/api/dk/servers/quick-install",
      title: "Quick Install Command",
      description: "Get a one-line install command that provisions a new server and installs the agent.",
      auth: "required",
      capability: "servers.manage",
      curlExample: `curl http://localhost:8090/api/dk/servers/quick-install \\
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/servers/quick-install", {
  headers: { Authorization: "Bearer YOUR_TOKEN" },
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
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/enrollment-token", {
  headers: { Authorization: "Bearer YOUR_TOKEN" },
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
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/enrollment-token/regenerate", {
  method: "POST",
  headers: { Authorization: "Bearer YOUR_TOKEN" },
});
const { token } = await res.json();`,
    },
  ],
};

// ── Group 4: Containers ────────────────────────────────────────────────────

const containers: ApiGroup = {
  id: "containers",
  title: "Containers",
  description: "Container lifecycle management, inspection, resource limits, and environment variables.",
  icon: Box,
  endpoints: [
    {
      id: "containers-create",
      method: "POST",
      path: "/api/dk/containers/create",
      title: "Create Container",
      description: "Create and optionally start a new container on a target server. Auto-pulls the image if not present.",
      auth: "required",
      capability: "containers.manage",
      requestBody: {
        contentType: "application/json",
        fields: [
          { name: "server_id", type: "string", required: true, description: "Target server ID." },
          { name: "name", type: "string", required: true, description: "Container name." },
          { name: "image", type: "string", required: true, description: "Docker image reference.", example: "nginx:latest" },
          { name: "ports", type: "object[]", required: false, description: "Port mappings (host→container)." },
          { name: "env", type: "string[]", required: false, description: "Environment variables (KEY=VALUE)." },
          { name: "volumes", type: "string[]", required: false, description: "Volume mounts (host:container)." },
          { name: "restart_policy", type: "string", required: false, description: "Restart policy (no, always, unless-stopped, on-failure)." },
          { name: "start", type: "boolean", required: false, description: "Start the container after creation (default: true)." },
        ],
      },
      responseExample: json({ id: "ctr_abc123", name: "my-nginx", status: "running" }),
      curlExample: `curl -X POST http://localhost:8090/api/dk/containers/create \\
  -H "Authorization: Bearer YOUR_TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{
    "server_id": "SERVER_ID",
    "name": "my-nginx",
    "image": "nginx:latest",
    "ports": [{"host": 8080, "container": 80}],
    "start": true
  }'`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/containers/create", {
  method: "POST",
  headers: {
    Authorization: "Bearer YOUR_TOKEN",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    server_id: "SERVER_ID",
    name: "my-nginx",
    image: "nginx:latest",
    ports: [{ host: 8080, container: 80 }],
    start: true,
  }),
});`,
    },
    {
      id: "containers-inspect",
      method: "GET",
      path: "/api/dk/containers/{id}/inspect",
      title: "Inspect Container",
      description: "Retrieve detailed information about a container including config, state, and network settings.",
      auth: "required",
      capability: "containers.view",
      parameters: [
        { name: "id", type: "string", required: true, description: "Container record ID." },
      ],
      curlExample: `curl http://localhost:8090/api/dk/containers/CONTAINER_ID/inspect \\
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/containers/CONTAINER_ID/inspect", {
  headers: { Authorization: "Bearer YOUR_TOKEN" },
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
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/containers/CONTAINER_ID/start", {
  method: "POST",
  headers: { Authorization: "Bearer YOUR_TOKEN" },
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
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/containers/CONTAINER_ID/stop", {
  method: "POST",
  headers: { Authorization: "Bearer YOUR_TOKEN" },
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
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/containers/CONTAINER_ID/restart", {
  method: "POST",
  headers: { Authorization: "Bearer YOUR_TOKEN" },
});`,
    },
    {
      id: "containers-remove",
      method: "POST",
      path: "/api/dk/containers/{id}/remove",
      title: "Remove Container",
      description: "Remove a container. The container must be stopped first unless force is applied.",
      auth: "required",
      capability: "containers.manage",
      parameters: [
        { name: "id", type: "string", required: true, description: "Container record ID." },
      ],
      curlExample: `curl -X POST http://localhost:8090/api/dk/containers/CONTAINER_ID/remove \\
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/containers/CONTAINER_ID/remove", {
  method: "POST",
  headers: { Authorization: "Bearer YOUR_TOKEN" },
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
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/containers/CONTAINER_ID/pause", {
  method: "POST",
  headers: { Authorization: "Bearer YOUR_TOKEN" },
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
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/containers/CONTAINER_ID/unpause", {
  method: "POST",
  headers: { Authorization: "Bearer YOUR_TOKEN" },
});`,
    },
    {
      id: "containers-desired-state",
      method: "PUT",
      path: "/api/dk/containers/{id}/desired-state",
      title: "Set Desired State",
      description: "Set the desired state for a container (running or stopped). The agent will reconcile automatically.",
      auth: "required",
      capability: "containers.manage",
      parameters: [
        { name: "id", type: "string", required: true, description: "Container record ID." },
      ],
      requestBody: {
        contentType: "application/json",
        fields: [
          { name: "desired_state", type: "string", required: true, description: "Target state: \"running\" or \"stopped\"." },
        ],
      },
      curlExample: `curl -X PUT http://localhost:8090/api/dk/containers/CONTAINER_ID/desired-state \\
  -H "Authorization: Bearer YOUR_TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{"desired_state": "running"}'`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/containers/CONTAINER_ID/desired-state", {
  method: "PUT",
  headers: {
    Authorization: "Bearer YOUR_TOKEN",
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
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/containers/CONTAINER_ID/resources", {
  headers: { Authorization: "Bearer YOUR_TOKEN" },
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
          { name: "cpu_limit", type: "number", required: false, description: "CPU limit in cores (e.g. 0.5, 1, 2)." },
          { name: "memory_limit", type: "number", required: false, description: "Memory limit in bytes." },
        ],
      },
      curlExample: `curl -X PUT http://localhost:8090/api/dk/containers/CONTAINER_ID/resources \\
  -H "Authorization: Bearer YOUR_TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{"cpu_limit": 1, "memory_limit": 536870912}'`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/containers/CONTAINER_ID/resources", {
  method: "PUT",
  headers: {
    Authorization: "Bearer YOUR_TOKEN",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ cpu_limit: 1, memory_limit: 536870912 }),
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
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/containers/CONTAINER_ID/env", {
  headers: { Authorization: "Bearer YOUR_TOKEN" },
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
          { name: "env", type: "string[]", required: true, description: "Full list of environment variables (KEY=VALUE).", example: "[\"NODE_ENV=production\", \"PORT=3000\"]" },
        ],
      },
      curlExample: `curl -X PUT http://localhost:8090/api/dk/containers/CONTAINER_ID/env \\
  -H "Authorization: Bearer YOUR_TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{"env": ["NODE_ENV=production", "PORT=3000"]}'`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/containers/CONTAINER_ID/env", {
  method: "PUT",
  headers: {
    Authorization: "Bearer YOUR_TOKEN",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ env: ["NODE_ENV=production", "PORT=3000"] }),
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
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/images", {
  headers: { Authorization: "Bearer YOUR_TOKEN" },
});
const images = await res.json();`,
    },
    {
      id: "images-remove",
      method: "DELETE",
      path: "/api/dk/images/{id}",
      title: "Remove Image",
      description: "Remove a Docker image from its server. Fails if the image is in use by a container.",
      auth: "required",
      capability: "images.manage",
      parameters: [
        { name: "id", type: "string", required: true, description: "Image record ID." },
      ],
      curlExample: `curl -X DELETE http://localhost:8090/api/dk/images/IMAGE_ID \\
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/images/IMAGE_ID", {
  method: "DELETE",
  headers: { Authorization: "Bearer YOUR_TOKEN" },
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
        ],
      },
      curlExample: `curl -X POST http://localhost:8090/api/dk/images/prune \\
  -H "Authorization: Bearer YOUR_TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{"server_id": "SERVER_ID"}'`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/images/prune", {
  method: "POST",
  headers: {
    Authorization: "Bearer YOUR_TOKEN",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ server_id: "SERVER_ID" }),
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
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/compose", {
  headers: { Authorization: "Bearer YOUR_TOKEN" },
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
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/compose/STACK_ID", {
  headers: { Authorization: "Bearer YOUR_TOKEN" },
});
const stack = await res.json();`,
    },
    {
      id: "compose-create",
      method: "POST",
      path: "/api/dk/compose",
      title: "Create Stack",
      description: "Create a new compose stack with YAML content. Does not deploy — use the Up endpoint to deploy.",
      auth: "required",
      capability: "compose.manage",
      requestBody: {
        contentType: "application/json",
        fields: [
          { name: "server_id", type: "string", required: true, description: "Target server ID." },
          { name: "name", type: "string", required: true, description: "Stack name." },
          { name: "yaml", type: "string", required: true, description: "Docker Compose YAML content." },
        ],
      },
      curlExample: `curl -X POST http://localhost:8090/api/dk/compose \\
  -H "Authorization: Bearer YOUR_TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{"server_id": "SERVER_ID", "name": "my-stack", "yaml": "version: \\"3\\"\\nservices:\\n  web:\\n    image: nginx"}'`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/compose", {
  method: "POST",
  headers: {
    Authorization: "Bearer YOUR_TOKEN",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    server_id: "SERVER_ID",
    name: "my-stack",
    yaml: "version: \\"3\\"\\nservices:\\n  web:\\n    image: nginx",
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
          { name: "name", type: "string", required: false, description: "New stack name." },
          { name: "yaml", type: "string", required: false, description: "Updated YAML content." },
        ],
      },
      curlExample: `curl -X PUT http://localhost:8090/api/dk/compose/STACK_ID \\
  -H "Authorization: Bearer YOUR_TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{"yaml": "version: \\"3\\"\\nservices:\\n  web:\\n    image: nginx:alpine"}'`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/compose/STACK_ID", {
  method: "PUT",
  headers: {
    Authorization: "Bearer YOUR_TOKEN",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    yaml: "version: \\"3\\"\\nservices:\\n  web:\\n    image: nginx:alpine",
  }),
});`,
    },
    {
      id: "compose-delete",
      method: "DELETE",
      path: "/api/dk/compose/{id}",
      title: "Delete Stack",
      description: "Delete a compose stack record. Does not stop running containers — use Down first.",
      auth: "required",
      capability: "compose.manage",
      parameters: [
        { name: "id", type: "string", required: true, description: "Compose stack record ID." },
      ],
      curlExample: `curl -X DELETE http://localhost:8090/api/dk/compose/STACK_ID \\
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/compose/STACK_ID", {
  method: "DELETE",
  headers: { Authorization: "Bearer YOUR_TOKEN" },
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
          { name: "driver", type: "string", required: false, description: "Network driver (default: bridge)." },
        ],
      },
      curlExample: `curl -X POST http://localhost:8090/api/dk/networks \\
  -H "Authorization: Bearer YOUR_TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{"server_id": "SERVER_ID", "name": "my-network", "driver": "bridge"}'`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/networks", {
  method: "POST",
  headers: {
    Authorization: "Bearer YOUR_TOKEN",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    server_id: "SERVER_ID",
    name: "my-network",
    driver: "bridge",
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
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/networks/NETWORK_ID", {
  method: "DELETE",
  headers: { Authorization: "Bearer YOUR_TOKEN" },
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
          { name: "container_id", type: "string", required: true, description: "Container record ID to connect." },
        ],
      },
      curlExample: `curl -X POST http://localhost:8090/api/dk/networks/NETWORK_ID/connect \\
  -H "Authorization: Bearer YOUR_TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{"container_id": "CONTAINER_ID"}'`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/networks/NETWORK_ID/connect", {
  method: "POST",
  headers: {
    Authorization: "Bearer YOUR_TOKEN",
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
          { name: "container_id", type: "string", required: true, description: "Container record ID to disconnect." },
        ],
      },
      curlExample: `curl -X POST http://localhost:8090/api/dk/networks/NETWORK_ID/disconnect \\
  -H "Authorization: Bearer YOUR_TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{"container_id": "CONTAINER_ID"}'`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/networks/NETWORK_ID/disconnect", {
  method: "POST",
  headers: {
    Authorization: "Bearer YOUR_TOKEN",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ container_id: "CONTAINER_ID" }),
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
          { name: "driver", type: "string", required: false, description: "Volume driver (default: local)." },
        ],
      },
      curlExample: `curl -X POST http://localhost:8090/api/dk/volumes \\
  -H "Authorization: Bearer YOUR_TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{"server_id": "SERVER_ID", "name": "my-volume"}'`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/volumes", {
  method: "POST",
  headers: {
    Authorization: "Bearer YOUR_TOKEN",
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
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/volumes/VOLUME_ID", {
  method: "DELETE",
  headers: { Authorization: "Bearer YOUR_TOKEN" },
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
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/volumes/VOLUME_ID/inspect", {
  headers: { Authorization: "Bearer YOUR_TOKEN" },
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
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/backups/policies", {
  headers: { Authorization: "Bearer YOUR_TOKEN" },
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
          { name: "name", type: "string", required: true, description: "Policy name." },
          { name: "container_id", type: "string", required: true, description: "Container to back up." },
          { name: "cron", type: "string", required: true, description: "Cron schedule expression.", example: "0 2 * * *" },
          { name: "retention_days", type: "number", required: false, description: "Days to keep backups (default: 30)." },
          { name: "s3_bucket", type: "string", required: false, description: "S3 bucket for remote storage." },
        ],
      },
      curlExample: `curl -X POST http://localhost:8090/api/dk/backups/policies \\
  -H "Authorization: Bearer YOUR_TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{"name": "daily-db", "container_id": "CTR_ID", "cron": "0 2 * * *", "retention_days": 30}'`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/backups/policies", {
  method: "POST",
  headers: {
    Authorization: "Bearer YOUR_TOKEN",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    name: "daily-db",
    container_id: "CTR_ID",
    cron: "0 2 * * *",
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
  -H "Authorization: Bearer YOUR_TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{"cron": "0 3 * * *", "retention_days": 60}'`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/backups/policies/POLICY_ID", {
  method: "PUT",
  headers: {
    Authorization: "Bearer YOUR_TOKEN",
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
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/backups/policies/POLICY_ID", {
  method: "DELETE",
  headers: { Authorization: "Bearer YOUR_TOKEN" },
});`,
    },
    {
      id: "backups-policies-trigger",
      method: "POST",
      path: "/api/dk/backups/policies/{id}/trigger",
      title: "Trigger Backup",
      description: "Manually trigger an immediate backup for a policy, bypassing the cron schedule.",
      auth: "required",
      capability: "backups.manage",
      parameters: [
        { name: "id", type: "string", required: true, description: "Policy record ID." },
      ],
      curlExample: `curl -X POST http://localhost:8090/api/dk/backups/policies/POLICY_ID/trigger \\
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/backups/policies/POLICY_ID/trigger", {
  method: "POST",
  headers: { Authorization: "Bearer YOUR_TOKEN" },
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
        { id: "bkp_01", policy_id: "pol_01", status: "completed", size: 52428800, created: "2026-03-29T02:00:00Z" },
      ]),
      curlExample: `curl http://localhost:8090/api/dk/backups/records \\
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/backups/records", {
  headers: { Authorization: "Bearer YOUR_TOKEN" },
});
const records = await res.json();`,
    },
    {
      id: "backups-records-restore",
      method: "POST",
      path: "/api/dk/backups/records/{id}/restore",
      title: "Restore Backup",
      description: "Restore a container from a specific backup record. The container will be stopped during restore.",
      auth: "required",
      capability: "backups.manage",
      parameters: [
        { name: "id", type: "string", required: true, description: "Backup record ID." },
      ],
      curlExample: `curl -X POST http://localhost:8090/api/dk/backups/records/BACKUP_ID/restore \\
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/backups/records/BACKUP_ID/restore", {
  method: "POST",
  headers: { Authorization: "Bearer YOUR_TOKEN" },
});`,
    },
    {
      id: "backups-cleanup",
      method: "POST",
      path: "/api/dk/backups/cleanup",
      title: "Cleanup Expired Backups",
      description: "Manually trigger cleanup of backup records that exceed their policy's retention period.",
      auth: "required",
      capability: "backups.manage",
      curlExample: `curl -X POST http://localhost:8090/api/dk/backups/cleanup \\
  -H "Authorization: Bearer YOUR_TOKEN"`,
      jsExample: `const res = await fetch("http://localhost:8090/api/dk/backups/cleanup", {
  method: "POST",
  headers: { Authorization: "Bearer YOUR_TOKEN" },
});
const { deleted } = await res.json();`,
    },
  ],
};

// ── Export ──────────────────────────────────────────────────────────────────

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
];
