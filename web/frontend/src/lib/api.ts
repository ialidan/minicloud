export const BASE_URL = "/api/v1";

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
): Promise<T> {
  const init: RequestInit = {
    method,
    credentials: "same-origin",
    headers: {},
  };

  if (body instanceof FormData) {
    init.body = body;
  } else if (body !== undefined) {
    (init.headers as Record<string, string>)["Content-Type"] =
      "application/json";
    init.body = JSON.stringify(body);
  }

  const res = await fetch(BASE_URL + path, init);
  const ct = res.headers.get("content-type") ?? "";

  if (ct.includes("application/json")) {
    const json = (await res.json()) as { data?: T; error?: string };
    if (!res.ok) {
      throw new ApiError(res.status, json.error ?? "Request failed");
    }
    return json.data as T;
  }

  if (!res.ok) {
    throw new ApiError(res.status, `Request failed (${res.status})`);
  }

  return res as unknown as T;
}

export const api = {
  get: <T>(path: string) => request<T>("GET", path),
  post: <T>(path: string, body?: unknown) => request<T>("POST", path, body),
  put: <T>(path: string, body?: unknown) => request<T>("PUT", path, body),
  patch: <T>(path: string, body?: unknown) => request<T>("PATCH", path, body),
  del: <T>(path: string) => request<T>("DELETE", path),
};

/** URL for directly downloading/viewing a file by ID. */
export function fileUrl(id: string): string {
  return `${BASE_URL}/files/${id}`;
}
