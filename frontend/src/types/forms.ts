/**
 * Form-specific TypeScript types for Meerkat CRM
 * 
 * This file contains types specifically for form handling,
 * validation, and form-related operations.
 */

import type { Contact } from './index';

/**
 * Form data for creating a new contact
 * Omits server-generated fields (ID, timestamps)
 */
export type ContactFormData = Omit<Contact, 'ID'>;

/**
 * Form data for updating a contact
 * All fields are optional except ID
 */
export type ContactUpdateFormData = Partial<Contact> & Pick<Contact, 'ID'>;

/**
 * Form data for creating a new activity
 */
export interface ActivityFormData {
  title: string;
  description?: string;
  location?: string;
  date: string;
  contact_ids: number[];
}

/**
 * Form data for updating an activity
 */
export type ActivityUpdateFormData = Partial<ActivityFormData> & { ID: number };

/**
 * Form data for creating a new note
 */
export interface NoteFormData {
  title: string;
  content: string;
  date: string;
  contact_id?: number;
}

/**
 * Form data for updating a note
 */
export type NoteUpdateFormData = Partial<NoteFormData> & { ID: number };

/**
 * Form data for creating a reminder
 */
export interface ReminderFormData {
  title: string;
  description?: string;
  date: string;
  contact_id?: number;
}

/**
 * Form data for updating a reminder
 */
export type ReminderUpdateFormData = Partial<ReminderFormData> & { ID: number };

/**
 * Login form data
 */
export interface LoginFormData {
  username: string;
  password: string;
}

/**
 * Registration form data
 */
export interface RegisterFormData {
  username: string;
  email: string;
  password: string;
  confirmPassword: string;
}

/**
 * Password reset request form
 */
export interface PasswordResetRequestFormData {
  email: string;
}

/**
 * Password reset form (with token)
 */
export interface PasswordResetFormData {
  token: string;
  password: string;
  confirmPassword: string;
}

/**
 * Search form data
 */
export interface SearchFormData {
  query: string;
  searchIn: 'contacts' | 'notes' | 'activities' | 'all';
  filters?: {
    circles?: string[];
    dateFrom?: string;
    dateTo?: string;
  };
}

/**
 * Contact filter form data
 */
export interface ContactFilterFormData {
  search: string;
  circle?: string;
  gender?: string;
  hasBirthday?: boolean;
  hasEmail?: boolean;
  hasPhone?: boolean;
}

/**
 * Activity filter form data
 */
export interface ActivityFilterFormData {
  search: string;
  contactId?: number;
  dateFrom?: string;
  dateTo?: string;
  hasLocation?: boolean;
}

/**
 * Note filter form data
 */
export interface NoteFilterFormData {
  search: string;
  contactId?: number;
  dateFrom?: string;
  dateTo?: string;
}

/**
 * Generic validation error for a field
 */
export interface FieldError {
  field: string;
  message: string;
}

/**
 * Form validation result
 */
export interface ValidationResult {
  isValid: boolean;
  errors: FieldError[];
}

/**
 * Field validators
 */
export type FieldValidator<T> = (value: T) => string | undefined;

/**
 * Form field configuration
 */
export interface FormField<T = string> {
  name: string;
  label: string;
  type: 'text' | 'email' | 'password' | 'number' | 'date' | 'textarea' | 'select' | 'autocomplete';
  value: T;
  required?: boolean;
  disabled?: boolean;
  placeholder?: string;
  helperText?: string;
  error?: string;
  validators?: FieldValidator<T>[];
  options?: Array<{ value: T; label: string }>; // For select/autocomplete
}

/**
 * Generic form configuration
 */
export interface FormConfig<T extends Record<string, unknown>> {
  fields: Array<FormField<T[keyof T]>>;
  onSubmit: (data: T) => void | Promise<void>;
  onCancel?: () => void;
  submitLabel?: string;
  cancelLabel?: string;
}

/**
 * File upload form data
 */
export interface FileUploadData {
  file: File;
  contactId?: number;
  type: 'avatar' | 'document' | 'photo';
}

/**
 * Bulk operation form data
 */
export interface BulkOperationFormData {
  action: 'delete' | 'export' | 'update' | 'tag';
  selectedIds: number[];
  parameters?: Record<string, unknown>;
}

/**
 * Import form data
 */
export interface ImportFormData {
  file: File;
  format: 'csv' | 'vcf' | 'json';
  mapping?: Record<string, string>; // Maps CSV columns to Contact fields
  skipErrors: boolean;
}

/**
 * Export form data
 */
export interface ExportFormData {
  format: 'csv' | 'vcf' | 'json' | 'pdf';
  selectedIds?: number[];
  includeNotes?: boolean;
  includeActivities?: boolean;
  dateRange?: {
    from: string;
    to: string;
  };
}

/**
 * Settings form data
 */
export interface SettingsFormData {
  theme: 'light' | 'dark' | 'auto';
  language: string;
  notifications: {
    email: boolean;
    birthday: boolean;
    reminders: boolean;
  };
  privacy: {
    shareData: boolean;
    analytics: boolean;
  };
}

/**
 * Form submission result
 */
export interface FormSubmissionResult<T = unknown> {
  success: boolean;
  data?: T;
  error?: string;
  validationErrors?: FieldError[];
}
