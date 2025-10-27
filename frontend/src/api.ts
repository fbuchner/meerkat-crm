// src/api.ts
// Basic API service for Perema backend (Legacy - prefer using api/client.ts)

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

export async function fetchContacts(token: string) {
  const response = await apiFetch(`${API_BASE_URL}/contacts`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
  });
  if (!response.ok) {
    throw new Error('Failed to fetch contacts');
  }
  return response.json();
}

// Add more API functions as needed, e.g. for login, notes, activities, reminders
