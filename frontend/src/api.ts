// src/api.ts
// Basic API service for Perema backend

export const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

export async function fetchContacts(token: string) {
  const response = await fetch(`${API_BASE_URL}/contacts`, {
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
