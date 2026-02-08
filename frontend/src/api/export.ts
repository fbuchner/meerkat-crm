// Export API functions
import { API_BASE_URL, apiFetch, getAuthHeaders, parseErrorResponse } from './client';

/**
 * Helper function to download a file from an API response
 */
async function downloadFileFromResponse(response: Response, defaultFilename: string): Promise<void> {
  // Get the filename from Content-Disposition header or use default
  const contentDisposition = response.headers.get('Content-Disposition');
  let filename = defaultFilename;
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

/**
 * Export all user data as CSV
 * Downloads a CSV file containing all contacts, activities, notes, relationships, and reminders
 */
export async function exportDataAsCsv(): Promise<void> {
  const response = await apiFetch(`${API_BASE_URL}/export`, {
    method: 'GET',
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    const error = await parseErrorResponse(response);
    throw error;
  }

  await downloadFileFromResponse(response, 'meerkat-export.csv');
}

/**
 * Export all contacts as VCF (vCard)
 * Downloads a VCF file containing all contacts with their photos
 */
export async function exportContactsAsVcf(): Promise<void> {
  const response = await apiFetch(`${API_BASE_URL}/export/vcf`, {
    method: 'GET',
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    const error = await parseErrorResponse(response);
    throw error;
  }

  await downloadFileFromResponse(response, 'meerkat-contacts.vcf');
}
