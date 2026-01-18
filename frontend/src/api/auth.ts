import { apiFetch, API_BASE_URL, getAuthHeaders } from './client';

type ErrorDetails = Record<string, unknown> | undefined | null;

function extractDetailMessage(details: ErrorDetails): string | null {
  if (!details || typeof details !== 'object') {
    return null;
  }

  const normalized = details as Record<string, unknown>;

  const reason = normalized.reason;
  if (typeof reason === 'string' && reason.trim().length > 0) {
    return reason.trim();
  }

  const messages: string[] = [];

  Object.values(normalized).forEach(value => {
    if (typeof value === 'string' && value.trim().length > 0) {
      messages.push(value.trim());
      return;
    }

    if (Array.isArray(value)) {
      value.forEach(item => {
        if (typeof item === 'string' && item.trim().length > 0) {
          messages.push(item.trim());
        }
      });
    }
  });

  if (messages.length === 0) {
    return null;
  }

  return messages.join(' ');
}

async function handleResponse(response: Response, fallback: string): Promise<Record<string, any>> {
  const raw = await response.text();
  let data: any = null;

  if (raw) {
    try {
      data = JSON.parse(raw);
    } catch (error) {
      data = raw;
    }
  }

  if (response.ok) {
    if (!data) {
      return {};
    }

    return typeof data === 'object' ? data : { message: data };
  }

  let message = fallback;

  if (data && typeof data === 'object') {
    const errorDetail = (data as { error?: Record<string, unknown> | string }).error;
    if (errorDetail && typeof errorDetail === 'object') {
      const details = (errorDetail as { details?: ErrorDetails }).details;
      const specificMessage = extractDetailMessage(details);
      if (specificMessage) {
        message = specificMessage;
      } else {
        const detailMessage = (errorDetail as { message?: unknown }).message;
        if (typeof detailMessage === 'string' && detailMessage.trim().length > 0) {
          message = detailMessage.trim();
        }
      }
    } else if (typeof errorDetail === 'string' && errorDetail.trim().length > 0) {
      // Handle simple string error format: {"error": "message"}
      message = errorDetail.trim();
    } else if (typeof data.message === 'string' && data.message.trim().length > 0) {
      message = data.message.trim();
    }
  } else if (typeof data === 'string' && data.trim().length > 0) {
    message = data.trim();
  }

  throw new Error(message);
}

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

export async function changePassword(currentPassword: string, newPassword: string, token?: string | null): Promise<string> {
  const response = await apiFetch(`${API_BASE_URL}/users/change-password`, {
    method: 'POST',
    headers: getAuthHeaders(token || undefined),
    body: JSON.stringify({
      current_password: currentPassword,
      new_password: newPassword,
    }),
  });

  const data = await handleResponse(response, 'Unable to change password.');
  return data?.message || 'Password updated successfully.';
}

export async function updateLanguage(language: string, token?: string | null): Promise<string> {
  const response = await apiFetch(`${API_BASE_URL}/users/language`, {
    method: 'PATCH',
    headers: getAuthHeaders(token || undefined),
    body: JSON.stringify({ language }),
  });

  const data = await handleResponse(response, 'Unable to update language.');
  return data?.message || 'Language updated successfully.';
}
