export type SetupStatus = {
  initialized: boolean;
  setupAuthorized: boolean;
  setupLocked: boolean;
  setupExpiresAt?: string | null;
  missingRuntime: string[];
  panelBaseUrl: string;
  panelDomain: string;
  lineDomain: string;
  lineServerAddress: string;
  xrayTarget: string;
  serviceState: string;
  totalRxHuman: string;
  totalTxHuman: string;
  refreshIntervalMs: number;
  clientCount: number;
  activeClientCount: number;
};

export type SessionResponse = {
  authenticated: boolean;
  admin?: {
    username: string;
  };
};

export type DashboardResponse = {
  status: { state: string; message: string };
  summary: {
    panelBaseUrl: string;
    panelDomain: string;
    lineDomain: string;
    lineServerAddress: string;
    xrayTarget: string;
    serviceState: string;
    refreshIntervalMs: number;
    clientCount: number;
    activeClientCount: number;
  };
  traffic: {
    totalRxHuman: string;
    totalTxHuman: string;
  };
};

export type Client = {
  id: number;
  name: string;
  slug: string;
  enabled: boolean;
  last_seen_at: string | null;
  rxHuman: string;
  txHuman: string;
  rxBpsHuman: string;
  txBpsHuman: string;
  shareLink: string;
  mihomoSubscriptionUrl: string;
};

type RequestInitX = RequestInit & {
  skipJsonHeader?: boolean;
};

async function request<T>(path: string, init: RequestInitX = {}) {
  const headers = new Headers(init.headers ?? {});

  if (!init.skipJsonHeader && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const resp = await fetch(path, {
    ...init,
    headers,
    credentials: "include"
  });

  const contentType = resp.headers.get("content-type") ?? "";
  const asJson = contentType.includes("application/json");
  const payload = asJson ? await resp.json().catch(() => ({})) : await resp.text();

  if (!resp.ok) {
    const message =
      typeof payload === "object" && payload && "error" in payload
        ? String(payload.error)
        : `Request failed (${resp.status})`;
    throw new Error(message);
  }

  return payload as T;
}

export const api = {
  setupStatus: () => request<SetupStatus>("/api/setup/status"),
  setupInit: (input: { username: string; password: string }) =>
    request("/api/setup/init", {
      method: "POST",
      body: JSON.stringify(input)
    }),
  session: () => request<SessionResponse>("/api/session"),
  login: (input: { username: string; password: string }) =>
    request("/api/auth/login", {
      method: "POST",
      body: JSON.stringify(input)
    }),
  logout: () =>
    request("/api/auth/logout", {
      method: "POST",
      body: JSON.stringify({})
    }),
  dashboard: () => request<DashboardResponse>("/api/dashboard"),
  clients: () => request<{ clients: Client[] }>("/api/clients"),
  createClient: (input: { name: string }) =>
    request("/api/clients", {
      method: "POST",
      body: JSON.stringify(input)
    }),
  patchClient: (
    id: number,
    input: { name?: string; enabled?: boolean; rotateToken?: boolean }
  ) =>
    request(`/api/clients/${id}`, {
      method: "PATCH",
      body: JSON.stringify(input)
    }),
  deleteClient: (id: number) =>
    request(`/api/clients/${id}`, {
      method: "DELETE"
    }),
  syncXray: () =>
    request<{ restart?: { message?: string } }>("/api/services/sync", {
      method: "POST",
      body: JSON.stringify({})
    }),
  updatePassword: (input: { currentPassword: string; newPassword: string }) =>
    request("/api/auth/password", {
      method: "POST",
      body: JSON.stringify(input)
    })
};
