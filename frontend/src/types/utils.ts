/**
 * Utility TypeScript types for Meerkat CRM
 * 
 * This file contains reusable utility types, form types, and common patterns
 * used throughout the application.
 */

/**
 * Makes specific properties of T required
 * @example
 * type User = { id?: number; name?: string; email?: string };
 * type RequiredUser = RequireFields<User, 'name' | 'email'>;
 * // Result: { id?: number; name: string; email: string }
 */
export type RequireFields<T, K extends keyof T> = T & Required<Pick<T, K>>;

/**
 * Makes specific properties of T optional
 * @example
 * type User = { id: number; name: string; email: string };
 * type PartialUser = OptionalFields<User, 'name' | 'email'>;
 * // Result: { id: number; name?: string; email?: string }
 */
export type OptionalFields<T, K extends keyof T> = Omit<T, K> & Partial<Pick<T, K>>;

/**
 * Extracts non-nullable values from T
 * @example
 * type MaybeString = string | null | undefined;
 * type DefiniteString = NonNullable<MaybeString>; // string
 */
export type NonNullableFields<T> = {
  [P in keyof T]: NonNullable<T[P]>;
};

/**
 * Generic form state wrapper
 * Useful for tracking form values, errors, and submission state
 */
export interface FormState<T> {
  values: T;
  errors: Partial<Record<keyof T, string>>;
  touched: Partial<Record<keyof T, boolean>>;
  isSubmitting: boolean;
  isValid: boolean;
}

/**
 * Generic async operation state
 * Tracks loading, error, and data states for async operations
 */
export interface AsyncState<T, E = Error> {
  data: T | null;
  loading: boolean;
  error: E | null;
}

/**
 * Pagination parameters
 */
export interface PaginationParams {
  page: number;
  limit: number;
}

/**
 * Paginated response wrapper
 */
export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  limit: number;
  hasMore?: boolean;
}

/**
 * Search and filter parameters
 */
export interface SearchParams {
  query: string;
  filters?: Record<string, string | number | boolean>;
}

/**
 * Sort parameters
 */
export interface SortParams {
  field: string;
  direction: 'asc' | 'desc';
}

/**
 * Common date range filter
 */
export interface DateRange {
  startDate: string;
  endDate: string;
}

/**
 * Generic dialog/modal state
 */
export interface DialogState<T = unknown> {
  open: boolean;
  data?: T;
  mode?: 'create' | 'edit' | 'view' | 'delete';
}

/**
 * Generic table column definition
 */
export interface TableColumn<T> {
  id: keyof T | string;
  label: string;
  sortable?: boolean;
  width?: number | string;
  align?: 'left' | 'center' | 'right';
  format?: (value: T[keyof T]) => string | React.ReactNode;
}

/**
 * Generic action button configuration
 */
export interface ActionButton {
  label: string;
  icon?: React.ReactNode;
  onClick: () => void;
  disabled?: boolean;
  variant?: 'text' | 'outlined' | 'contained';
  color?: 'primary' | 'secondary' | 'error' | 'warning' | 'info' | 'success';
}

/**
 * Notification/Snackbar message
 */
export interface Notification {
  message: string;
  severity: 'success' | 'error' | 'warning' | 'info';
  autoHideDuration?: number;
}

/**
 * Generic select option for dropdowns
 */
export interface SelectOption<T = string> {
  value: T;
  label: string;
  disabled?: boolean;
  group?: string;
}

/**
 * Event handler types for common scenarios
 */
export type ChangeHandler = (event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => void;
export type ClickHandler = (event: React.MouseEvent<HTMLButtonElement>) => void;
export type SubmitHandler = (event: React.FormEvent<HTMLFormElement>) => void;
export type SelectChangeHandler<T = string> = (value: T | null) => void;

/**
 * Branded types for type-safe IDs
 * Prevents accidentally mixing different ID types
 */
export type ContactId = number & { readonly __brand: 'ContactId' };
export type ActivityId = number & { readonly __brand: 'ActivityId' };
export type NoteId = number & { readonly __brand: 'NoteId' };
export type ReminderId = number & { readonly __brand: 'ReminderId' };
export type UserId = number & { readonly __brand: 'UserId' };

/**
 * Type guard helper functions
 */
export function isContactId(id: number): id is ContactId {
  return typeof id === 'number' && id > 0;
}

export function isActivityId(id: number): id is ActivityId {
  return typeof id === 'number' && id > 0;
}

export function isNoteId(id: number): id is NoteId {
  return typeof id === 'number' && id > 0;
}

/**
 * Response data that might be null or undefined
 */
export type Nullable<T> = T | null;
export type Optional<T> = T | undefined;
export type Maybe<T> = T | null | undefined;

/**
 * Deep partial - makes all nested properties optional
 */
export type DeepPartial<T> = {
  [P in keyof T]?: T[P] extends object ? DeepPartial<T[P]> : T[P];
};

/**
 * Deep readonly - makes all nested properties readonly
 */
export type DeepReadonly<T> = {
  readonly [P in keyof T]: T[P] extends object ? DeepReadonly<T[P]> : T[P];
};

/**
 * Extract keys of T where value type matches V
 * @example
 * type User = { id: number; name: string; age: number };
 * type StringKeys = KeysOfType<User, string>; // 'name'
 * type NumberKeys = KeysOfType<User, number>; // 'id' | 'age'
 */
export type KeysOfType<T, V> = {
  [K in keyof T]: T[K] extends V ? K : never;
}[keyof T];

/**
 * Mutable version of T (removes readonly)
 */
export type Mutable<T> = {
  -readonly [P in keyof T]: T[P];
};

/**
 * Promise that resolves to T
 */
export type AsyncData<T> = Promise<T>;

/**
 * Function that returns a promise
 */
export type AsyncFunction<T extends unknown[], R> = (...args: T) => Promise<R>;
