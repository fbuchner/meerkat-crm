// src/auth.ts
// Simple JWT auth service for meerkat crm frontend

const API_SERVER_URL = process.env.REACT_APP_API_URL || '';
export const API_BASE_URL = `${API_SERVER_URL}/api/v1`;

export interface LoginResponse {
  token: string;
  language?: string;
  date_format?: string;
}

export async function loginUser(identifier: string, password: string): Promise<LoginResponse> {
  const response = await fetch(`${API_BASE_URL}/login`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ identifier, password }),
  });
  if (!response.ok) {
    throw new Error('Login failed');
  }
  const data = await response.json();
  return {
    token: data.token,
    language: data.language,
    date_format: data.date_format,
  };
}

export function saveToken(token: string) {
  localStorage.setItem('jwt_token', token);
}

export function getToken(): string | null {
  return localStorage.getItem('jwt_token');
}

export function logoutUser() {
  localStorage.removeItem('jwt_token');
}

export function logoutAndRedirect() {
  localStorage.removeItem('jwt_token');
  window.location.href = '/login';
}

interface DecodedToken {
  user_id: number;
  username: string;
  is_admin: boolean;
  exp: number;
}

export function decodeToken(): DecodedToken | null {
  const token = getToken();
  if (!token) {
    return null;
  }

  try {
    const parts = token.split('.');
    if (parts.length !== 3) {
      return null;
    }

    const payload = parts[1];
    const decoded = JSON.parse(atob(payload));

    return {
      user_id: decoded.user_id,
      username: decoded.username,
      is_admin: decoded.is_admin || false,
      exp: decoded.exp,
    };
  } catch {
    return null;
  }
}

// Check if the current user is an admin based on JWT token
export function isAdmin(): boolean {
  const decoded = decodeToken();
  return decoded?.is_admin || false;
}
