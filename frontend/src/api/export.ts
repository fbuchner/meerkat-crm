// Export API functions
import { API_BASE_URL, apiFetch, parseErrorResponse } from './client';
import { getToken } from '../auth';

/**
 * Export all user data as CSV
 * Downloads a CSV file containing all contacts, activities, notes, relationships, and reminders
 */
export async function exportDataAsCsv(): Promise<void> {
  const token = getToken();
  
  const response = await apiFetch(`${API_BASE_URL}/export`, {
    method: 'GET',
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    const error = await parseErrorResponse(response);
    throw error;
  }

  // Get the filename from Content-Disposition header or use default
  const contentDisposition = response.headers.get('Content-Disposition');
  let filename = 'meerkat-export.csv';
  if (contentDisposition) {
    const filenameMatch = contentDisposition.match(/filename=([^;]+)/);
    if (filenameMatch) {
      filename = filenameMatch[1].replace(/"/g, '').trim();
    }
  }

  // Get the blob data
  const blob = await response.blob();
  
  // Create a download link and trigger download
  const url = window.URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  window.URL.revokeObjectURL(url);
}
