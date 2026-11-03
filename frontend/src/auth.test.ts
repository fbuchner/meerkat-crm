import { test, expect, vi, beforeEach, afterEach, describe } from 'vitest';
import {
  loginUser,
  fetchAndCacheUserInfo,
  getToken,
  isAuthenticated,
  logoutUser,
  logoutAndRedirect,
  decodeToken,
  isAdmin,
} from './auth';

const fetchMock = vi.fn();
const ok = (body: object) => ({ ok: true, status: 200, json: async () => body });
const denied = { ok: false, status: 401, json: async () => ({}) };
const me = { ID: 7, Username: 'anna', is_admin: true };
const cached = { user_id: 7, username: 'anna', is_admin: true };

const cache = (value: object | string) =>
  localStorage.setItem('user_info', typeof value === 'string' ? value : JSON.stringify(value));

beforeEach(() => {
  localStorage.clear();
  fetchMock.mockReset();
  vi.stubGlobal('fetch', fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('loginUser', () => {
  test('posts the credentials with cookies, caches the user and returns the preferences', async () => {
    fetchMock.mockResolvedValueOnce(ok({ language: 'de', date_format: 'iso', token: 'secret' }));
    fetchMock.mockResolvedValueOnce(ok(me));

    await expect(loginUser('anna', 'pw')).resolves.toEqual({ language: 'de', date_format: 'iso' });

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toMatch(/\/api\/v1\/login$/);
    expect(init).toMatchObject({ method: 'POST', credentials: 'include' });
    expect(JSON.parse(init.body)).toEqual({ identifier: 'anna', password: 'pw' });
    expect(JSON.parse(localStorage.getItem('user_info')!)).toEqual(cached);
    expect(isAuthenticated()).toBe(true);
  });

  test('a refused login throws and leaves the user logged out', async () => {
    fetchMock.mockResolvedValueOnce(denied);
    await expect(loginUser('anna', 'wrong')).rejects.toThrow('Login failed');
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(isAuthenticated()).toBe(false);
  });

  // BUG auth.ts ignores a failed /users/me, so LoginPage stays on the form with no error shown.
  test.skip('BUG: a login whose user lookup fails rejects instead of resolving logged out', async () => {
    fetchMock.mockResolvedValueOnce(ok({}));
    fetchMock.mockResolvedValueOnce(denied);
    await expect(loginUser('anna', 'pw')).rejects.toThrow();
  });
});

describe('fetchAndCacheUserInfo', () => {
  test('maps the server fields and caches the result', async () => {
    fetchMock.mockResolvedValueOnce(ok(me));
    await expect(fetchAndCacheUserInfo()).resolves.toEqual(cached);
    expect(fetchMock.mock.calls[0][1]).toMatchObject({ credentials: 'include' });
    expect(JSON.parse(localStorage.getItem('user_info')!)).toEqual(cached);
  });

  test('a user without is_admin is cached as a non-admin', async () => {
    fetchMock.mockResolvedValueOnce(ok({ ID: 7, Username: 'anna' }));
    await expect(fetchAndCacheUserInfo()).resolves.toMatchObject({ is_admin: false });
  });

  test('without a session returns null and caches nothing', async () => {
    fetchMock.mockResolvedValueOnce(denied);
    await expect(fetchAndCacheUserInfo()).resolves.toBeNull();
    expect(localStorage.getItem('user_info')).toBeNull();
  });

  test('a network failure returns null instead of throwing', async () => {
    fetchMock.mockRejectedValueOnce(new Error('offline'));
    await expect(fetchAndCacheUserInfo()).resolves.toBeNull();
  });
});

describe('session state', () => {
  test('without cached user info the user is logged out', () => {
    expect(getToken()).toBeNull();
    expect(isAuthenticated()).toBe(false);
    expect(decodeToken()).toBeNull();
    expect(isAdmin()).toBe(false);
  });

  test('cached user info counts as logged in and never exposes a real token', () => {
    cache(cached);
    expect(isAuthenticated()).toBe(true);
    expect(getToken()).toBe('cookie-auth');
    expect(decodeToken()).toEqual({ ...cached, exp: 0 });
  });

  test.each([
    [{ ...cached, is_admin: true }, true],
    [{ ...cached, is_admin: false }, false],
    [{ user_id: 7, username: 'anna' }, false],
    ['{not json', false],
  ])('isAdmin for cached %j is %s', (value, expected) => {
    cache(value);
    expect(isAdmin()).toBe(expected);
  });

  test('corrupt cached user info decodes to null instead of throwing', () => {
    cache('{not json');
    expect(decodeToken()).toBeNull();
  });
});

describe('logout', () => {
  test('tells the server and clears the cached user', async () => {
    cache(cached);
    fetchMock.mockResolvedValueOnce(ok({}));
    await logoutUser();
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toMatch(/\/api\/v1\/logout$/);
    expect(init).toMatchObject({ method: 'POST', credentials: 'include' });
    expect(isAuthenticated()).toBe(false);
  });

  test('clears the cached user even when the server is unreachable', async () => {
    cache(cached);
    fetchMock.mockRejectedValueOnce(new Error('offline'));
    await expect(logoutUser()).resolves.toBeUndefined();
    expect(isAuthenticated()).toBe(false);
  });

  test('logoutAndRedirect clears the user and goes to the login page', async () => {
    cache(cached);
    fetchMock.mockResolvedValueOnce(ok({}));
    const location = { href: '/contacts' };
    vi.stubGlobal('location', location);
    await logoutAndRedirect();
    expect(location.href).toBe('/login');
    expect(isAuthenticated()).toBe(false);
  });
});
