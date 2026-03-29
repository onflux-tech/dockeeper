import { Key } from "lucide-react";

import type { ApiGroup } from "@/types/docs";

export const apiGroups: ApiGroup[] = [
  {
    id: "authentication",
    title: "Authentication",
    description:
      "Endpoints for admin authentication, token management, and initial setup.",
    icon: Key,
    endpoints: [
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
        responseExample: JSON.stringify(
          {
            token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
            refresh: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
            admin: {
              id: "abc123",
              email: "admin@example.com",
              created: "2026-01-01 00:00:00.000Z",
              updated: "2026-01-01 00:00:00.000Z",
            },
          },
          null,
          2,
        ),
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
const data = await res.json();`,
      },
      {
        id: "auth-me",
        method: "GET",
        path: "/api/dk/auth/me",
        title: "Get Current User",
        description:
          "Returns the authenticated admin's profile data. Requires a valid access token.",
        auth: "superuser",
        responseExample: JSON.stringify(
          {
            id: "abc123",
            email: "admin@example.com",
            created: "2026-01-01 00:00:00.000Z",
            updated: "2026-01-01 00:00:00.000Z",
          },
          null,
          2,
        ),
        curlExample: `curl http://localhost:8090/api/dk/auth/me \\
  -H "Authorization: Bearer YOUR_TOKEN"`,
        jsExample: `const res = await fetch("http://localhost:8090/api/dk/auth/me", {
  headers: { Authorization: "Bearer YOUR_TOKEN" },
});
const data = await res.json();`,
      },
    ],
  },
];
