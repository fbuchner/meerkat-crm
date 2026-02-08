import { apiFetch, API_BASE_URL, getAuthHeaders } from './client';
import { handleResponse } from './errorHandling';

export async function requestPasswordReset(email: string): Promise<string> {
  const response = await apiFetch(`${API_BASE_URL}/password-reset/request`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ email }),
  });

  const data = await handleResponse(response, 'Unable to request password reset.');
  return data?.message || 'If an account exists, password reset instructions were sent.';
}

export async function confirmPasswordReset(token: string, password: string): Promise<string> {
  const response = await apiFetch(`${API_BASE_URL}/password-reset/confirm`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ token, password }),
  });

  const data = await handleResponse(response, 'Unable to reset password.');
  return data?.message || 'Password reset successful.';
}

export async function changePassword(currentPassword: string, newPassword: string): Promise<string> {
  const response = await apiFetch(`${API_BASE_URL}/users/change-password`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify({
      current_password: currentPassword,
      new_password: newPassword,
    }),
  });

  const data = await handleResponse(response, 'Unable to change password.');
  return data?.message || 'Password updated successfully.';
}
