// Preference API calls -- T20a (docs/fork-plan/tickets/10-T20a-preferences.md,
// docs/fork-plan/91-envelope-data-model.md §91.9).
import { apiFetch, API_BASE_URL, getAuthHeaders, parseErrorResponse } from './client';

// Mirrors backend/models/preference.go's conventional (open) category set —
// the trailing "…" in §91.9 means this is NOT a closed enum; the frontend
// offers these known categories and free-forms via the dialog's "Other"
// path... actually no: category is free text on the wire (no oneof on the
// backend), so these are the curated labels we render, and an unrecognized
// value still displays via its raw token. No dynamic type-list endpoint
// exists anywhere in this codebase -- same hand-kept-sync convention as
// every other enum mirror here.
export const PREFERENCE_CATEGORIES = [
  'food',
  'drink',
  'clothing_size',
  'hobby',
  'gift',
  'dislike',
  'media',
] as const;

// Mirrors backend/models/preference.go's closed Source set (§91.9).
export type PreferenceSource = 'conversation_note' | 'user' | 'ai-suggested' | 'external';
// Mirrors the shared normal/private/secret set (RelationshipEdgeSensitivity).
export type PreferenceSensitivity = 'normal' | 'private' | 'secret';

export interface Preference {
  id: string;
  created_at: string;
  updated_at: string;
  entity_id: string;
  category: string;
  key?: string;
  value: string;
  source?: PreferenceSource;
  confidence?: number;
  last_confirmed?: string;
  sensitivity: PreferenceSensitivity;
}

export interface PreferenceInput {
  entity_id: string;
  category: string;
  key?: string;
  value: string;
  source?: PreferenceSource;
  confidence?: number;
  last_confirmed?: string;
  sensitivity?: PreferenceSensitivity;
}

export interface PreferencesResponse {
  preferences: Preference[];
  total: number;
  page: number;
  limit: number;
}

// GET /preferences
export async function getPreferences(params?: {
  entityId?: string;
  page?: number;
  limit?: number;
}): Promise<PreferencesResponse> {
  const { entityId, page = 1, limit = 100 } = params || {};
  const queryParams = new URLSearchParams({ page: page.toString(), limit: limit.toString() });
  if (entityId) queryParams.append('entity_id', entityId);
  const response = await apiFetch(
    `${API_BASE_URL}/preferences?${queryParams.toString()}`,
    { headers: getAuthHeaders() }
  );
  if (!response.ok) throw await parseErrorResponse(response);
  return response.json();
}

// POST /preferences
export async function createPreference(input: PreferenceInput): Promise<Preference> {
  const response = await apiFetch(`${API_BASE_URL}/preferences`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify(input),
  });
  if (!response.ok) throw await parseErrorResponse(response);
  const result = await response.json();
  return result.preference;
}

// PUT /preferences/:id (full-replace)
export async function updatePreference(id: string, input: PreferenceInput): Promise<Preference> {
  const response = await apiFetch(`${API_BASE_URL}/preferences/${id}`, {
    method: 'PUT',
    headers: getAuthHeaders(),
    body: JSON.stringify(input),
  });
  if (!response.ok) throw await parseErrorResponse(response);
  return response.json();
}

// DELETE /preferences/:id
export async function deletePreference(id: string): Promise<void> {
  const response = await apiFetch(`${API_BASE_URL}/preferences/${id}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  });
  if (!response.ok) throw await parseErrorResponse(response);
}
