// Shared error handling utilities for API responses

export type ErrorDetails = Record<string, unknown> | undefined | null;

export function extractDetailMessage(details: ErrorDetails): string | null {
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

export async function handleResponse(response: Response, fallback: string): Promise<Record<string, any>> {
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
