import { apiFetch, API_BASE_URL, getAuthHeaders, parseErrorResponse } from './client';

// LifeEvent type tokens — must stay in sync with backend/models/life_event.go's
// LifeEventType* constants.
export const LIFE_EVENT_TYPES = [
  'married',
  'graduated',
  'job_change',
  'had_child',
  'adopted_pet',
  'retired',
  'moved',
] as const;
export type LifeEventType = (typeof LIFE_EVENT_TYPES)[number];

export interface PartialDate {
  year?: number;
  month?: number;
  day?: number;
}

export interface LifeEvent {
  id: string;
  created_at: string;
  updated_at: string;
  entity_id: string;
  type: string;
  date?: PartialDate;
  description?: string;
  source?: string;
  related_entity_ids?: string[];
  remind?: boolean;
}

export interface LifeEventCreateResponse {
  message: string;
  life_event: LifeEvent;
}

export interface LifeEventListResponse {
  life_events: LifeEvent[];
  total: number;
  page: number;
  limit: number;
}

export interface GetLifeEventsParams {
  entity_id?: string;
  page?: number;
  limit?: number;
}

export async function getLifeEvents(
  params: GetLifeEventsParams = {}
): Promise<LifeEventListResponse> {
  const { entity_id, page = 1, limit = 25 } = params;
  const queryParams = new URLSearchParams({
    page: page.toString(),
    limit: limit.toString(),
  });
  if (entity_id) queryParams.append('entity_id', entity_id);

  const response = await apiFetch(
    `${API_BASE_URL}/life-events?${queryParams.toString()}`,
    { headers: getAuthHeaders() }
  );
  if (!response.ok) throw await parseErrorResponse(response);
  return response.json();
}

export async function createLifeEvent(
  data: {
    entity_id: string;
    type: string;
    date?: PartialDate;
    description?: string;
    source?: string;
    related_entity_ids?: string[];
    remind?: boolean;
  }
): Promise<LifeEventCreateResponse> {
  const response = await apiFetch(`${API_BASE_URL}/life-events`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify(data),
  });
  if (!response.ok) throw await parseErrorResponse(response);
  return response.json();
}

export async function updateLifeEvent(
  id: string,
  data: {
    entity_id: string;
    type: string;
    date?: PartialDate;
    description?: string;
    source?: string;
    related_entity_ids?: string[];
    remind?: boolean;
  }
): Promise<LifeEvent> {
  const response = await apiFetch(`${API_BASE_URL}/life-events/${id}`, {
    method: 'PUT',
    headers: getAuthHeaders(),
    body: JSON.stringify(data),
  });
  if (!response.ok) throw await parseErrorResponse(response);
  return response.json();
}

export async function deleteLifeEvent(id: string): Promise<void> {
  const response = await apiFetch(`${API_BASE_URL}/life-events/${id}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  });
  if (!response.ok) throw await parseErrorResponse(response);
}

export function partialDateDisplay(date?: PartialDate): string {
  if (!date) return '';
  const y = date.year != null ? String(date.year) : '';
  const m = date.month != null ? String(date.month).padStart(2, '0') : '';
  const d = date.day != null ? String(date.day).padStart(2, '0') : '';
  if (y && m && d) return `${y}-${m}-${d}`;
  if (y) return y;
  if (m && d) return `${m}/${d}`;
  if (m) return `${m}/??`;
  return '';
}

export function partialDateHasMonthDay(date?: PartialDate): boolean {
  return date != null && date.month != null && date.day != null;
}

export function partialDateIsYearOnly(date?: PartialDate): boolean {
  return date != null && date.year != null && date.month == null && date.day == null;
}
