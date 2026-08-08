import { apiFetch, API_BASE_URL, getAuthHeaders, parseErrorResponse } from './client';
import type { ImportRowPreview, ImportResult, RowImportAction } from './import';

// Per-entity totals of the connected Monica account
export interface MonicaEntityCounts {
  contacts: number;
  activities: number;
  notes: number;
  reminders: number;
  calls: number;
  tasks: number;
  gifts: number;
  debts: number;
}

// Response from connecting to a Monica instance
export interface MonicaConnectResponse {
  session_id: string;
  totals: MonicaEntityCounts;
  estimated_fetch_seconds: number;
}

// Phases of the fetch/import pipeline, reported by the status endpoint
export type MonicaPhase =
  | 'connecting'
  | 'fetching_contacts'
  | 'fetching_activities'
  | 'fetching_notes'
  | 'fetching_reminders'
  | 'fetching_extras'
  | 'fetching_relationships'
  | 'building_preview'
  | 'ready'
  | 'importing'
  | 'importing_photos'
  | 'done'
  | 'failed';

// Polling payload for fetch and import progress
export interface MonicaImportStatus {
  session_id: string;
  phase: MonicaPhase;
  phase_done: number;
  phase_total: number;
  error?: string;
  result?: MonicaImportResult;
}

// Related entities that would be imported alongside one contact
export interface MonicaRowRelatedCounts {
  activities: number;
  notes: number;
  reminders: number;
  relationships: number;
  extra_notes: number;
}

// Preview row extended with Monica-specific context
export interface MonicaImportRowPreview extends ImportRowPreview {
  related: MonicaRowRelatedCounts;
  has_photo: boolean;
}

// Full preview of a fetched Monica snapshot
export interface MonicaPreviewResponse {
  session_id: string;
  rows: MonicaImportRowPreview[];
  total_rows: number;
  valid_rows: number;
  duplicate_count: number;
  error_count: number;
  totals: MonicaRowRelatedCounts;
}

// Import result extended with related-entity and photo counts
export interface MonicaImportResult extends ImportResult {
  activities_created: number;
  notes_created: number;
  reminders_created: number;
  relationships_created: number;
  relationships_collapsed: number;
  photos_queued: number;
  photos_saved: number;
  photos_failed: number;
}

// Connect makes nine serialized requests to Monica at ~1/second, and a slow
// self-hosted instance or a single rate-limit backoff pushes that well past
// the default 30s timeout. Timing out here is worse than slow: the backend
// still completes and holds the session — with the API token — while the
// browser never learns the session id needed to use or cancel it.
const CONNECT_TIMEOUT_MS = 180000;

// Validate the Monica URL + API token and create an import session
export async function monicaConnect(
  baseUrl: string,
  apiToken: string
): Promise<MonicaConnectResponse> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/import/monica/connect`,
    {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify({ base_url: baseUrl, api_token: apiToken }),
    },
    CONNECT_TIMEOUT_MS
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  return response.json();
}

// Start the background snapshot fetch
export async function monicaStartFetch(
  sessionId: string,
  options: {
    include_relationships: boolean;
    include_extras: boolean;
    // Monica stores each relationship on both contacts. When set, only one
    // row per pair is created; the other contact shows it as incoming.
    collapse_reciprocal_relationships: boolean;
  }
): Promise<void> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/import/monica/fetch`,
    {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify({ session_id: sessionId, ...options }),
    }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }
}

// Poll fetch/import progress
export async function monicaGetStatus(
  sessionId: string
): Promise<MonicaImportStatus> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/import/monica/status?session_id=${encodeURIComponent(sessionId)}`
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  return response.json();
}

// Load the contact preview once the snapshot is ready
export async function monicaGetPreview(
  sessionId: string
): Promise<MonicaPreviewResponse> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/import/monica/preview?session_id=${encodeURIComponent(sessionId)}`
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  return response.json();
}

// Execute the import with per-contact actions
export async function monicaConfirm(
  sessionId: string,
  actions: RowImportAction[]
): Promise<MonicaImportResult> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/import/monica/confirm`,
    {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify({ session_id: sessionId, actions }),
    }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  return response.json();
}

// Cancel a session, dropping the API token server-side
export async function monicaCancel(sessionId: string): Promise<void> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/import/monica/session?session_id=${encodeURIComponent(sessionId)}`,
    { method: 'DELETE' }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }
}
