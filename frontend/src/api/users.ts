// User settings and preferences API calls
import { apiFetch, API_BASE_URL, getAuthHeaders } from './client';
import { handleResponse } from './errorHandling';

export async function updateLanguage(language: string): Promise<string> {
  const response = await apiFetch(`${API_BASE_URL}/users/language`, {
    method: 'PATCH',
    headers: getAuthHeaders(),
    body: JSON.stringify({ language }),
  });

  const data = await handleResponse(response, 'Unable to update language.');
  return data?.message || 'Language updated successfully.';
}

export async function updateDateFormat(dateFormat: string): Promise<string> {
  const response = await apiFetch(`${API_BASE_URL}/users/date-format`, {
    method: 'PATCH',
    headers: getAuthHeaders(),
    body: JSON.stringify({ date_format: dateFormat }),
  });

  const data = await handleResponse(response, 'Unable to update date format.');
  return data?.message || 'Date format updated successfully.';
}

export async function getCustomFieldNames(): Promise<string[]> {
  const response = await apiFetch(`${API_BASE_URL}/users/custom-fields`, {
    method: 'GET',
    headers: getAuthHeaders(),
  });

  const data = await handleResponse(response, 'Unable to get custom field names.');
  return data?.custom_field_names || [];
}

export async function updateCustomFieldNames(names: string[]): Promise<string[]> {
  const response = await apiFetch(`${API_BASE_URL}/users/custom-fields`, {
    method: 'PATCH',
    headers: getAuthHeaders(),
    body: JSON.stringify({ names }),
  });

  const data = await handleResponse(response, 'Unable to update custom field names.');
  return data?.custom_field_names || names;
}
