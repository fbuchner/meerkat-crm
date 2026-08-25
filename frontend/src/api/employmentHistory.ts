import { apiFetch, API_BASE_URL, getAuthHeaders, parseErrorResponse } from './client';

export interface EmploymentHistoryEntry {
  ID: number;
  CreatedAt: string;
  UpdatedAt: string;
  contact_id: number;
  organization: string;
  department?: string;
  job_title?: string;
  role?: string;
  start_date?: string;
  end_date?: string;
  notes?: string;
}

export interface EmploymentHistoryFormData {
  organization: string;
  department?: string;
  job_title?: string;
  role?: string;
  start_date?: string;
  end_date?: string;
  notes?: string;
}

export interface EmploymentHistoryResponse {
  employment_history: EmploymentHistoryEntry[];
}

export async function getEmploymentHistory(
  contactId: number | string
): Promise<EmploymentHistoryResponse> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/${contactId}/employment-history`,
    { headers: getAuthHeaders() }
  );
  if (!response.ok) throw await parseErrorResponse(response);
  return response.json();
}

export async function createEmploymentHistory(
  contactId: number | string,
  data: EmploymentHistoryFormData
): Promise<{ employment_history: EmploymentHistoryEntry }> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/${contactId}/employment-history`,
    { method: 'POST', headers: getAuthHeaders(), body: JSON.stringify(data) }
  );
  if (!response.ok) throw await parseErrorResponse(response);
  return response.json();
}

export async function updateEmploymentHistory(
  contactId: number | string,
  entryId: number,
  data: EmploymentHistoryFormData
): Promise<{ employment_history: EmploymentHistoryEntry }> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/${contactId}/employment-history/${entryId}`,
    { method: 'PUT', headers: getAuthHeaders(), body: JSON.stringify(data) }
  );
  if (!response.ok) throw await parseErrorResponse(response);
  return response.json();
}

export async function deleteEmploymentHistory(
  contactId: number | string,
  entryId: number
): Promise<void> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/${contactId}/employment-history/${entryId}`,
    { method: 'DELETE', headers: getAuthHeaders() }
  );
  if (!response.ok) throw await parseErrorResponse(response);
}

export async function endEmploymentHistory(
  contactId: number | string,
  entryId: number
): Promise<{ employment_history: EmploymentHistoryEntry }> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/${contactId}/employment-history/${entryId}/end`,
    { method: 'POST', headers: getAuthHeaders() }
  );
  if (!response.ok) throw await parseErrorResponse(response);
  return response.json();
}
