import { apiFetch, API_BASE_URL, getAuthHeaders } from './client';
import { handleResponse } from './errorHandling';

export interface CalendarSubscription {
  id: number;
  name: string;
  url: string;
  username: string;
  has_password: boolean;
  sync_enabled: boolean;
  past_days: number;
  future_days: number;
  last_synced_at: string | null;
  last_sync_status: '' | 'success' | 'error';
  last_sync_error: string;
  created_at: string;
}

export interface CalendarSubscriptionInput {
  name: string;
  url: string;
  username?: string;
  password?: string;
  clear_password?: boolean;
  sync_enabled?: boolean;
  past_days?: number;
  future_days?: number;
}

export interface CalendarSyncResult {
  message: string;
  created: number;
  updated: number;
  skipped: number;
}

export async function getCalendarSubscriptions(): Promise<CalendarSubscription[]> {
  const response = await apiFetch(`${API_BASE_URL}/calendars`, {
    method: 'GET',
    headers: getAuthHeaders(),
  });
  const data = await handleResponse(response, 'Unable to load calendar subscriptions.');
  return data?.calendars || [];
}

export async function createCalendarSubscription(
  input: CalendarSubscriptionInput
): Promise<CalendarSubscription> {
  const response = await apiFetch(`${API_BASE_URL}/calendars`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify(input),
  });
  return handleResponse(response, 'Unable to add calendar.') as Promise<CalendarSubscription>;
}

export async function updateCalendarSubscription(
  id: number,
  input: CalendarSubscriptionInput
): Promise<CalendarSubscription> {
  const response = await apiFetch(`${API_BASE_URL}/calendars/${id}`, {
    method: 'PUT',
    headers: getAuthHeaders(),
    body: JSON.stringify(input),
  });
  return handleResponse(response, 'Unable to update calendar.') as Promise<CalendarSubscription>;
}

export async function deleteCalendarSubscription(id: number): Promise<void> {
  const response = await apiFetch(`${API_BASE_URL}/calendars/${id}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  });
  await handleResponse(response, 'Unable to delete calendar.');
}

export async function syncCalendarSubscription(id: number): Promise<CalendarSyncResult> {
  const response = await apiFetch(
    `${API_BASE_URL}/calendars/${id}/sync`,
    {
      method: 'POST',
      headers: getAuthHeaders(),
    },
    120000
  );
  return handleResponse(response, 'Unable to sync calendar.') as Promise<CalendarSyncResult>;
}
