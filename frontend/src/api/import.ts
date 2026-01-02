import { apiFetch, API_BASE_URL, parseErrorResponse } from './client';

// Column mapping between CSV column and contact field
export interface ColumnMapping {
  csv_column: string;
  contact_field: string;
}

// Response from CSV upload
export interface ImportUploadResponse {
  session_id: string;
  headers: string[];
  suggested_mappings: ColumnMapping[];
  row_count: number;
  sample_data: string[][];
}

// Match info for duplicate detection
export interface DuplicateMatch {
  existing_contact_id: number;
  existing_firstname: string;
  existing_lastname: string;
  existing_email: string;
  match_reason: 'name' | 'email';
}

// Preview row with parsed contact and status
export interface ImportRowPreview {
  row_index: number;
  parsed_contact: Record<string, string>;
  validation_errors: string[];
  duplicate_match: DuplicateMatch | null;
  suggested_action: 'add' | 'skip' | 'update';
}

// Response from preview request
export interface ImportPreviewResponse {
  session_id: string;
  rows: ImportRowPreview[];
  total_rows: number;
  valid_rows: number;
  duplicate_count: number;
  error_count: number;
}

// Action for a specific row
export interface RowImportAction {
  row_index: number;
  action: 'skip' | 'add' | 'update';
}

// Final import result
export interface ImportResult {
  total_processed: number;
  created: number;
  updated: number;
  skipped: number;
  errors: string[];
}

// Contact fields that can be imported
export const IMPORTABLE_CONTACT_FIELDS = [
  'firstname',
  'lastname',
  'nickname',
  'gender',
  'email',
  'phone',
  'birthday',
  'address',
  'how_we_met',
  'food_preference',
  'work_information',
  'contact_information',
  'circles',
] as const;

// Human-readable labels for contact fields
export const CONTACT_FIELD_LABELS: Record<string, string> = {
  firstname: 'First Name',
  lastname: 'Last Name',
  nickname: 'Nickname',
  gender: 'Gender',
  email: 'Email',
  phone: 'Phone',
  birthday: 'Birthday',
  address: 'Address',
  how_we_met: 'How We Met',
  food_preference: 'Food Preferences',
  work_information: 'Work Information',
  contact_information: 'Contact Information',
  circles: 'Circles',
};

// Upload a CSV file for import
export async function uploadCSVForImport(
  file: File,
  token: string
): Promise<ImportUploadResponse> {
  const formData = new FormData();
  formData.append('file', file);

  const response = await apiFetch(`${API_BASE_URL}/contacts/import/upload`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
    },
    body: formData,
  });

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  return response.json();
}

// Get import preview with applied mappings
export async function getImportPreview(
  sessionId: string,
  mappings: ColumnMapping[],
  token: string
): Promise<ImportPreviewResponse> {
  const response = await apiFetch(`${API_BASE_URL}/contacts/import/preview`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      session_id: sessionId,
      mappings,
    }),
  });

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  return response.json();
}

// Confirm and execute the import
export async function confirmImport(
  sessionId: string,
  actions: RowImportAction[],
  token: string
): Promise<ImportResult> {
  const response = await apiFetch(`${API_BASE_URL}/contacts/import/confirm`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      session_id: sessionId,
      actions,
    }),
  });

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  return response.json();
}
