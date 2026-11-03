import { test, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useReminderManagement } from './useReminderManagement';
import * as api from '../api/reminders';
import type { Reminder, ReminderFormData } from '../api/reminders';

vi.mock('../api/reminders');

const mocked = vi.mocked(api);
const reminder: Reminder = {
  ID: 3,
  message: 'Call',
  by_mail: true,
  remind_at: '2026-06-15T00:00:00Z',
  recurrence: 'once',
  reoccur_from_completion: false,
  completed: false,
  email_sent: false,
  contact_id: 1,
};
const data: ReminderFormData = {
  message: 'Call',
  by_mail: true,
  remind_at: '2026-06-15T00:00:00.000Z',
  recurrence: 'once',
  reoccur_from_completion: false,
  contact_id: 1,
};

// Failure paths log through console.error by design; keep the test output readable.
const quietLogs = () => vi.spyOn(console, 'error').mockImplementation(() => {});

beforeEach(() => {
  vi.resetAllMocks();
  mocked.getRemindersForContact.mockResolvedValue([reminder]);
});

afterEach(() => {
  vi.restoreAllMocks();
});

test('refresh loads the reminders of the numeric contact id', async () => {
  const { result } = renderHook(() => useReminderManagement('12'));
  await act(() => result.current.refreshReminders());
  expect(mocked.getRemindersForContact).toHaveBeenCalledWith(12);
  expect(result.current.reminders).toEqual([reminder]);
  expect(result.current.error).toBeNull();
});

test('a failed refresh exposes the message, and the next success clears it', async () => {
  quietLogs();
  mocked.getRemindersForContact.mockRejectedValueOnce(new Error('offline'));
  const { result } = renderHook(() => useReminderManagement('12'));
  await act(() => result.current.refreshReminders());
  expect(result.current.error).toBe('offline');
  await act(() => result.current.refreshReminders());
  expect(result.current.error).toBeNull();
  expect(result.current.reminders).toEqual([reminder]);
});

test('without a contact id nothing is fetched or created', async () => {
  const { result } = renderHook(() => useReminderManagement(undefined));
  await act(async () => {
    await result.current.refreshReminders();
    await result.current.handleSaveReminder(data);
  });
  expect(mocked.getRemindersForContact).not.toHaveBeenCalled();
  expect(mocked.createReminder).not.toHaveBeenCalled();
  expect(mocked.updateReminder).not.toHaveBeenCalled();
});

test('save creates when not editing, refreshes and closes the dialog', async () => {
  const { result } = renderHook(() => useReminderManagement('12'));
  act(() => result.current.handleAddReminder());
  expect(result.current.reminderDialogOpen).toBe(true);
  await act(() => result.current.handleSaveReminder(data));
  expect(mocked.createReminder).toHaveBeenCalledWith(12, data);
  expect(mocked.updateReminder).not.toHaveBeenCalled();
  expect(result.current.reminders).toEqual([reminder]);
  expect(result.current.reminderDialogOpen).toBe(false);
});

test('save updates the reminder being edited, then leaves edit mode', async () => {
  const { result } = renderHook(() => useReminderManagement('12'));
  act(() => result.current.handleEditReminder(reminder));
  expect(result.current.reminderDialogOpen).toBe(true);
  expect(result.current.editingReminder).toEqual(reminder);
  await act(() => result.current.handleSaveReminder(data));
  expect(mocked.updateReminder).toHaveBeenCalledWith(3, data);
  expect(mocked.createReminder).not.toHaveBeenCalled();
  expect(result.current.editingReminder).toBeNull();
  expect(result.current.reminderDialogOpen).toBe(false);
});

test('add discards a reminder that was being edited', () => {
  const { result } = renderHook(() => useReminderManagement('12'));
  act(() => result.current.handleEditReminder(reminder));
  act(() => result.current.handleAddReminder());
  expect(result.current.editingReminder).toBeNull();
  expect(result.current.reminderDialogOpen).toBe(true);
});

test('a failed save notifies, rethrows and keeps the dialog and edit state', async () => {
  quietLogs();
  const showError = vi.fn();
  mocked.updateReminder.mockRejectedValue(new Error('boom'));
  const { result } = renderHook(() => useReminderManagement('12', { showError }));
  act(() => result.current.handleEditReminder(reminder));
  await act(async () => {
    await expect(result.current.handleSaveReminder(data)).rejects.toThrow('boom');
  });
  expect(showError).toHaveBeenCalledWith('boom');
  expect(result.current.reminderDialogOpen).toBe(true);
  expect(result.current.editingReminder).toEqual(reminder);
  expect(mocked.getRemindersForContact).not.toHaveBeenCalled();
});

test('complete and delete call the API and refresh', async () => {
  const { result } = renderHook(() => useReminderManagement('12'));
  await act(() => result.current.handleCompleteReminder(3));
  await act(() => result.current.handleDeleteReminder(4));
  expect(mocked.completeReminder).toHaveBeenCalledWith(3);
  expect(mocked.deleteReminder).toHaveBeenCalledWith(4);
  expect(mocked.getRemindersForContact).toHaveBeenCalledTimes(2);
});

test('failed complete and delete notify, rethrow and skip the refresh', async () => {
  quietLogs();
  const showError = vi.fn();
  mocked.completeReminder.mockRejectedValue(new Error('c'));
  mocked.deleteReminder.mockRejectedValue(new Error('d'));
  const { result } = renderHook(() => useReminderManagement('12', { showError }));
  await act(async () => {
    await expect(result.current.handleCompleteReminder(3)).rejects.toThrow('c');
    await expect(result.current.handleDeleteReminder(3)).rejects.toThrow('d');
  });
  expect(showError.mock.calls).toEqual([['c'], ['d']]);
  expect(mocked.getRemindersForContact).not.toHaveBeenCalled();
});
