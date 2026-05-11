import { apiFetch, API_BASE_URL, getAuthHeaders, parseErrorResponse } from './client';

export interface CalDAVSyncRequest {
  url: string;
  username: string;
  password: string;
  limit?: number;
}

export interface CalDAVSyncResponse {
  message: string;
  created: number;
  skipped: number;
}

export async function syncCalDAVActivities(
  data: CalDAVSyncRequest
): Promise<CalDAVSyncResponse> {
  const response = await apiFetch(
    `${API_BASE_URL}/caldav/sync`,
    {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify(data),
    },
    60000
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  return response.json();
}
