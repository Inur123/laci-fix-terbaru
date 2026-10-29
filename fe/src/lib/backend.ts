import "server-only";

import { cookies, headers } from "next/headers";

const backendUrl = (process.env.BACKEND_URL || "http://127.0.0.1:8080").replace(/\/$/, "");

export class BackendError extends Error {
  constructor(
    message: string,
    public readonly status: number,
    public readonly code?: string,
  ) {
    super(message);
    this.name = "BackendError";
  }
}

type RequestOptions = Omit<RequestInit, "body"> & {
  body?: BodyInit | Record<string, unknown> | unknown[];
};

function withoutEmptyFiles(formData: FormData): FormData {
  const normalized = new FormData();
  for (const [name, value] of formData.entries()) {
    if (typeof value !== "string" && value.size === 0) continue;
    normalized.append(name, value);
  }
  return normalized;
}

export async function backendRequest<T = any>(path: string, options: RequestOptions = {}): Promise<T> {
  const requestHeaders = new Headers(options.headers);
  const cookieHeader = (await headers()).get("cookie");
  if (cookieHeader) requestHeaders.set("cookie", cookieHeader);
  if (process.env.API_KEY) requestHeaders.set("x-api-key", process.env.API_KEY);

  let body = options.body;
  if (body instanceof FormData) body = withoutEmptyFiles(body);
  if (body && !(body instanceof FormData) && !(body instanceof URLSearchParams) && typeof body !== "string") {
    requestHeaders.set("content-type", "application/json");
    body = JSON.stringify(body);
  }

  const response = await fetch(`${backendUrl}${path.startsWith("/") ? path : `/${path}`}`, {
    ...options,
    body: body as BodyInit | undefined,
    headers: requestHeaders,
    cache: options.cache ?? "no-store",
  });

  const contentType = response.headers.get("content-type") || "";
  const payload = contentType.includes("application/json")
    ? await response.json().catch(() => null)
    : await response.text().catch(() => "");

  if (!response.ok) {
    const message = payload?.message || payload?.error || `Backend merespons ${response.status}`;
    throw new BackendError(message, response.status, payload?.code);
  }
  return payload as T;
}

export async function backendDataOrNull<T = any>(path: string): Promise<T | null> {
  try {
    const result = await backendRequest<{ data?: T }>(path);
    return result.data ?? null;
  } catch (error) {
    if (error instanceof BackendError && error.status === 404) return null;
    throw error;
  }
}

export function queryString(values: Record<string, unknown>): string {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(values)) {
    if (value !== undefined && value !== null && value !== "") params.set(key, String(value));
  }
  const query = params.toString();
  return query ? `?${query}` : "";
}

export async function selectedPeriodeId(): Promise<string | undefined> {
  return (await cookies()).get("view_periode_id")?.value;
}

export function errorMessage(error: unknown, fallback: string): string {
  return error instanceof Error && error.message ? error.message : fallback;
}

export async function activePeriode(): Promise<any | null> {
  const result = await backendRequest<any>("/api/periodes");
  return (result.data ?? []).find((periode: any) => periode.isActive) ?? null;
}
