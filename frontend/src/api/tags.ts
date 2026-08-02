import { apiFetch, API_BASE_URL, getAuthHeaders, parseErrorResponse } from './client';

export interface Tag {
  id: string;
  created_at: string;
  updated_at: string;
  name: string;
}

export interface ContactTag {
  id: number;
  tag_id: string;
  contact_vcard_uid: string;
}

export interface TagWithContacts {
  tag: Tag;
  contacts: ContactTag[];
}

export interface TagCreateResponse {
  message: string;
  tag: Tag;
}

export interface TagListResponse {
  tags: Tag[];
  total: number;
  page: number;
  limit: number;
  contacts?: ContactTag[];
}

// GET /tags
export async function listTags(params?: {
  page?: number;
  limit?: number;
  include_contacts?: boolean;
}): Promise<TagListResponse> {
  const { page = 1, limit = 100, include_contacts = false } = params || {};
  const queryParams = new URLSearchParams({
    page: page.toString(),
    limit: limit.toString(),
  });
  if (include_contacts) queryParams.append('include_contacts', 'true');
  const response = await apiFetch(
    `${API_BASE_URL}/tags?${queryParams.toString()}`,
    { headers: getAuthHeaders() }
  );
  if (!response.ok) throw await parseErrorResponse(response);
  return response.json();
}

// POST /tags
export async function createTag(name: string): Promise<TagCreateResponse> {
  const response = await apiFetch(`${API_BASE_URL}/tags`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify({ name }),
  });
  if (!response.ok) throw await parseErrorResponse(response);
  return response.json();
}

// PUT /tags/:id
export async function updateTag(id: string, name: string): Promise<Tag> {
  const response = await apiFetch(`${API_BASE_URL}/tags/${id}`, {
    method: 'PUT',
    headers: getAuthHeaders(),
    body: JSON.stringify({ name }),
  });
  if (!response.ok) throw await parseErrorResponse(response);
  return response.json();
}

// DELETE /tags/:id
export async function deleteTag(id: string): Promise<void> {
  const response = await apiFetch(`${API_BASE_URL}/tags/${id}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  });
  if (!response.ok) throw await parseErrorResponse(response);
}

// GET /tags/:id
export async function getTag(id: string): Promise<TagWithContacts> {
  const response = await apiFetch(`${API_BASE_URL}/tags/${id}`, {
    headers: getAuthHeaders(),
  });
  if (!response.ok) throw await parseErrorResponse(response);
  return response.json();
}

// POST /tags/:id/contacts
export async function addContactTag(
  tagId: string,
  contactVCardUid: string
): Promise<ContactTag> {
  const response = await apiFetch(`${API_BASE_URL}/tags/${tagId}/contacts`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify({ contact_vcard_uid: contactVCardUid }),
  });
  if (!response.ok) throw await parseErrorResponse(response);
  return response.json();
}

// DELETE /tags/:id/contacts/:vcard_uid
export async function removeContactTag(
  tagId: string,
  contactVCardUid: string
): Promise<void> {
  const response = await apiFetch(
    `${API_BASE_URL}/tags/${tagId}/contacts/${contactVCardUid}`,
    { method: 'DELETE', headers: getAuthHeaders() }
  );
  if (!response.ok) throw await parseErrorResponse(response);
}
