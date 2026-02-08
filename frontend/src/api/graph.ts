// API client for network graph data
import { apiFetch, API_BASE_URL, getAuthHeaders, parseErrorResponse } from './client';
import { GraphResponse } from '../types/graph';

export async function getGraph(): Promise<GraphResponse> {
  const response = await apiFetch(
    `${API_BASE_URL}/graph`,
    { headers: getAuthHeaders() }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  return response.json();
}
