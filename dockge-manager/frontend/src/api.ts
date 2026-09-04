export type Target = {
  id: string;
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
  stacks: (id: string) => request<{ stacks: Stack[] }>(`/api/v1/targets/${id}/stacks`),
  action: (id: string, stack: string, action: string) =>
    request<Record<string, unknown>>(`/api/v1/targets/${id}/stacks/${encodeURIComponent(stack)}/actions/${action}`, { method: "POST" }),
  logs: (id: string, stack: string) =>
    request<{ stdout?: string; stderr?: string }>(`/api/v1/targets/${id}/stacks/${encodeURIComponent(stack)}/logs?tail=300`)
};
