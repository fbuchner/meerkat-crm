import { test, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useContactDialogs } from './useContactDialogs';
import * as notesApi from '../api/notes';
import * as activitiesApi from '../api/activities';

vi.mock('../api/notes');
vi.mock('../api/activities');

const notes = vi.mocked(notesApi);
const activities = vi.mocked(activitiesApi);
const activity = { title: 'Lunch', description: 'Pizza', location: 'Berlin', date: '2026-04-01', contact_ids: [1, 2] };

const quietLogs = () => vi.spyOn(console, 'error').mockImplementation(() => {});

function setup() {
  const onRefresh = vi.fn().mockResolvedValue(undefined);
  const showError = vi.fn();
  const { result } = renderHook(() => useContactDialogs('12', onRefresh, { showError }));
  return { result, onRefresh, showError };
}

beforeEach(() => {
  vi.resetAllMocks();
});

afterEach(() => {
  vi.restoreAllMocks();
});

test('dialog flags start closed and toggle independently', () => {
  const { result } = setup();
  expect(result.current.noteDialogOpen).toBe(false);
  expect(result.current.activityDialogOpen).toBe(false);
  act(() => result.current.setNoteDialogOpen(true));
  expect(result.current.noteDialogOpen).toBe(true);
  expect(result.current.activityDialogOpen).toBe(false);
  act(() => result.current.setActivityDialogOpen(true));
  expect(result.current.activityDialogOpen).toBe(true);
});

test('saving a note converts the date and numeric contact id, then refreshes', async () => {
  const { result, onRefresh } = setup();
  await act(() => result.current.handleSaveNote('Hello', '2026-03-15'));
  expect(notes.createNote).toHaveBeenCalledWith('12', {
    content: 'Hello',
    date: '2026-03-15T00:00:00.000Z',
    contact_id: 12,
  });
  expect(onRefresh).toHaveBeenCalledTimes(1);
});

test('without a contact id a note is not saved', async () => {
  const onRefresh = vi.fn().mockResolvedValue(undefined);
  const { result } = renderHook(() => useContactDialogs(undefined, onRefresh));
  await act(() => result.current.handleSaveNote('Hello', '2026-03-15'));
  expect(notes.createNote).not.toHaveBeenCalled();
  expect(onRefresh).not.toHaveBeenCalled();
});

test('a failed note save notifies and rethrows a displayable error', async () => {
  quietLogs();
  notes.createNote.mockRejectedValue(new Error('boom'));
  const { result, onRefresh, showError } = setup();
  await act(async () => {
    await expect(result.current.handleSaveNote('Hello', '2026-03-15')).rejects.toThrow('boom');
  });
  expect(showError).toHaveBeenCalledWith('boom');
  expect(onRefresh).not.toHaveBeenCalled();
});

test('an unparseable note date is reported and never reaches the API', async () => {
  quietLogs();
  const { result, showError } = setup();
  await act(async () => {
    await expect(result.current.handleSaveNote('Hello', 'not-a-date')).rejects.toThrow();
  });
  expect(notes.createNote).not.toHaveBeenCalled();
  expect(showError).toHaveBeenCalledTimes(1);
});

test('saving an activity converts the date and refreshes', async () => {
  const { result, onRefresh } = setup();
  await act(() => result.current.handleSaveActivity(activity));
  expect(activities.createActivity).toHaveBeenCalledWith({ ...activity, date: '2026-04-01T00:00:00.000Z' });
  expect(onRefresh).toHaveBeenCalledTimes(1);
});

test('a failed activity save notifies and rethrows a displayable error', async () => {
  quietLogs();
  activities.createActivity.mockRejectedValue(new Error('boom'));
  const { result, onRefresh, showError } = setup();
  await act(async () => {
    await expect(result.current.handleSaveActivity(activity)).rejects.toThrow('boom');
  });
  expect(showError).toHaveBeenCalledWith('boom');
  expect(onRefresh).not.toHaveBeenCalled();
});
