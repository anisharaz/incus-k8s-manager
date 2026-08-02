// Shared API client. Every backend call should go through here rather than
// raw fetch(), so credentials/error-parsing/401-handling stay consistent.

interface ApiErrorBody {
  error: string;
  message: string;
  code: number;
}

export class ApiError extends Error {
  code: number;
  errorType: string;

  constructor(body: ApiErrorBody) {
    super(body.message);
    this.name = "ApiError";
    this.code = body.code;
    this.errorType = body.error;
  }
}

// Fired whenever any request comes back 401 (not logged in / session
// expired). AuthContext registers a single handler here on mount so it can
// react uniformly whether the 401 came from the initial session-restore
// check or from a mid-session request after the cookie expired.
let unauthorizedHandler: (() => void) | null = null;

export function setUnauthorizedHandler(handler: (() => void) | null): void {
  unauthorizedHandler = handler;
}

async function request<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const response = await fetch(path, {
    ...init,
    credentials: "include",
    headers: {
      ...(init?.body ? { "Content-Type": "application/json" } : {}),
      ...init?.headers,
    },
  });

  if (response.status === 204) {
    return undefined as T;
  }

  if (!response.ok) {
    let body: ApiErrorBody;
    try {
      body = await response.json();
    } catch {
      body = {
        error: "request failed",
        message: `Request failed with status ${response.status}`,
        code: response.status,
      };
    }

    if (response.status === 401) {
      unauthorizedHandler?.();
    }

    throw new ApiError(body);
  }

  return (await response.json()) as T;
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body?: unknown) =>
    request<T>(path, {
      method: "POST",
      body: body !== undefined ? JSON.stringify(body) : undefined,
    }),
  delete: <T = void>(path: string) => request<T>(path, { method: "DELETE" }),
};
