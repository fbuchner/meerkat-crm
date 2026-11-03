import { test, expect, vi, beforeEach, afterEach } from 'vitest';
import { configure, render, screen, waitFor, cleanup, fireEvent, act } from '@testing-library/react';
import './i18n/config';
import App from './App';

vi.mock('./DashboardPage', () => ({ default: () => <div>dashboard page</div> }));

// Role queries over the whole shell are slow; a busy machine must not turn that into a failure.
vi.setConfig({ testTimeout: 30000 });
configure({ asyncUtilTimeout: 10000 });

const fetchMock = vi.fn();

// The server's answer to GET /users/me; everything else (OIDC config) is unavailable.
function serverSession(user: object | null) {
  fetchMock.mockImplementation(async (url: string) =>
    url.endsWith('/users/me') && user
      ? { ok: true, status: 200, json: async () => user }
      : { ok: false, status: 401, json: async () => ({}) }
  );
}

const cacheUser = (isAdmin = false) =>
  localStorage.setItem('user_info', JSON.stringify({ user_id: 1, username: 'anna', is_admin: isAdmin }));

const expectLoginForm = async () => {
  expect(await screen.findByRole('button', { name: 'Login' })).toBeInTheDocument();
  expect(screen.getByRole('textbox', { name: /Username or Email/ })).toBeInTheDocument();
  expect(screen.queryByRole('button', { name: 'Logout' })).not.toBeInTheDocument();
};

const expectAppShell = async () => {
  expect(await screen.findByRole('button', { name: 'Logout' })).toBeInTheDocument();
  expect(await screen.findByText('dashboard page')).toBeInTheDocument();
  expect(screen.getByRole('main')).toBeInTheDocument();
  expect(screen.queryByRole('button', { name: 'Login' })).not.toBeInTheDocument();
};

beforeEach(() => {
  localStorage.clear();
  window.history.pushState({}, '', '/');
  fetchMock.mockReset();
  vi.stubGlobal('fetch', fetchMock);
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

test('logged out shows the login form, not the app shell', async () => {
  serverSession(null);
  render(<App />);
  await expectLoginForm();
  await waitFor(() => expect(fetchMock).toHaveBeenCalledWith(expect.stringContaining('/users/me'), expect.anything()));
  expect(screen.queryByRole('main')).not.toBeInTheDocument();
});

test('logged out on a deep link still shows the login form', async () => {
  serverSession(null);
  window.history.pushState({}, '', '/contacts/42');
  render(<App />);
  await expectLoginForm();
});

test('logged in shows the app shell with navigation and the dashboard', async () => {
  serverSession(null);
  cacheUser();
  render(<App />);
  await expectAppShell();
  for (const name of ['Dashboard', 'Contacts', 'Activities', 'Notes', 'Network']) {
    expect(screen.getByRole('link', { name, hidden: true })).toBeInTheDocument();
  }
  expect(fetchMock).not.toHaveBeenCalled();
});

test('a server session without cached user info (OIDC redirect) restores the app shell', async () => {
  serverSession({ ID: 1, Username: 'anna', is_admin: false });
  render(<App />);
  await expectAppShell();
  expect(JSON.parse(localStorage.getItem('user_info')!)).toEqual({ user_id: 1, username: 'anna', is_admin: false });
});

test('logging out in another tab returns this tab to the login form', async () => {
  serverSession(null);
  cacheUser();
  render(<App />);
  await expectAppShell();
  localStorage.removeItem('user_info');
  act(() => {
    window.dispatchEvent(new StorageEvent('storage', { key: 'user_info' }));
  });
  await expectLoginForm();
});

test.each([
  [true, 'shows'],
  [false, 'hides'],
])('settings menu for is_admin=%s %s the Users link', async (isAdmin) => {
  serverSession(null);
  cacheUser(isAdmin);
  render(<App />);
  await expectAppShell();
  fireEvent.click(screen.getByRole('button', { name: 'Settings', hidden: true }));
  expect(await screen.findByRole('link', { name: 'Profile', hidden: true })).toBeInTheDocument();
  expect(screen.queryAllByRole('link', { name: 'Users', hidden: true })).toHaveLength(isAdmin ? 1 : 0);
});
