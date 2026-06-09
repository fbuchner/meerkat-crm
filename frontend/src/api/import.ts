import { apiFetch, API_BASE_URL, getAuthHeaders, parseErrorResponse } from './client';

// Column mapping between CSV column and contact field
export interface ColumnMapping {
  csv_column: string;
  contact_field: string;
  // Multi-value entry index (0-based). Ties a value column to its label/parts within the
  // same logical entry (e.g. "E-mail 1 - Value" + "E-mail 1 - Label")
  group: number;
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

// Contact fields that can be imported (mirrors backend models.ImportableContactFields)
export const IMPORTABLE_CONTACT_FIELDS = [
  // Name
  'firstname',
  'lastname',
  'middle_name',
  'prefix',
  'suffix',
  'nickname',
  'gender',
  // Dates
  'birthday',
  'anniversary',
  // Multi-value: email / phone
  'email',
  'email_label',
  'phone',
  'phone_label',
  // Multi-value: address parts
  'address_street',
  'address_city',
  'address_region',
  'address_postal',
  'address_country',
  'address_label',
  // Multi-value: web / IM
  'url',
  'url_label',
  'impp',
  'impp_label',
  // Organization
  'organization',
  'department',
  'job_title',
  'role',
  // Free-text
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
  middle_name: 'Middle Name',
  prefix: 'Name Prefix',
  suffix: 'Name Suffix',
  nickname: 'Nickname',
  gender: 'Gender',
  birthday: 'Birthday',
  anniversary: 'Anniversary',
  email: 'Email',
  email_label: 'Email – Type',
  phone: 'Phone',
  phone_label: 'Phone – Type',
  address_street: 'Address – Street',
  address_city: 'Address – City',
  address_region: 'Address – Region/State',
  address_postal: 'Address – Postal Code',
  address_country: 'Address – Country',
  address_label: 'Address – Type',
  url: 'Website',
  url_label: 'Website – Type',
  impp: 'IM / Social',
  impp_label: 'IM / Social – Type',
  organization: 'Organization',
  department: 'Department',
  job_title: 'Job Title',
  role: 'Role',
  how_we_met: 'How We Met',
  food_preference: 'Food Preferences',
  work_information: 'Work Information',
  contact_information: 'Contact Information',
  circles: 'Circles',
};

// Upload a CSV file for import
export async function uploadCSVForImport(
  file: File
): Promise<ImportUploadResponse> {
  const formData = new FormData();
  formData.append('file', file);

  const response = await apiFetch(`${API_BASE_URL}/contacts/import/upload`, {
    method: 'POST',
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
  mappings: ColumnMapping[]
): Promise<ImportPreviewResponse> {
  const response = await apiFetch(`${API_BASE_URL}/contacts/import/preview`, {
    method: 'POST',
    headers: getAuthHeaders(),
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

// Confirm and execute the import (CSV)
export async function confirmImport(
  sessionId: string,
  actions: RowImportAction[]
): Promise<ImportResult> {
  const response = await apiFetch(`${API_BASE_URL}/contacts/import/confirm`, {
    method: 'POST',
    headers: getAuthHeaders(),
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

// Upload a VCF file for import (returns preview directly, no mapping needed)
export async function uploadVCFForImport(
  file: File
): Promise<ImportPreviewResponse> {
  const formData = new FormData();
  formData.append('file', file);

  const response = await apiFetch(`${API_BASE_URL}/contacts/import/vcf/upload`, {
    method: 'POST',
    body: formData,
  });

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  return response.json();
}

// Confirm and execute VCF import (with photo processing)
export async function confirmVCFImport(
  sessionId: string,
  actions: RowImportAction[]
): Promise<ImportResult> {
  const response = await apiFetch(`${API_BASE_URL}/contacts/import/vcf/confirm`, {
    method: 'POST',
    headers: getAuthHeaders(),
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
