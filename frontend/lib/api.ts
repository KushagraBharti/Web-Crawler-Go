export const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8080';

export class APIError extends Error {
  code?: string;
  status: number;
  details?: unknown;

  constructor(message: string, status: number, code?: string, details?: unknown) {
    super(message);
    this.name = 'APIError';
    this.status = status;
    this.code = code;
    this.details = details;
  }
}

export async function fetchJSON<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { 'Content-Type': 'application/json', ...(options?.headers || {}) },
    ...options
  });
  if (!res.ok) {
    let message = `Request failed: ${res.status}`;
    let code: string | undefined;
    let details: unknown;
    const raw = await res.text();
    if (raw) {
      try {
        const body = JSON.parse(raw) as {
          error?: {
            code?: string;
            message?: string;
            details?: unknown;
          };
        };
        if (body?.error?.message) {
          message = body.error.message;
        }
        if (body?.error?.code) {
          code = body.error.code;
        }
        details = body?.error?.details;
      } catch {
        message = raw;
      }
    }
    throw new APIError(message, res.status, code, details);
  }
  return res.json() as Promise<T>;
}
