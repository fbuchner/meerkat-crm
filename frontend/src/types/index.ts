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
  ID: number;
  email: string;
  username?: string;
  CreatedAt: string;
  UpdatedAt: string;
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
