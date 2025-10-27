// API client with authentication and error handling
import { getToken } from '../auth';

// API base URL with versioning
const API_SERVER_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';
export const API_BASE_URL = `${API_SERVER_URL}/api/v1`;

// Centralized fetch wrapper that handles token expiration
export async function apiFetch(
  url: string,
  options: RequestInit = {}
): Promise<Response> {
  const response = await fetch(url, options);

  // Check if token has expired (401 Unauthorized)
  if (response.status === 401) {
    // Token is invalid or expired - logout and redirect
    localStorage.removeItem('jwt_token');
    window.location.href = '/login';
    throw new Error('Session expired. Please login again.');
  }

  return response;
}

// Helper to create authenticated headers
export function getAuthHeaders(token?: string): HeadersInit {
  const authToken = token || getToken();
  return {
    'Authorization': `Bearer ${authToken}`,
    'Content-Type': 'application/json',
  };
}
