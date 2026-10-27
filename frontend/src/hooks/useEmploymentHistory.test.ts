import { test, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useEmploymentHistory } from './useEmploymentHistory';
import * as api from '../api/employmentHistory';

vi.mock('../api/employmentHistory');
vi.mock('../utils/errorHandler', () => ({
  handleError: vi.fn(),
  handleFetchError: vi.fn(),
}));

const mocked = vi.mocked(api);
const entry = { ID: 5, CreatedAt: '', UpdatedAt: '', contact_id: 1, organization: 'Acme' };

beforeEach(() => {
  vi.resetAllMocks();
  mocked.getEmploymentHistory.mockResolvedValue({ employment_history: [entry] });
});

test('refresh loads entries', async () => {
  const { result } = renderHook(() => useEmploymentHistory('1'));
  await act(() => result.current.refreshEmploymentHistory());
  expect(mocked.getEmploymentHistory).toHaveBeenCalledWith('1');
  expect(result.current.entries).toEqual([entry]);
  expect(result.current.loading).toBe(false);
});

test('without a contact id nothing is fetched or written', async () => {
  const { result } = renderHook(() => useEmploymentHistory(undefined));
  await act(async () => {
    await result.current.refreshEmploymentHistory();
    await result.current.handleSaveEntry({ organization: 'X' });
    await result.current.handleDeleteEntry(1);
    await result.current.handleEndEntry(1);
  });
  expect(mocked.getEmploymentHistory).not.toHaveBeenCalled();
  expect(mocked.createEmploymentHistory).not.toHaveBeenCalled();
  expect(mocked.deleteEmploymentHistory).not.toHaveBeenCalled();
  expect(mocked.endEmploymentHistory).not.toHaveBeenCalled();
});

test('save creates when not editing, then refreshes, notifies contact and closes dialog', async () => {
  const onContactChanged = vi.fn();
  const { result } = renderHook(() => useEmploymentHistory('1', undefined, onContactChanged));
  act(() => result.current.handleAddEntry());
  expect(result.current.dialogOpen).toBe(true);
  await act(() => result.current.handleSaveEntry({ organization: 'Acme' }));
  expect(mocked.createEmploymentHistory).toHaveBeenCalledWith('1', { organization: 'Acme' });
  expect(mocked.updateEmploymentHistory).not.toHaveBeenCalled();
  expect(result.current.entries).toEqual([entry]);
  expect(onContactChanged).toHaveBeenCalledTimes(1);
  expect(result.current.dialogOpen).toBe(false);
  expect(result.current.editingEntry).toBeNull();
});

test('save updates when editing an entry', async () => {
  const { result } = renderHook(() => useEmploymentHistory('1'));
  act(() => result.current.handleEditEntry(entry));
  expect(result.current.dialogOpen).toBe(true);
  expect(result.current.editingEntry).toEqual(entry);
  await act(() => result.current.handleSaveEntry({ organization: 'New' }));
  expect(mocked.updateEmploymentHistory).toHaveBeenCalledWith('1', 5, { organization: 'New' });
  expect(mocked.createEmploymentHistory).not.toHaveBeenCalled();
});

test('add clears any entry being edited', () => {
  const { result } = renderHook(() => useEmploymentHistory('1'));
  act(() => result.current.handleEditEntry(entry));
  act(() => result.current.handleAddEntry());
  expect(result.current.editingEntry).toBeNull();
});

test('delete and end call the API, refresh and notify contact', async () => {
  const onContactChanged = vi.fn();
  const { result } = renderHook(() => useEmploymentHistory('1', undefined, onContactChanged));
  await act(() => result.current.handleDeleteEntry(5));
  await act(() => result.current.handleEndEntry(5));
  expect(mocked.deleteEmploymentHistory).toHaveBeenCalledWith('1', 5);
  expect(mocked.endEmploymentHistory).toHaveBeenCalledWith('1', 5);
  expect(mocked.getEmploymentHistory).toHaveBeenCalledTimes(2);
  expect(onContactChanged).toHaveBeenCalledTimes(2);
});

test('failed save rethrows, keeps dialog open and skips contact refresh', async () => {
  const onContactChanged = vi.fn();
  mocked.createEmploymentHistory.mockRejectedValue(new Error('boom'));
  const { result } = renderHook(() => useEmploymentHistory('1', undefined, onContactChanged));
  act(() => result.current.handleAddEntry());
  await act(async () => {
    await expect(result.current.handleSaveEntry({ organization: 'Acme' })).rejects.toThrow('boom');
  });
  expect(result.current.dialogOpen).toBe(true);
  expect(onContactChanged).not.toHaveBeenCalled();
});

test('failed delete and end are swallowed after notifying', async () => {
  mocked.deleteEmploymentHistory.mockRejectedValue(new Error('d'));
  mocked.endEmploymentHistory.mockRejectedValue(new Error('e'));
  const { result } = renderHook(() => useEmploymentHistory('1'));
  await act(async () => {
    await expect(result.current.handleDeleteEntry(5)).resolves.toBeUndefined();
    await expect(result.current.handleEndEntry(5)).resolves.toBeUndefined();
  });
});
