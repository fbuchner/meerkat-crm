// Household API calls -- T1 (docs/fork-plan/tickets/09-T1-households.md).
import { apiFetch, API_BASE_URL, getAuthHeaders, parseErrorResponse } from './client';
import { RelationshipEdge } from './relationshipEdges';

// Mirrors backend/models/household.go's HouseholdType* constants and the
// `oneof=family_unit roommates other` validator on Household.Type. No dynamic
// type-list endpoint exists anywhere in this codebase -- this is a hardcoded
// mirror kept in sync by hand (same convention as every other enum here).
export type HouseholdType = 'family_unit' | 'roommates' | 'other';

export const HOUSEHOLD_TYPES: HouseholdType[] = ['family_unit', 'roommates', 'other'];

// Conventional (not enforced) role tokens, mirroring backend/models/
// household.go's HouseholdRole* constants.
export const HOUSEHOLD_ROLES = ['head', 'child', 'pet', 'roommate'] as const;
export type HouseholdRole = (typeof HOUSEHOLD_ROLES)[number];

export interface Household {
  id: string;
  created_at: string;
  updated_at: string;
  name: string;
  type: HouseholdType;
}

export interface HouseholdMember {
  id: number;
  household_id: string;
  member_vcard_uid: string;
  role?: string;
  since?: string;
  until?: string;
}

export interface HouseholdWithMembers {
  household: Household;
  members: HouseholdMember[];
}

export interface HouseholdListResponse {
  households: Household[];
  total: number;
  page: number;
  limit: number;
  members?: HouseholdMember[];
}

export interface HouseholdInput {
  name: string;
  type: HouseholdType;
}

export interface HouseholdMemberInput {
  member_vcard_uid: string;
  role?: string;
  since?: string;
  until?: string;
}

export interface SuggestRelationshipsResponse {
  message: string;
  household_id: string;
  suggested_edges: RelationshipEdge[];
  total: number;
}

// GET /households
export async function listHouseholds(params?: {
  page?: number;
  limit?: number;
  include_members?: boolean;
}): Promise<HouseholdListResponse> {
  const { page = 1, limit = 100, include_members = false } = params || {};
  const queryParams = new URLSearchParams({
    page: page.toString(),
    limit: limit.toString(),
  });
  if (include_members) queryParams.append('include_members', 'true');
  const response = await apiFetch(
    `${API_BASE_URL}/households?${queryParams.toString()}`,
    { headers: getAuthHeaders() }
  );
  if (!response.ok) throw await parseErrorResponse(response);
  return response.json();
}

// POST /households
export async function createHousehold(input: HouseholdInput): Promise<Household> {
  const response = await apiFetch(`${API_BASE_URL}/households`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify(input),
  });
  if (!response.ok) throw await parseErrorResponse(response);
  const result = await response.json();
  return result.household;
}

// GET /households/:id
export async function getHousehold(id: string): Promise<HouseholdWithMembers> {
  const response = await apiFetch(`${API_BASE_URL}/households/${id}`, {
    headers: getAuthHeaders(),
  });
  if (!response.ok) throw await parseErrorResponse(response);
  return response.json();
}

// PUT /households/:id
export async function updateHousehold(id: string, input: HouseholdInput): Promise<Household> {
  const response = await apiFetch(`${API_BASE_URL}/households/${id}`, {
    method: 'PUT',
    headers: getAuthHeaders(),
    body: JSON.stringify(input),
  });
  if (!response.ok) throw await parseErrorResponse(response);
  return response.json();
}

// DELETE /households/:id
export async function deleteHousehold(id: string): Promise<void> {
  const response = await apiFetch(`${API_BASE_URL}/households/${id}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  });
  if (!response.ok) throw await parseErrorResponse(response);
}

// POST /households/:id/members
export async function addHouseholdMember(
  householdId: string,
  input: HouseholdMemberInput
): Promise<HouseholdMember> {
  const response = await apiFetch(`${API_BASE_URL}/households/${householdId}/members`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify(input),
  });
  if (!response.ok) throw await parseErrorResponse(response);
  const result = await response.json();
  return result.member;
}

// DELETE /households/:id/members/:vcard_uid
export async function removeHouseholdMember(
  householdId: string,
  memberVCardUid: string
): Promise<void> {
  const response = await apiFetch(
    `${API_BASE_URL}/households/${householdId}/members/${memberVCardUid}`,
    { method: 'DELETE', headers: getAuthHeaders() }
  );
  if (!response.ok) throw await parseErrorResponse(response);
}

// PATCH /households/:id/members/:vcard_uid — update a member's role in-place (T1 review B3+B4).
export async function updateHouseholdMember(
  householdId: string,
  memberVCardUid: string,
  role: string
): Promise<void> {
  const response = await apiFetch(
    `${API_BASE_URL}/households/${householdId}/members/${memberVCardUid}`,
    { method: 'PATCH', headers: getAuthHeaders(), body: JSON.stringify({ role }) }
  );
  if (!response.ok) throw await parseErrorResponse(response);
}

// POST /households/:id/suggest-relationships -- the trigger that produces
// suggested RelationshipEdges for every applicable member pair. Idempotent:
// re-running never duplicates edges (services.GenerateHouseholdSuggestions).
export async function suggestHouseholdRelationships(
  householdId: string
): Promise<SuggestRelationshipsResponse> {
  const response = await apiFetch(
    `${API_BASE_URL}/households/${householdId}/suggest-relationships`,
    { method: 'POST', headers: getAuthHeaders() }
  );
  if (!response.ok) throw await parseErrorResponse(response);
  return response.json();
}
