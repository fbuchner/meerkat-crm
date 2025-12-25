// Relationship-related API calls
import { apiFetch, API_BASE_URL, getAuthHeaders, parseErrorResponse } from './client';
import { Contact } from './contacts';

export interface Relationship {
  ID: number;
  CreatedAt: string;
  UpdatedAt: string;
  name: string;
  type: string;
  gender?: string;
  birthday?: string;
  contact_id: number;
  related_contact_id?: number;
  related_contact?: Pick<Contact, 'ID' | 'firstname' | 'lastname' | 'gender' | 'birthday'>;
}

export interface RelationshipFormData {
  name: string;
  type: string;
  gender?: string;
  birthday?: string;
  related_contact_id?: number | null;
}

export interface RelationshipsResponse {
  relationships: Relationship[];
}

// Get all relationships for a contact
export async function getRelationships(
  contactId: number | string,
  token: string
): Promise<RelationshipsResponse> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/${contactId}/relationships`,
    { headers: getAuthHeaders(token) }
  );

  if (!response.ok) {
    throw new Error('Failed to fetch relationships');
  }

  return response.json();
}

// Create a new relationship
export async function createRelationship(
  contactId: number | string,
  data: RelationshipFormData,
  token: string
): Promise<Relationship> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/${contactId}/relationships`,
    {
      method: 'POST',
      headers: getAuthHeaders(token),
      body: JSON.stringify(data),
    }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  const result = await response.json();
  return result.relationship || result;
}

// Update an existing relationship
export async function updateRelationship(
  contactId: number | string,
  relationshipId: number,
  data: RelationshipFormData,
  token: string
): Promise<Relationship> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/${contactId}/relationships/${relationshipId}`,
    {
      method: 'PUT',
      headers: getAuthHeaders(token),
      body: JSON.stringify(data),
    }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  const result = await response.json();
  return result.relationship || result;
}

// Delete a relationship
export async function deleteRelationship(
  contactId: number | string,
  relationshipId: number,
  token: string
): Promise<void> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/${contactId}/relationships/${relationshipId}`,
    {
      method: 'DELETE',
      headers: getAuthHeaders(token),
    }
  );

  if (!response.ok) {
    throw new Error('Failed to delete relationship');
  }
}

// Preset relationship types
export const RELATIONSHIP_TYPES = [
  'Child',
  'Parent',
  'Mother',
  'Father',
  'Spouse',
  'Partner',
  'Sibling',
  'Brother',
  'Sister',
  'Friend',
  'Best Friend',
  'Colleague',
  'Manager',
  'Employee',
  'Neighbor',
  'Other',
] as const;
