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

// Create contact
export async function createContact(
  data: Partial<Contact>,
  token: string
): Promise<Contact> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts`,
    {
      method: 'POST',
      headers: getAuthHeaders(token),
      body: JSON.stringify(data),
    }
  );

  if (!response.ok) {
    throw new Error('Failed to create contact');
  }

  const result = await response.json();
  return result.contact || result;
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

// Upload contact profile picture
export async function uploadProfilePicture(
  id: string | number,
  imageBlob: Blob,
  token: string
): Promise<void> {
  const formData = new FormData();
  formData.append('photo', imageBlob, 'profile.jpg');

  const response = await apiFetch(
    `${API_BASE_URL}/contacts/${id}/profile_picture`,
    {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${token}` },
      body: formData
    }
  );

  if (!response.ok) {
    throw new Error('Failed to upload profile picture');
  }
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
  // Backend returns array directly, not wrapped in object
  return Array.isArray(data) ? data : [];
}

// Get random contacts (returns 5 contacts)
export async function getRandomContacts(token: string): Promise<Contact[]> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/random`,
    { headers: getAuthHeaders(token) }
  );

  if (!response.ok) {
    throw new Error('Failed to fetch random contacts');
  }

  const data = await response.json();
  return data.contacts || [];
}

// Get upcoming birthdays (returns up to 10 contacts)
export async function getUpcomingBirthdays(token: string): Promise<Contact[]> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/birthdays`,
    { headers: getAuthHeaders(token) }
  );

  if (!response.ok) {
    throw new Error('Failed to fetch upcoming birthdays');
  }

  const data = await response.json();
  return data.contacts || [];
}
