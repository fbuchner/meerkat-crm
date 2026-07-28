import { apiFetch, API_BASE_URL, getAuthHeaders } from './client';
import { handleResponse } from './errorHandling';

export interface ApiToken {
  id: number;
  name: string;
  created_at: string;
  last_used_at: string | null;
  revoked_at: string | null;
  /** Null only for tokens created before expiry was introduced. */
  expires_at: string | null;
}

/** Selectable lifetimes; the backend caps this at 365 days. */
export const API_TOKEN_EXPIRY_OPTIONS = [30, 60, 90, 180, 365] as const;

export const DEFAULT_API_TOKEN_EXPIRY_DAYS = 90;

export interface ApiTokenCreateResponse extends ApiToken {
  token: string;
}

export interface ApiTokensListResponse {
  tokens: ApiToken[];
}

export async function getApiTokens(): Promise<ApiTokensListResponse> {
  const response = await apiFetch(`${API_BASE_URL}/api-tokens`, {
    method: 'GET',
    headers: getAuthHeaders(),
  });
  const data = await handleResponse(response, 'Unable to load API tokens.');
  return { tokens: data?.tokens || [] };
}

export async function createApiToken(
  name: string,
  expiresInDays: number = DEFAULT_API_TOKEN_EXPIRY_DAYS,
): Promise<ApiTokenCreateResponse> {
  const response = await apiFetch(`${API_BASE_URL}/api-tokens`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify({ name, expires_in_days: expiresInDays }),
  });
  const data = await handleResponse(response, 'Unable to create API token.');
  return data as ApiTokenCreateResponse;
}

export async function revokeApiToken(id: number): Promise<void> {
  const response = await apiFetch(`${API_BASE_URL}/api-tokens/${id}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  });
  await handleResponse(response, 'Unable to revoke API token.');
}
