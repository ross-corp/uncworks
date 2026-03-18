/** Centralized fetch wrapper that injects the API key Authorization header. */

const API_BASE = import.meta.env.VITE_API_URL ?? "";
const API_KEY = import.meta.env.VITE_API_KEY ?? "";

/**
 * Wrapper around fetch() that automatically injects the Authorization header
 * and prepends the API base URL. Use this for all REST API calls.
 */
export function apiFetch(path: string, init?: RequestInit): Promise<Response> {
  const headers = new Headers(init?.headers);
  if (API_KEY) {
    headers.set("Authorization", `Bearer ${API_KEY}`);
  }

  return fetch(`${API_BASE}${path}`, {
    ...init,
    headers,
  });
}

/** Returns the full WebSocket URL for a given path, with API key as query param. */
export function apiWsUrl(path: string): string {
  const base = API_BASE || window.location.origin;
  const wsBase = base.replace(/^http/, "ws");
  const url = new URL(`${wsBase}${path}`);
  if (API_KEY) {
    url.searchParams.set("token", API_KEY);
  }
  return url.toString();
}

/** Returns the full URL for EventSource (SSE) connections. */
export function apiSseUrl(path: string): string {
  return `${API_BASE}${path}`;
}
