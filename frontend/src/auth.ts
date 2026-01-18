// src/auth.ts
// Simple JWT auth service for meerkat crm frontend

const API_SERVER_URL = process.env.REACT_APP_API_URL || '';
export const API_BASE_URL = `${API_SERVER_URL}/api/v1`;

export interface LoginResponse {
  token: string;
  language?: string;
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
