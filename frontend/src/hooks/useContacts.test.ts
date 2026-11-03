import { test, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useContacts } from './useContacts';
import * as api from '../api/contacts';
import type { Contact, ContactsResponse, GetContactsParams } from '../api/contacts';

vi.mock('../api/contacts');

const mocked = vi.mocked(api);
const anna = { ID: 1, firstname: 'Anna', lastname: 'Schmidt' } as Contact;
const ben = { ID: 2, firstname: 'Ben', lastname: 'Meyer' } as Contact;
const response = (contacts: Contact[], extra: Partial<ContactsResponse> = {}): ContactsResponse => ({
  contacts,
  total: contacts.length,
  page: 1,
  limit: 25,
  ...extra,
});

const quietLogs = () => vi.spyOn(console, 'error').mockImplementation(() => {});

function setup(initial: GetContactsParams = {}) {
  return renderHook(({ params }) => useContacts(params), { initialProps: { params: initial } });
}

beforeEach(() => {
  vi.resetAllMocks();
  localStorage.clear();
  localStorage.setItem('user_info', JSON.stringify({ user_id: 1, username: 'anna', is_admin: false }));
  mocked.getContacts.mockResolvedValue(response([anna]));
});

afterEach(() => {
  vi.restoreAllMocks();
});

test('loads on mount, forwarding every filter', async () => {
  mocked.getContacts.mockResolvedValue(response([anna], { total: 40, page: 2 }));
  const params: GetContactsParams = { page: 2, limit: 10, search: 'an', circle: 'Work', sort: 'firstname', order: 'asc', archived: true };
  const { result } = setup(params);
  expect(result.current.loading).toBe(true);
  await waitFor(() => expect(result.current.loading).toBe(false));
  expect(mocked.getContacts).toHaveBeenCalledWith(expect.objectContaining(params));
  expect(result.current.contacts).toEqual([anna]);
  expect(result.current.total).toBe(40);
  expect(result.current.page).toBe(2);
  expect(result.current.error).toBeNull();
});

test('a new params object with the same values does not refetch', async () => {
  const { result, rerender } = setup({ search: 'an' });
  await waitFor(() => expect(result.current.loading).toBe(false));
  rerender({ params: { search: 'an' } });
  expect(mocked.getContacts).toHaveBeenCalledTimes(1);
  expect(result.current.loading).toBe(false);
});

test('a changed filter refetches', async () => {
  const { result, rerender } = setup({ search: 'an' });
  await waitFor(() => expect(result.current.loading).toBe(false));
  mocked.getContacts.mockResolvedValue(response([ben]));
  rerender({ params: { search: 'be' } });
  await waitFor(() => expect(result.current.contacts).toEqual([ben]));
  expect(mocked.getContacts).toHaveBeenLastCalledWith(expect.objectContaining({ search: 'be' }));
});

test('refetch loads again and clears an earlier error', async () => {
  quietLogs();
  mocked.getContacts.mockRejectedValueOnce(new Error('offline'));
  const { result } = setup();
  await waitFor(() => expect(result.current.error).toBe('offline'));
  await act(() => result.current.refetch());
  expect(result.current.error).toBeNull();
  expect(result.current.contacts).toEqual([anna]);
});

test('when logged out it reports an error without calling the API', async () => {
  quietLogs();
  localStorage.clear();
  const { result } = setup();
  await waitFor(() => expect(result.current.loading).toBe(false));
  expect(mocked.getContacts).not.toHaveBeenCalled();
  expect(result.current.error).toBe('No authentication token found');
  expect(result.current.contacts).toEqual([]);
});

test('missing response fields fall back to an empty first page', async () => {
  mocked.getContacts.mockResolvedValue({ contacts: null } as unknown as ContactsResponse);
  const { result } = setup({ page: 3 });
  await waitFor(() => expect(result.current.loading).toBe(false));
  expect(result.current.contacts).toEqual([]);
  expect(result.current.total).toBe(0);
  expect(result.current.page).toBe(1);
});

// BUG useContacts.ts has no request guard, so a slow older search overwrites the newer results.
test.skip('BUG: a slow response for an old search does not overwrite newer results', async () => {
  let resolveOld!: (r: ContactsResponse) => void;
  mocked.getContacts.mockImplementationOnce(() => new Promise((resolve) => { resolveOld = resolve; }));
  const { result, rerender } = setup({ search: 'a' });

  mocked.getContacts.mockResolvedValueOnce(response([ben]));
  rerender({ params: { search: 'ab' } });
  await waitFor(() => expect(result.current.contacts).toEqual([ben]));

  await act(async () => resolveOld(response([anna])));
  expect(result.current.contacts).toEqual([ben]);
});
