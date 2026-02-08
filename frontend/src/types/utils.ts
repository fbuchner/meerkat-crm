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
 * Response data that might be null or undefined
 */
export type Nullable<T> = T | null;
export type Optional<T> = T | undefined;
export type Maybe<T> = T | null | undefined;

