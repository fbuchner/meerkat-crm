import { describe, test, expect } from 'vitest';
import { ApiError, parseErrorResponse } from './client';

describe('ApiError.getDisplayMessage', () => {
  test('returns the message when there are no details', () => {
    const err = new ApiError('Something went wrong', 'INTERNAL', 500);
    expect(err.getDisplayMessage()).toBe('Something went wrong');
  });

  test('joins field-level details when present', () => {
    const err = new ApiError('Validation failed', 'VALIDATION', 400, {
      email: 'Email is invalid',
      password: 'Password too weak',
    });
    expect(err.getDisplayMessage()).toBe('Email is invalid. Password too weak');
  });

  test('returns the message for an empty details object', () => {
    const err = new ApiError('Validation failed', 'VALIDATION', 400, {});
    expect(err.getDisplayMessage()).toBe('Validation failed');
  });
});

describe('parseErrorResponse', () => {
  test('parses a structured backend error', async () => {
    const response = new Response(
      JSON.stringify({
        error: { code: 'NOT_FOUND', message: 'Contact not found', details: { id: '42' } },
        request_id: 'req-123',
      }),
      { status: 404 }
    );

    const err = await parseErrorResponse(response);
    expect(err).toBeInstanceOf(ApiError);
    expect(err.message).toBe('Contact not found');
    expect(err.code).toBe('NOT_FOUND');
    expect(err.status).toBe(404);
    expect(err.details).toEqual({ id: '42' });
    expect(err.requestId).toBe('req-123');
  });

  test('falls back to status text for a non-JSON body', async () => {
    const response = new Response('<html>Bad Gateway</html>', {
      status: 502,
      statusText: 'Bad Gateway',
    });

    const err = await parseErrorResponse(response);
    expect(err.message).toBe('Bad Gateway');
    expect(err.code).toBe('UNKNOWN_ERROR');
    expect(err.status).toBe(502);
  });

  test('falls back to a generic message when status text is empty', async () => {
    const response = new Response('not json', { status: 500 });

    const err = await parseErrorResponse(response);
    expect(err.message).toBe('An error occurred');
    expect(err.status).toBe(500);
  });
});
