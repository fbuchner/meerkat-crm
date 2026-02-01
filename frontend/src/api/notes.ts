// Notes-related API calls
import { apiFetch, API_BASE_URL, getAuthHeaders, parseErrorResponse } from './client';

export interface Note {
  ID: number;
  content: string;
  date: string;
  contact_id?: number;
  CreatedAt: string;
  UpdatedAt: string;
}

export interface NotesResponse {
  notes: Note[];
  total?: number;
  page?: number;
  limit?: number;
}

export interface GetNotesParams {
  page?: number;
  limit?: number;
  search?: string;
  fromDate?: string;
  toDate?: string;
}

// Get notes for a contact
export async function getContactNotes(
  contactId: string | number,
  token: string
): Promise<NotesResponse> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/${contactId}/notes`,
    { headers: getAuthHeaders(token) }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  return response.json();
}

// Get all unassigned notes
export async function getUnassignedNotes(
  token: string,
  params: GetNotesParams = {}
): Promise<NotesResponse> {
  const { page = 1, limit = 25 } = params;
  const search = params.search?.trim();

  const queryParams = new URLSearchParams({
    page: page.toString(),
    limit: limit.toString(),
  });

  if (search) {
    queryParams.append('search', search);
  }

  if (params.fromDate) {
    queryParams.append('fromDate', params.fromDate);
  }

  if (params.toDate) {
    queryParams.append('toDate', params.toDate);
  }

  const response = await apiFetch(
    `${API_BASE_URL}/notes?${queryParams.toString()}`,
    { headers: getAuthHeaders(token) }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  return response.json();
}

// Get single note
export async function getNote(
  id: string | number,
  token: string
): Promise<Note> {
  const response = await apiFetch(
    `${API_BASE_URL}/notes/${id}`,
    { headers: getAuthHeaders(token) }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  return response.json();
}

// Create note for contact
export async function createNote(
  contactId: string | number,
  data: { content: string; date: string; contact_id?: number },
  token: string
): Promise<Note> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/${contactId}/notes`,
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

// Create unassigned note
export async function createUnassignedNote(
  data: { content: string; date: string },
  token: string
): Promise<Note> {
  const response = await apiFetch(
    `${API_BASE_URL}/notes`,
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

// Update note
export async function updateNote(
  id: string | number,
  data: { content: string; date: string; contact_id?: number },
  token: string
): Promise<Note> {
  const response = await apiFetch(
    `${API_BASE_URL}/notes/${id}`,
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

// Delete note
export async function deleteNote(
  id: string | number,
  token: string
): Promise<void> {
  const response = await apiFetch(
    `${API_BASE_URL}/notes/${id}`,
    {
      method: 'DELETE',
      headers: getAuthHeaders(token),
    }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }
}
