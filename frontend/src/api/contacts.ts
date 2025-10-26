// Contact-related API calls
import { apiFetch, API_BASE_URL, getAuthHeaders } from './client';

export interface Contact {
  ID: number;
  firstname: string;
  lastname: string;
  nickname?: string;
  gender?: string;
  email?: string;
  phone?: string;
  birthday?: string;
  address?: string;
  how_we_met?: string;
  food_preference?: string;
  work_information?: string;
  contact_information?: string;
  circles?: string[];
}

export interface ContactsResponse {
  contacts: Contact[];
  total: number;
  page: number;
  limit: number;
}

export interface GetContactsParams {
  page?: number;
  limit?: number;
  search?: string;
  circle?: string;
}

// Get all contacts with pagination and filters
export async function getContacts(
  params: GetContactsParams,
  token: string
): Promise<ContactsResponse> {
  const { page = 1, limit = 25, search = '', circle = '' } = params;
  
  const queryParams = new URLSearchParams({
    page: page.toString(),
    limit: limit.toString(),
  });
  
  if (search) queryParams.append('search', search);
  if (circle) queryParams.append('circle', circle);

  const response = await apiFetch(
    `${API_BASE_URL}/contacts?${queryParams.toString()}`,
    { headers: getAuthHeaders(token) }
  );

  if (!response.ok) {
    throw new Error('Failed to fetch contacts');
  }

  return response.json();
}

// Get single contact
export async function getContact(
  id: string | number,
  token: string
): Promise<Contact> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/${id}`,
    { headers: getAuthHeaders(token) }
  );

  if (!response.ok) {
    throw new Error('Failed to fetch contact');
  }

  return response.json();
}

// Update contact
export async function updateContact(
  id: string | number,
  data: Partial<Contact>,
  token: string
): Promise<Contact> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/${id}`,
    {
      method: 'PUT',
      headers: getAuthHeaders(token),
      body: JSON.stringify(data),
    }
  );

  if (!response.ok) {
    throw new Error('Failed to update contact');
  }

  return response.json();
}

// Delete contact
export async function deleteContact(
  id: string | number,
  token: string
): Promise<void> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/${id}`,
    {
      method: 'DELETE',
      headers: getAuthHeaders(token),
    }
  );

  if (!response.ok) {
    throw new Error('Failed to delete contact');
  }
}

// Get contact profile picture
export async function getContactProfilePicture(
  id: string | number,
  token: string
): Promise<Blob | null> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/${id}/profile_picture`,
    { headers: { 'Authorization': `Bearer ${token}` } }
  );

  if (!response.ok) {
    return null;
  }

  return response.blob();
}

// Get all circles
export async function getCircles(token: string): Promise<string[]> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/circles`,
    { headers: getAuthHeaders(token) }
  );

  if (!response.ok) {
    throw new Error('Failed to fetch circles');
  }

  const data = await response.json();
  return data.circles || [];
}
