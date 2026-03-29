import type { LucideIcon } from "lucide-react";

// ── HTTP & Auth ─────────────────────────────────────────────────────────────

export type HttpMethod = "GET" | "POST" | "PUT" | "PATCH" | "DELETE";

export type AuthType = "required" | "api-key" | "public" | "superuser";

// ── API Parameters & Request Body ───────────────────────────────────────────

export interface ApiParameter {
  name: string;
  type: string;
  required: boolean;
  description: string;
  example?: string;
}

export interface ApiRequestBody {
  contentType: string;
  fields: ApiParameter[];
}

// ── API Endpoint & Group ────────────────────────────────────────────────────

export interface ApiEndpoint {
  id: string;
  method: HttpMethod;
  path: string;
  title: string;
  description: string;
  auth: AuthType;
  capability?: string;
  parameters?: ApiParameter[];
  queryParams?: ApiParameter[];
  requestBody?: ApiRequestBody;
  responseExample?: string;
  curlExample: string;
  jsExample: string;
}

export interface ApiGroup {
  id: string;
  title: string;
  description: string;
  icon: LucideIcon;
  endpoints: ApiEndpoint[];
}

// ── Changelog ───────────────────────────────────────────────────────────────

export type ChangeType =
  | "feat"
  | "fix"
  | "perf"
  | "breaking"
  | "refactor"
  | "docs";

export type ChangeArea =
  | "manager"
  | "agent"
  | "web"
  | "api"
  | "compose"
  | "infra";

export interface ChangelogChange {
  type: ChangeType;
  area: ChangeArea;
  description: string;
}

export interface ChangelogEntry {
  version: string;
  date: string;
  title?: string;
  summary?: string;
  changes: ChangelogChange[];
}
