const API_BASE_URL =
  import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

export class ApiError extends Error {
  status: number;
  code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

async function request<T>(
  path: string,
  apiKey: string,
  options: RequestInit = {},
): Promise<T> {
  const res = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      "X-API-Key": apiKey,
      ...options.headers,
    },
  });

  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new ApiError(
      res.status,
      body.code ?? "UNKNOWN",
      body.error ?? res.statusText,
    );
  }

  const text = await res.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

export interface Client {
  client_id: string;
  name: string;
  email: string;
  created_at: string;
  is_active: boolean;
  default_algorithm: string;
}

export interface Stats {
  client_id: string;
  total_checks: number;
  allowed: number;
  rejected: number;
  by_algorithm: Record<string, number>;
}

export interface Rule {
  name: string;
  algorithm: string;
  limit: number;
  window: number;
}

export interface Exemption {
  identifier: string;
  reason: string;
}

export const api = {
  me: (apiKey: string) => request<Client>("/me", apiKey),

  stats: (apiKey: string, clientId: string) =>
    request<Stats>(`/stats/${clientId}`, apiKey),

  listRules: (apiKey: string) =>
    request<{ rules: Rule[] }>("/rules/list", apiKey).then((r) => r.rules),
  createRule: (apiKey: string, rule: Rule) =>
    request<Rule>("/rules", apiKey, {
      method: "POST",
      body: JSON.stringify(rule),
    }),
  deleteRule: (apiKey: string, name: string) =>
    request<void>(`/rules/${encodeURIComponent(name)}`, apiKey, {
      method: "DELETE",
    }),

  listExemptions: (apiKey: string) =>
    request<{ exemptions: Exemption[] }>("/exemptions/list", apiKey).then(
      (r) => r.exemptions,
    ),
  createExemption: (apiKey: string, exemption: Exemption) =>
    request<Exemption>("/exemptions", apiKey, {
      method: "POST",
      body: JSON.stringify(exemption),
    }),
  deleteExemption: (apiKey: string, identifier: string) =>
    request<void>(`/exemptions/${encodeURIComponent(identifier)}`, apiKey, {
      method: "DELETE",
    }),
};
