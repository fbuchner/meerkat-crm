import { apiFetch, API_BASE_URL, getAuthHeaders } from './client';
import { handleResponse } from './errorHandling';

export interface Webhook {
  id: number;
  name: string;
  url: string;
  events: string[];
  is_active: boolean;
  created_at: string;
}

export interface WebhookCreateResponse extends Webhook {
  secret: string;
}

export interface WebhookDelivery {
  id: number;
  event_type: string;
  status_code: number | null;
  error: string | null;
  attempts: number;
  created_at: string;
}

export async function getWebhooks(): Promise<Webhook[]> {
  const response = await apiFetch(`${API_BASE_URL}/webhooks`, {
    method: 'GET',
    headers: getAuthHeaders(),
  });
  const data = await handleResponse(response, 'Unable to load webhooks.');
  return data?.webhooks || [];
}

export async function createWebhook(input: {
  name: string;
  url: string;
  events: string[];
  is_active: boolean;
}): Promise<WebhookCreateResponse> {
  const response = await apiFetch(`${API_BASE_URL}/webhooks`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify(input),
  });
  return handleResponse(response, 'Unable to create webhook.') as Promise<WebhookCreateResponse>;
}

export async function updateWebhook(
  id: number,
  input: { name: string; url: string; events: string[]; is_active: boolean }
): Promise<Webhook> {
  const response = await apiFetch(`${API_BASE_URL}/webhooks/${id}`, {
    method: 'PUT',
    headers: getAuthHeaders(),
    body: JSON.stringify(input),
  });
  return handleResponse(response, 'Unable to update webhook.') as Promise<Webhook>;
}

export async function deleteWebhook(id: number): Promise<void> {
  const response = await apiFetch(`${API_BASE_URL}/webhooks/${id}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  });
  await handleResponse(response, 'Unable to delete webhook.');
}

export async function testWebhook(id: number): Promise<{ delivery: WebhookDelivery }> {
  const response = await apiFetch(`${API_BASE_URL}/webhooks/${id}/test`, {
    method: 'POST',
    headers: getAuthHeaders(),
  });
  return handleResponse(response, 'Unable to test webhook.') as Promise<{ delivery: WebhookDelivery }>;
}

export async function getWebhookDeliveries(id: number): Promise<WebhookDelivery[]> {
  const response = await apiFetch(`${API_BASE_URL}/webhooks/${id}/deliveries`, {
    method: 'GET',
    headers: getAuthHeaders(),
  });
  const data = await handleResponse(response, 'Unable to load deliveries.');
  return data?.deliveries || [];
}
