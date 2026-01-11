// Activities-related API calls
import { apiFetch, API_BASE_URL, getAuthHeaders, parseErrorResponse } from './client';

export interface ActivityContact {
  ID: number;
  firstname: string;
  lastname: string;
  nickname?: string;
}

export interface Activity {
  ID: number;
  title: string;
  description?: string;
  location?: string;
  date: string;
  CreatedAt: string;
  UpdatedAt: string;
  contacts?: ActivityContact[];
}

export interface ActivitiesResponse {
  activities: Activity[];
  total: number;
  page: number;
  limit: number;
}

export interface GetActivitiesParams {
  page?: number;
  limit?: number;
  includeContacts?: boolean;
  search?: string;
}

// Get all activities
export async function getActivities(
  params: GetActivitiesParams,
  token: string
): Promise<ActivitiesResponse> {
  const { page = 1, limit = 25, includeContacts = false } = params;
  const search = params.search?.trim();
  
  const queryParams = new URLSearchParams({
    page: page.toString(),
    limit: limit.toString(),
  });
  
  if (includeContacts) {
    queryParams.append('include', 'contacts');
  }

  if (search) {
    queryParams.append('search', search);
  }

  const response = await apiFetch(
    `${API_BASE_URL}/activities?${queryParams.toString()}`,
    { headers: getAuthHeaders(token) }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  return response.json();
}

// Get activities for a contact
export async function getContactActivities(
  contactId: string | number,
  token: string
): Promise<{ activities: Activity[] }> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/${contactId}/activities`,
    { headers: getAuthHeaders(token) }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  return response.json();
}

// Get single activity
export async function getActivity(
  id: string | number,
  token: string
): Promise<Activity> {
  const response = await apiFetch(
    `${API_BASE_URL}/activities/${id}`,
    { headers: getAuthHeaders(token) }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  return response.json();
}

// Create activity
export async function createActivity(
  data: {
    title: string;
    description: string;
    location: string;
    date: string;
    contact_ids: number[];
  },
  token: string
): Promise<Activity> {
  const response = await apiFetch(
    `${API_BASE_URL}/activities`,
    {
      method: 'POST',
      headers: getAuthHeaders(token),
      body: JSON.stringify(data),
    }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  return response.json();
}

// Update activity
export async function updateActivity(
  id: string | number,
  data: {
    title?: string;
    description?: string;
    location?: string;
    date?: string;
    contact_ids?: number[];
  },
  token: string
): Promise<Activity> {
  const response = await apiFetch(
    `${API_BASE_URL}/activities/${id}`,
    {
      method: 'PUT',
      headers: getAuthHeaders(token),
      body: JSON.stringify(data),
    }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  return response.json();
}

// Delete activity
export async function deleteActivity(
  id: string | number,
  token: string
): Promise<void> {
  const response = await apiFetch(
    `${API_BASE_URL}/activities/${id}`,
    {
      method: 'DELETE',
      headers: getAuthHeaders(token),
    }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }
}
