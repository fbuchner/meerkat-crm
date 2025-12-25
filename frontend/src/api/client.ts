// API client with authentication and error handling
import { getToken } from '../auth';

// API base URL with versioning
const API_SERVER_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';
export const API_BASE_URL = `${API_SERVER_URL}/api/v1`;

// Default request timeout in milliseconds (30 seconds)
const DEFAULT_TIMEOUT = parseInt(process.env.REACT_APP_REQUEST_TIMEOUT || '30000', 10);

// Backend error response structure
interface BackendErrorResponse {
  error: {
    code: string;
    message: string;
    details?: Record<string, string>;
  };
  request_id?: string;
  timestamp?: string;
}

// Custom API error class with detailed information
export class ApiError extends Error {
  code: string;
  status: number;
  details?: Record<string, string>;
  requestId?: string;

  constructor(
    message: string,
    code: string = 'UNKNOWN_ERROR',
    status: number = 500,
    details?: Record<string, string>,
    requestId?: string
  ) {
    super(message);
    this.name = 'ApiError';
    this.code = code;
    this.status = status;
    this.details = details;
    this.requestId = requestId;
  }

  // Get a user-friendly error message including field details
  getDisplayMessage(): string {
    if (this.details && Object.keys(this.details).length > 0) {
      const fieldErrors = Object.entries(this.details)
        .map(([, msg]) => `${msg}`)
        .join('. ');
      return fieldErrors;
    }
    return this.message;
  }
}

// Parse error response from backend
export async function parseErrorResponse(response: Response): Promise<ApiError> {
  try {
    const data: BackendErrorResponse = await response.json();
    if (data.error) {
      return new ApiError(
        data.error.message,
        data.error.code,
        response.status,
        data.error.details,
        data.request_id
      );
    }
  } catch {
    // JSON parsing failed, fall back to status text
  }
  
  return new ApiError(
    response.statusText || 'An error occurred',
    'UNKNOWN_ERROR',
    response.status
  );
}

// Centralized fetch wrapper that handles token expiration and timeouts
export async function apiFetch(
  url: string,
  options: RequestInit = {},
  timeout: number = DEFAULT_TIMEOUT
): Promise<Response> {
  // Create AbortController for timeout
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), timeout);

  try {
    const response = await fetch(url, {
      ...options,
      signal: controller.signal,
    });

    clearTimeout(timeoutId);

    // Check if token has expired (401 Unauthorized)
    if (response.status === 401) {
      // Token is invalid or expired - logout and redirect
      localStorage.removeItem('jwt_token');
      window.location.href = '/login';
      throw new Error('Session expired. Please login again.');
    }

    return response;
  } catch (error) {
    clearTimeout(timeoutId);
    
    // Handle timeout error
    if (error instanceof Error && error.name === 'AbortError') {
      throw new Error(`Request timeout after ${timeout / 1000} seconds. Please check your connection.`);
    }
    
    throw error;
  }
}

// Helper to create authenticated headers
export function getAuthHeaders(token?: string): HeadersInit {
  const authToken = token || getToken();
  return {
    'Authorization': `Bearer ${authToken}`,
    'Content-Type': 'application/json',
  };
}
