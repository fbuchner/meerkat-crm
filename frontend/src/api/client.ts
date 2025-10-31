// API client with authentication and error handling
import { getToken } from '../auth';

// API base URL with versioning
const API_SERVER_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';
export const API_BASE_URL = `${API_SERVER_URL}/api/v1`;

// Default request timeout in milliseconds (30 seconds)
const DEFAULT_TIMEOUT = parseInt(process.env.REACT_APP_REQUEST_TIMEOUT || '30000', 10);

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
