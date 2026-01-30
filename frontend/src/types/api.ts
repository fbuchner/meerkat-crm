/**
 * API Request and Response TypeScript types
 * 
 * This file defines the shape of all API requests and responses,
 * ensuring type safety when communicating with the backend.
 */

import type {
  Contact,
  Activity,
  Note,
  Reminder,
  User,
  Relationship,
} from './index';

/**
 * Generic API error response
 */
export interface ApiErrorResponse {
  error: string;
  message?: string;
  statusCode?: number;
  validationErrors?: Array<{
    field: string;
    message: string;
  }>;
}

/**
 * Generic API success response wrapper
 */
export interface ApiSuccessResponse<T = unknown> {
  message?: string;
  data?: T;
}

/**
 * Authentication responses
 */
export interface LoginResponse {
  token: string;
  user: User;
}

export interface RegisterResponse {
  message: string;
  user: User;
}

/**
 * Contact API responses
 */
export interface GetContactsResponse {
  contacts: Contact[];
  total: number;
  page: number;
  limit: number;
}

export interface GetContactResponse extends Contact {}

export interface CreateContactResponse {
  message: string;
  contact: Contact;
}

export interface UpdateContactResponse {
  message: string;
  contact: Contact;
}

export interface DeleteContactResponse {
  message: string;
}

/**
 * Activity API responses
 */
export interface GetActivitiesResponse {
  activities: Activity[];
  total: number;
  page: number;
  limit: number;
}

export interface GetActivityResponse extends Activity {}

export interface CreateActivityResponse {
  message: string;
  activity: Activity;
}

export interface UpdateActivityResponse {
  message: string;
  activity: Activity;
}

export interface DeleteActivityResponse {
  message: string;
}

/**
 * Note API responses
 */
export interface GetNotesResponse {
  notes: Note[];
  total: number;
  page: number;
  limit: number;
}

export interface GetNoteResponse extends Note {}

export interface CreateNoteResponse {
  message: string;
  note: Note;
}

export interface UpdateNoteResponse {
  message: string;
  note: Note;
}

export interface DeleteNoteResponse {
  message: string;
}

/**
 * Reminder API responses
 */
export interface GetRemindersResponse {
  reminders: Reminder[];
  total: number;
  page: number;
  limit: number;
}

export interface GetReminderResponse extends Reminder {}

export interface CreateReminderResponse {
  message: string;
  reminder: Reminder;
}

export interface UpdateReminderResponse {
  message: string;
  reminder: Reminder;
}

export interface DeleteReminderResponse {
  message: string;
}

/**
 * Relationship API responses
 */
export interface GetRelationshipsResponse {
  relationships: Relationship[];
}

export interface CreateRelationshipResponse {
  message: string;
  relationship: Relationship;
}

export interface DeleteRelationshipResponse {
  message: string;
}

/**
 * Photo upload response
 */
export interface UploadPhotoResponse {
  message: string;
  photo_url: string;
  photo_thumbnail?: string;
}

/**
 * Health check response
 */
export interface HealthCheckResponse {
  status: 'ok' | 'error';
  timestamp: string;
  version?: string;
}

/**
 * Search response
 */
export interface SearchResponse {
  contacts?: Contact[];
  activities?: Activity[];
  notes?: Note[];
  total: number;
}

/**
 * Circles/Tags response
 */
export interface GetCirclesResponse {
  circles: string[];
}

/**
 * Statistics response
 */
export interface StatisticsResponse {
  totalContacts: number;
  totalActivities: number;
  totalNotes: number;
  totalReminders: number;
  upcomingBirthdays: Array<{
    contact: Contact;
    daysUntil: number;
  }>;
  recentActivities: Activity[];
}

/**
 * Export response
 */
export interface ExportResponse {
  data: string; // CSV, VCF, or JSON string
  filename: string;
  mimeType: string;
}

/**
 * Import response
 */
export interface ImportResponse {
  message: string;
  imported: number;
  failed: number;
  errors?: Array<{
    row: number;
    error: string;
  }>;
}

/**
 * Batch operation response
 */
export interface BatchOperationResponse {
  message: string;
  successful: number;
  failed: number;
  errors?: Array<{
    id: number;
    error: string;
  }>;
}

/**
 * HTTP methods for API requests
 */
export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';

/**
 * API request configuration
 */
export interface ApiRequestConfig {
  method: HttpMethod;
  headers?: Record<string, string>;
  body?: unknown;
  params?: Record<string, string | number | boolean>;
  timeout?: number;
}

/**
 * Typed fetch options
 */
export interface TypedFetchOptions extends Omit<RequestInit, 'body'> {
  body?: unknown;
}

/**
 * API endpoint URLs (type-safe paths)
 */
export const API_ENDPOINTS = {
  auth: {
    login: '/auth/login',
    register: '/auth/register',
    logout: '/auth/logout',
    refresh: '/auth/refresh',
  },
  contacts: {
    list: '/contacts',
    get: (id: number) => `/contacts/${id}`,
    create: '/contacts',
    update: (id: number) => `/contacts/${id}`,
    delete: (id: number) => `/contacts/${id}`,
    activities: (id: number) => `/contacts/${id}/activities`,
    notes: (id: number) => `/contacts/${id}/notes`,
    photo: (id: number) => `/contacts/${id}/photo`,
  },
  activities: {
    list: '/activities',
    get: (id: number) => `/activities/${id}`,
    create: '/activities',
    update: (id: number) => `/activities/${id}`,
    delete: (id: number) => `/activities/${id}`,
  },
  notes: {
    list: '/notes',
    get: (id: number) => `/notes/${id}`,
    create: '/notes',
    update: (id: number) => `/notes/${id}`,
    delete: (id: number) => `/notes/${id}`,
  },
  reminders: {
    list: '/reminders',
    get: (id: number) => `/reminders/${id}`,
    create: '/reminders',
    update: (id: number) => `/reminders/${id}`,
    delete: (id: number) => `/reminders/${id}`,
  },
  relationships: {
    list: (contactId: number) => `/contacts/${contactId}/relationships`,
    create: '/relationships',
    delete: (id: number) => `/relationships/${id}`,
  },
  health: '/health',
  search: '/search',
  circles: '/circles',
} as const;

/**
 * Type for API endpoint paths
 */
export type ApiEndpoint = typeof API_ENDPOINTS;
