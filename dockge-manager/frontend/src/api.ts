export type Target = {
  id: string;
  environment_id: string;
  name: string;
  base_url: string;
  verify_tls: boolean;
  enabled: boolean;
  last_seen_at: string | null;
  last_version: string | null;
};

export type Stack = {
  name: string;
  status: number;
  api_managed?: boolean;
  composeYAML?: string;
};

export type Application = {
  id: string;
  name: string;
  description: string;
  created_at: string;
};

export type Deployment = {
  id: string;
  application_id: string;
  target_id: string;
  stack_name: string;
  status: string;
  current_revision: number;
  active_revision: number;
  last_error: string;
  last_deployed_at: string | null;
  created_at: string;
  updated_at: string;
};

export type Operation = {
  id: string;
  target_id: string;
  stack_name: string;
  action: string;
  status: string;
  http_status: number | null;
  created_at: string;
  completed_at: string | null;
};

export type AuditEvent = {
  id: string;
  actor: string;
  event_type: string;
  target_id: string | null;
  resource: string;
  created_at: string;
};

const tokenKey = "dockge-manager-token";

export function getToken(): string {
  return localStorage.getItem(tokenKey) || "";
}

export function setToken(token: string): void {
  if (token) localStorage.setItem(tokenKey, token);
  else localStorage.removeItem(tokenKey);
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers || {});
  headers.set("Content-Type", "application/json");
  const token = getToken();
  if (token) headers.set("Authorization", `Bearer ${token}`);
  const response = await fetch(path, { ...init, headers });
  const body = response.status === 204 ? null : await response.json().catch(() => ({}));
  if (!response.ok) {
    const detail = body?.detail ?? body?.error ?? `HTTP ${response.status}`;
    throw new Error(typeof detail === "string" ? detail : JSON.stringify(detail));
  }
  return body as T;
}

export async function login(email: string, password: string): Promise<void> {
  const data = await request<{ access_token: string }>("/api/v1/auth/login", {
    method: "POST",
    body: JSON.stringify({ email, password })
  });
  setToken(data.access_token);
}

export const api = {
  me: () => request<{ email: string }>("/api/v1/auth/me"),
  targets: () => request<Target[]>("/api/v1/targets"),
  createTarget: (body: { name: string; base_url: string; token: string; verify_tls: boolean }) =>
    request<Target>("/api/v1/targets", { method: "POST", body: JSON.stringify(body) }),
  deleteTarget: (id: string) => request<void>(`/api/v1/targets/${id}`, { method: "DELETE" }),
  testTarget: (id: string) => request<Record<string, unknown>>(`/api/v1/targets/${id}/test`, { method: "POST" }),
  info: (id: string) => request<Record<string, unknown>>(`/api/v1/targets/${id}/info`),
  stacks: (id: string) => request<{ stacks: Stack[] }>(`/api/v1/targets/${id}/stacks`),
  action: (id: string, stack: string, action: string) =>
    request<Record<string, unknown>>(`/api/v1/targets/${id}/stacks/${encodeURIComponent(stack)}/actions/${action}`, { method: "POST" }),
  logs: (id: string, stack: string) =>
    request<{ stdout?: string; stderr?: string }>(`/api/v1/targets/${id}/stacks/${encodeURIComponent(stack)}/logs?tail=300`),

  applications: () => request<Application[]>("/api/v1/applications"),
  createApplication: (body: { name: string; description: string }) =>
    request<Application>("/api/v1/applications", { method: "POST", body: JSON.stringify(body) }),

  deployments: () => request<Deployment[]>("/api/v1/deployments"),
  createDeployment: (body: {
    application_id: string;
    target_id: string;
    stack_name: string;
    compose_yaml: string;
    compose_env: string;
    adopt_external: boolean;
  }) => request<Deployment>("/api/v1/deployments", { method: "POST", body: JSON.stringify(body) }),
  deploy: (id: string) => request<Record<string, unknown>>(`/api/v1/deployments/${id}/deploy`, { method: "POST" }),
  rollback: (id: string) => request<Record<string, unknown>>(`/api/v1/deployments/${id}/rollback`, { method: "POST" }),
  snapshots: (id: string) => request<Array<Record<string, unknown>>>(`/api/v1/deployments/${id}/snapshots`),

  operations: () => request<Operation[]>("/api/v1/operations?limit=100"),
  audit: () => request<AuditEvent[]>("/api/v1/audit?limit=100")
};
