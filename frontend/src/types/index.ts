/**
 * Central type definitions for the Meerkat CRM application
 * 
 * This file consolidates all TypeScript interfaces and types used across the application.
 * Types are re-exported from their source modules for backwards compatibility.
 */

// Re-export types from API modules
export type {
  Contact,
  ContactsResponse,
  GetContactsParams,
  Birthday,
} from '../api/contacts';

export type {
  Note,
  NotesResponse,
} from '../api/notes';

export type {
  Activity,
  ActivityContact,
  ActivitiesResponse,
  GetActivitiesParams,
} from '../api/activities';

// Additional shared types

/**
 * Error object returned from API calls
 */
export interface ApiError {
  message: string;
  statusCode?: number;
  details?: unknown;
}

/**
 * Generic API response wrapper
 */
export interface ApiResponse<T> {
  data?: T;
  error?: ApiError;
}

/**
 * User authentication response
 */
export interface AuthResponse {
  token: string;
  user: User;
}

/**
 * User profile information
 */
export interface User {
  id: number;
  email: string;
  username: string;
  language: string;
  is_admin: boolean;
  created_at: string;
  updated_at: string;
}

/**
 * Paginated list of users for admin
 */
export interface UsersListResponse {
  users: User[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
}

/**
 * Input for updating a user
 */
export interface UserUpdateInput {
  username?: string;
  email?: string;
  password?: string;
  is_admin?: boolean;
}

/**
 * Form validation error
 */
export interface ValidationError {
  field: string;
  message: string;
}

/**
 * Reminder data structure
 */
export interface Reminder {
  ID: number;
  title: string;
  description?: string;
  date: string;
  contact_id?: number;
  sent: boolean;
  CreatedAt: string;
  UpdatedAt: string;
}

/**
 * Relationship between contacts
 */
export interface Relationship {
  ID: number;
  contact_id: number;
  related_contact_id: number;
  relationship_type: string;
  CreatedAt: string;
  UpdatedAt: string;
}

/**
 * Photo metadata
 */
export interface Photo {
  id: string;
  url: string;
  thumbnailUrl?: string;
  contactId: number;
  uploadedAt: string;
}

// Re-export utility types
export type {
  RequireFields,
  OptionalFields,
  NonNullableFields,
  FormState,
  AsyncState,
  PaginationParams,
  PaginatedResponse,
  SearchParams,
  SortParams,
  DateRange,
  DialogState,
  TableColumn,
  ActionButton,
  Notification,
  SelectOption,
  ChangeHandler,
  ClickHandler,
  SubmitHandler,
  SelectChangeHandler,
  ContactId,
  ActivityId,
  NoteId,
  ReminderId,
  UserId,
  Nullable,
  Optional,
  Maybe,
  DeepPartial,
  DeepReadonly,
  KeysOfType,
  Mutable,
  AsyncData,
  AsyncFunction,
} from './utils';

// Re-export form types
export type {
  ContactFormData,
  ContactUpdateFormData,
  ActivityFormData,
  ActivityUpdateFormData,
  NoteFormData,
  NoteUpdateFormData,
  ReminderFormData,
  ReminderUpdateFormData,
  LoginFormData,
  RegisterFormData,
  PasswordResetRequestFormData,
  PasswordResetFormData,
  SearchFormData,
  ContactFilterFormData,
  ActivityFilterFormData,
  NoteFilterFormData,
  FieldError,
  ValidationResult,
  FieldValidator,
  FormField,
  FormConfig,
  FileUploadData,
  BulkOperationFormData,
  ImportFormData,
  ExportFormData,
  SettingsFormData,
  FormSubmissionResult,
} from './forms';

// Re-export API types
export type {
  ApiErrorResponse,
  ApiSuccessResponse,
  LoginResponse,
  RegisterResponse,
  GetContactsResponse,
  GetContactResponse,
  CreateContactResponse,
  UpdateContactResponse,
  DeleteContactResponse,
  GetActivitiesResponse,
  GetActivityResponse,
  CreateActivityResponse,
  UpdateActivityResponse,
  DeleteActivityResponse,
  GetNotesResponse,
  GetNoteResponse,
  CreateNoteResponse,
  UpdateNoteResponse,
  DeleteNoteResponse,
  GetRemindersResponse,
  GetReminderResponse,
  CreateReminderResponse,
  UpdateReminderResponse,
  DeleteReminderResponse,
  GetRelationshipsResponse,
  CreateRelationshipResponse,
  DeleteRelationshipResponse,
  UploadPhotoResponse,
  HealthCheckResponse,
  SearchResponse,
  GetCirclesResponse,
  StatisticsResponse,
  ExportResponse,
  ImportResponse,
  BatchOperationResponse,
  HttpMethod,
  ApiRequestConfig,
  TypedFetchOptions,
  ApiEndpoint,
} from './api';

export { API_ENDPOINTS } from './api';
