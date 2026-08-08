import { apiFetch, API_BASE_URL, getAuthHeaders } from './client';
import { handleResponse } from './errorHandling';

export type CardDAVDirection = 'two_way' | 'pull' | 'push';

export interface CardDAVConnection {
  id: number;
  base_url: string;
  address_book_path: string;
  address_book_name: string;
  username: string;
  has_password: boolean;
  direction: CardDAVDirection;
  sync_enabled: boolean;
  /** True while a sync started via startCardDAVSync is still running. */
  sync_running: boolean;
  last_synced_at: string | null;
  last_sync_status: '' | 'success' | 'partial' | 'error';
  last_sync_error: string;
  /** Counts from the last completed run, or null before the first one. */
  last_sync_stats: ContactSyncStats | null;
  created_at: string;
}

export interface CardDAVConnectionInput {
  base_url: string;
  address_book_path: string;
  address_book_name?: string;
  username?: string;
  password?: string;
  clear_password?: boolean;
  direction?: CardDAVDirection;
  sync_enabled?: boolean;
}

export interface DiscoveredAddressBook {
  path: string;
  name: string;
  description: string;
}

export interface ContactSyncStats {
  pulled_created: number;
  pulled_updated: number;
  pulled_deleted: number;
  pushed_created: number;
  pushed_updated: number;
  pushed_deleted: number;
  conflicts: number;
  skipped: number;
  errors: number;
}


// Returns null when no connection is configured yet (404).
export async function getCardDAVConnection(): Promise<CardDAVConnection | null> {
  const response = await apiFetch(`${API_BASE_URL}/carddav/connection`, {
    method: 'GET',
    headers: getAuthHeaders(),
  });
  if (response.status === 404) {
    return null;
  }
  return handleResponse(response, 'Unable to load the CardDAV connection.') as Promise<CardDAVConnection>;
}

export async function saveCardDAVConnection(
  input: CardDAVConnectionInput
): Promise<CardDAVConnection> {
  const response = await apiFetch(`${API_BASE_URL}/carddav/connection`, {
    method: 'PUT',
    headers: getAuthHeaders(),
    body: JSON.stringify(input),
  });
  return handleResponse(response, 'Unable to save the CardDAV connection.') as Promise<CardDAVConnection>;
}

export async function deleteCardDAVConnection(): Promise<void> {
  const response = await apiFetch(`${API_BASE_URL}/carddav/connection`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  });
  await handleResponse(response, 'Unable to disconnect the CardDAV server.');
}

export async function discoverCardDAVAddressBooks(input: {
  base_url: string;
  username?: string;
  password?: string;
}): Promise<DiscoveredAddressBook[]> {
  const response = await apiFetch(
    `${API_BASE_URL}/carddav/discover`,
    {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify(input),
    },
    60000
  );
  const data = await handleResponse(response, 'Unable to discover address books.');
  return data?.address_books || [];
}

/**
 * Starts a sync on the server and returns as soon as it has been accepted (202).
 * A first sync can run for minutes, so the outcome is not awaited here — poll
 * getCardDAVConnection and watch `sync_running`, then read `last_sync_status`
 * and `last_sync_stats`.
 */
export async function startCardDAVSync(): Promise<void> {
  const response = await apiFetch(`${API_BASE_URL}/carddav/connection/sync`, {
    method: 'POST',
    headers: getAuthHeaders(),
  });
  await handleResponse(response, 'Unable to start the contact sync.');
}
