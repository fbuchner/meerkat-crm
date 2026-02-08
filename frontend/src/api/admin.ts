// Admin API calls for user management
import { apiFetch, API_BASE_URL, getAuthHeaders, parseErrorResponse } from './client';
import type { User, UsersListResponse, UserUpdateInput } from '../types';

// Get current authenticated user's information
export async function getCurrentUser(): Promise<User> {
  const response = await apiFetch(`${API_BASE_URL}/users/me`, {
    method: 'GET',
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  return response.json();
}


// Get paginated list of all users (admin only)
export async function getUsers(
  page: number = 1,
  limit: number = 25
): Promise<UsersListResponse> {
  const params = new URLSearchParams({
    page: page.toString(),
    limit: limit.toString(),
  });

  const response = await apiFetch(`${API_BASE_URL}/admin/users?${params}`, {
    method: 'GET',
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  return response.json();
}


// Get a single user by ID (admin only)
export async function getUserById(id: number): Promise<User> {
  const response = await apiFetch(`${API_BASE_URL}/admin/users/${id}`, {
    method: 'GET',
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  return response.json();
}


// Update a user (admin only)
export async function updateUser(
  id: number,
  data: UserUpdateInput
): Promise<User> {
  const response = await apiFetch(`${API_BASE_URL}/admin/users/${id}`, {
    method: 'PATCH',
    headers: getAuthHeaders(),
    body: JSON.stringify(data),
  });

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  return response.json();
}

/**
 * Delete a user (admin only)
 */
export async function deleteUser(id: number): Promise<void> {
  const response = await apiFetch(`${API_BASE_URL}/admin/users/${id}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }
}
