import { describe, test, expect } from 'vitest';
import { extractDetailMessage, handleResponse } from './errorHandling';

describe('extractDetailMessage', () => {
  test('returns null for null, undefined, and non-objects', () => {
    expect(extractDetailMessage(null)).toBeNull();
    expect(extractDetailMessage(undefined)).toBeNull();
  });

  test('prefers the reason field over other values', () => {
    expect(extractDetailMessage({ reason: 'Rate limit exceeded', email: 'Invalid email' })).toBe(
      'Rate limit exceeded'
    );
  });

  test('joins string values from the details object', () => {
    expect(extractDetailMessage({ email: 'Email already taken', username: 'Username too short' })).toBe(
      'Email already taken Username too short'
    );
  });

  test('collects strings from array values', () => {
    expect(extractDetailMessage({ fields: ['First error', 'Second error'] })).toBe(
      'First error Second error'
    );
  });

  test('ignores empty strings and non-string values', () => {
    expect(extractDetailMessage({ a: '', b: 42, c: { nested: true } })).toBeNull();
  });
});

describe('handleResponse', () => {
  const jsonResponse = (body: unknown, status = 200) =>
    new Response(JSON.stringify(body), { status });

  test('returns the parsed body for a successful JSON response', async () => {
    await expect(handleResponse(jsonResponse({ id: 1, name: 'Ada' }), 'fallback')).resolves.toEqual({
      id: 1,
      name: 'Ada',
    });
  });

  test('returns an empty object for a successful empty body', async () => {
    await expect(handleResponse(new Response('', { status: 200 }), 'fallback')).resolves.toEqual({});
  });

  test('wraps a successful non-object body in a message field', async () => {
    await expect(handleResponse(jsonResponse('done'), 'fallback')).resolves.toEqual({
      message: 'done',
    });
  });

  test('throws the field detail message from a structured backend error', async () => {
    const response = jsonResponse(
      { error: { code: 'VALIDATION', message: 'Validation failed', details: { email: 'Email already registered' } } },
      400
    );
    await expect(handleResponse(response, 'fallback')).rejects.toThrow('Email already registered');
  });

  test('falls back to the error message when there are no details', async () => {
    const response = jsonResponse({ error: { code: 'NOT_FOUND', message: 'Contact not found' } }, 404);
    await expect(handleResponse(response, 'fallback')).rejects.toThrow('Contact not found');
  });

  test('handles the simple string error format', async () => {
    await expect(handleResponse(jsonResponse({ error: 'Something broke' }, 500), 'fallback')).rejects.toThrow(
      'Something broke'
    );
  });

  test('uses a top-level message field when present', async () => {
    await expect(handleResponse(jsonResponse({ message: 'Too many requests' }, 429), 'fallback')).rejects.toThrow(
      'Too many requests'
    );
  });

  test('uses a plain-text error body as the message', async () => {
    await expect(handleResponse(new Response('Bad Gateway', { status: 502 }), 'fallback')).rejects.toThrow(
      'Bad Gateway'
    );
  });

  test('falls back to the provided message for an empty error body', async () => {
    await expect(handleResponse(new Response('', { status: 500 }), 'Request failed')).rejects.toThrow(
      'Request failed'
    );
  });
});
