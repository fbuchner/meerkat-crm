// src/auth.ts
// Simple JWT auth service for Perema frontend

export const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

export async function loginUser(email: string, password: string): Promise<string> {
  const response = await fetch(`${API_BASE_URL}/login`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ email, password }),
  });
  if (!response.ok) {
    throw new Error('Login failed');
  }
  const data = await response.json();
  return data.token;
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
