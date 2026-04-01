import { apiFetch, API_BASE_URL, getAuthHeaders } from './client';
import { handleResponse } from './errorHandling';

export interface ApiToken {
  id: number;
  name: string;
  created_at: string;
  last_used_at: string | null;
  revoked_at: string | null;
}

export interface ApiTokenCreateResponse extends ApiToken {
  token: string;
}

export interface ApiTokensListResponse {
  tokens: ApiToken[];
}

export async function getApiTokens(): Promise<ApiTokensListResponse> {
  const response = await apiFetch(`${API_BASE_URL}/admin/api-tokens`, {
    method: 'GET',
    headers: getAuthHeaders(),
  });
  const data = await handleResponse(response, 'Unable to load API tokens.');
  return { tokens: data?.tokens || [] };
}

export async function createApiToken(name: string): Promise<ApiTokenCreateResponse> {
  const response = await apiFetch(`${API_BASE_URL}/admin/api-tokens`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify({ name }),
  });
  const data = await handleResponse(response, 'Unable to create API token.');
  return data as ApiTokenCreateResponse;
}

export async function revokeApiToken(id: number): Promise<void> {
  const response = await apiFetch(`${API_BASE_URL}/admin/api-tokens/${id}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  });
  await handleResponse(response, 'Unable to revoke API token.');
}
