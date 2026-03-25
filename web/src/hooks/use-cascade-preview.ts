import { useQuery } from "@tanstack/react-query";
import { getAccessToken } from "@/lib/api";

// ── Types ────────────────────────────────────────

export type CascadeEntityType = "container" | "volume" | "server";

export interface ResourceCount {
  deletable: number;
  protected: number;
}

export interface ResourceCountWithItems extends ResourceCount {
  items: ResourceItem[];
}

export interface ResourceItem {
  id: string;
  name: string;
  eligible: boolean;
  reason: string;
}

export interface CascadeEntity {
  id: string;
  name: string;
  type: string;
}

export interface ContainerCascadePreview {
  entity: CascadeEntity;
  impact: {
    alert_rules: ResourceCount;
    volumes: ResourceCountWithItems;
    notification_logs: ResourceCount;
    backup_policies: ResourceCount;
    terminal_sessions: ResourceCount;
    metrics_records: ResourceCount;
  };
}

export interface VolumeCascadePreview {
  entity: CascadeEntity;
  impact: {
    backup_policies: ResourceCount;
    backup_records: ResourceCount;
  };
  in_use_by: string[];
}

export interface ServerCascadePreview {
  entity: CascadeEntity;
  impact: {
    containers: ResourceCount;
    volumes: ResourceCount;
    images: ResourceCount;
    networks: ResourceCount;
    compose_stacks: ResourceCount;
    alert_rules: ResourceCount;
    backup_policies: ResourceCount;
    backup_records: ResourceCount;
    notification_logs: ResourceCount;
    terminal_sessions: ResourceCount;
    traefik_routes: ResourceCount;
    metrics_records: ResourceCount;
  };
}

export type CascadePreviewData =
  | ContainerCascadePreview
  | VolumeCascadePreview
  | ServerCascadePreview;

// ── Helpers ────────────────────────────────────────

function authHeaders(): Record<string, string> {
  const token = getAccessToken();
  const h: Record<string, string> = {};
  if (token) h["Authorization"] = token;
  return h;
}

async function fetchJson<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, init);
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(body.error || res.statusText);
  }
  return res.json();
}

const PREVIEW_ENDPOINTS: Record<CascadeEntityType, string> = {
  container: "/api/dk/containers",
  volume: "/api/dk/volumes",
  server: "/api/dk/servers",
};

// ── Hook ────────────────────────────────────────

type PreviewMap = {
  container: ContainerCascadePreview;
  volume: VolumeCascadePreview;
  server: ServerCascadePreview;
};

export function useCascadePreview<T extends CascadeEntityType>(
  entityType: T,
  entityId: string,
) {
  return useQuery<PreviewMap[T]>({
    queryKey: ["cascade-preview", entityType, entityId],
    queryFn: () =>
      fetchJson<PreviewMap[T]>(
        `${PREVIEW_ENDPOINTS[entityType]}/${entityId}/cascade-preview`,
        { headers: authHeaders() },
      ),
    enabled: !!entityId,
  });
}
