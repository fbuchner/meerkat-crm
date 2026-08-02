import { apiFetch, API_BASE_URL, getAuthHeaders, parseErrorResponse } from './client';

export interface Circle {
  id: string;
  created_at: string;
  updated_at: string;
  name: string;
}

export interface CircleMember {
  id: number;
  circle_id: string;
  member_vcard_uid: string;
}

export interface CircleWithMembers {
  circle: Circle;
  members: CircleMember[];
}

export interface CircleCreateResponse {
  message: string;
  circle: Circle;
}

export interface CircleListResponse {
  circles: Circle[];
  total: number;
  page: number;
  limit: number;
  members?: CircleMember[];
}

// GET /circles
export async function listCircles(params?: {
  page?: number;
  limit?: number;
  include_members?: boolean;
}): Promise<CircleListResponse> {
  const { page = 1, limit = 100, include_members = false } = params || {};
  const queryParams = new URLSearchParams({
    page: page.toString(),
    limit: limit.toString(),
  });
  if (include_members) queryParams.append('include_members', 'true');
  const response = await apiFetch(
    `${API_BASE_URL}/circles?${queryParams.toString()}`,
    { headers: getAuthHeaders() }
  );
  if (!response.ok) throw await parseErrorResponse(response);
  return response.json();
}

// POST /circles
export async function createCircle(name: string): Promise<CircleCreateResponse> {
  const response = await apiFetch(`${API_BASE_URL}/circles`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify({ name }),
  });
  if (!response.ok) throw await parseErrorResponse(response);
  return response.json();
}

// PUT /circles/:id
export async function updateCircle(id: string, name: string): Promise<Circle> {
  const response = await apiFetch(`${API_BASE_URL}/circles/${id}`, {
    method: 'PUT',
    headers: getAuthHeaders(),
    body: JSON.stringify({ name }),
  });
  if (!response.ok) throw await parseErrorResponse(response);
  return response.json();
}

// DELETE /circles/:id
export async function deleteCircle(id: string): Promise<void> {
  const response = await apiFetch(`${API_BASE_URL}/circles/${id}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  });
  if (!response.ok) throw await parseErrorResponse(response);
}

// GET /circles/:id
export async function getCircle(id: string): Promise<CircleWithMembers> {
  const response = await apiFetch(`${API_BASE_URL}/circles/${id}`, {
    headers: getAuthHeaders(),
  });
  if (!response.ok) throw await parseErrorResponse(response);
  return response.json();
}

// POST /circles/:id/members
export async function addCircleMember(
  circleId: string,
  memberVCardUid: string
): Promise<CircleMember> {
  const response = await apiFetch(`${API_BASE_URL}/circles/${circleId}/members`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify({ member_vcard_uid: memberVCardUid }),
  });
  if (!response.ok) throw await parseErrorResponse(response);
  return response.json();
}

// DELETE /circles/:id/members/:vcard_uid
export async function removeCircleMember(
  circleId: string,
  memberVCardUid: string
): Promise<void> {
  const response = await apiFetch(
    `${API_BASE_URL}/circles/${circleId}/members/${memberVCardUid}`,
    { method: 'DELETE', headers: getAuthHeaders() }
  );
  if (!response.ok) throw await parseErrorResponse(response);
}
