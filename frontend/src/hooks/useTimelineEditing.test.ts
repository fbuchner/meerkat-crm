import { test, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useTimelineEditing } from './useTimelineEditing';
import * as contactsApi from '../api/contacts';
import * as notesApi from '../api/notes';
import * as activitiesApi from '../api/activities';
import type { Note } from '../api/notes';
import type { Activity } from '../api/activities';

vi.mock('../api/contacts');
vi.mock('../api/notes');
vi.mock('../api/activities');

const contacts = vi.mocked(contactsApi);
const notes = vi.mocked(notesApi);
const activities = vi.mocked(activitiesApi);

const note: Note = { ID: 1, content: 'Hello', date: '2026-03-15T00:00:00Z', CreatedAt: '', UpdatedAt: '' };
const person = { ID: 9, firstname: 'Anna', lastname: 'Schmidt' };
const activity: Activity = {
  ID: 2,
  title: 'Lunch',
  description: 'Pizza',
  location: 'Berlin',
  date: '2026-04-01T00:00:00Z',
  CreatedAt: '',
  UpdatedAt: '',
  contacts: [person],
};
const directory = { contacts: [person], total: 1, page: 1, limit: 1000 } as contactsApi.ContactsResponse;

const quietLogs = () => vi.spyOn(console, 'error').mockImplementation(() => {});

function setup() {
  const onRefresh = vi.fn().mockResolvedValue(undefined);
  const showError = vi.fn();
  const { result } = renderHook(() => useTimelineEditing(5, onRefresh, { showError }));
  return { result, onRefresh, showError };
}

beforeEach(() => {
  vi.resetAllMocks();
  contacts.getContacts.mockResolvedValue(directory);
});

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
});

test('editing a note fills the form with its content and calendar date', async () => {
  const { result } = setup();
  await act(() => result.current.handleStartEditTimelineItem('note', note));
  expect(result.current.editingTimelineItem).toEqual({ type: 'note', id: 1 });
  expect(result.current.editTimelineValues).toEqual({ noteContent: 'Hello', noteDate: '2026-03-15' });
  expect(contacts.getContacts).not.toHaveBeenCalled();
});

test('editing a note without a date leaves the date field empty', async () => {
  const { result } = setup();
  await act(() => result.current.handleStartEditTimelineItem('note', { ...note, date: '' }));
  expect(result.current.editTimelineValues.noteDate).toBe('');
});

test('editing an activity fills the form and loads the contact picker only once', async () => {
  const { result } = setup();
  await act(() => result.current.handleStartEditTimelineItem('activity', activity));
  expect(result.current.editingTimelineItem).toEqual({ type: 'activity', id: 2 });
  expect(result.current.editTimelineValues).toEqual({
    activityTitle: 'Lunch',
    activityDescription: 'Pizza',
    activityLocation: 'Berlin',
    activityDate: '2026-04-01',
    activityContacts: [person],
  });
  expect(result.current.allContacts).toEqual([person]);
  await act(() => result.current.handleStartEditTimelineItem('activity', activity));
  expect(contacts.getContacts).toHaveBeenCalledTimes(1);
});

test('the activity form still opens when the contact picker fails to load', async () => {
  quietLogs();
  contacts.getContacts.mockRejectedValue(new Error('offline'));
  const { result } = setup();
  await act(() => result.current.handleStartEditTimelineItem('activity', activity));
  expect(result.current.editTimelineValues.activityTitle).toBe('Lunch');
  expect(result.current.allContacts).toEqual([]);
});

test('text typed while the contact picker loads is not overwritten', async () => {
  let resolve!: (r: contactsApi.ContactsResponse) => void;
  contacts.getContacts.mockImplementation(() => new Promise((r) => { resolve = r; }));
  const { result } = setup();

  await act(async () => { void result.current.handleStartEditTimelineItem('activity', activity); });
  expect(result.current.editTimelineValues.activityTitle).toBe('Lunch');

  act(() => result.current.setEditTimelineValues({ ...result.current.editTimelineValues, activityTitle: 'Typed' }));
  await act(async () => resolve(directory));

  expect(result.current.editTimelineValues.activityTitle).toBe('Typed');
});

test('cancel leaves edit mode and clears the form', async () => {
  const { result } = setup();
  await act(() => result.current.handleStartEditTimelineItem('note', note));
  act(() => result.current.handleCancelEditTimelineItem());
  expect(result.current.editingTimelineItem).toBeNull();
  expect(result.current.editTimelineValues).toEqual({});
});

test('a note with blank content is not saved', async () => {
  const { result } = setup();
  await act(() => result.current.handleStartEditTimelineItem('note', { ...note, content: '   ' }));
  await act(() => result.current.handleUpdateNote(1));
  expect(notes.updateNote).not.toHaveBeenCalled();
  expect(result.current.editingTimelineItem).toEqual({ type: 'note', id: 1 });
});

test('saving a note sends the edited date, refreshes and leaves edit mode', async () => {
  const { result, onRefresh } = setup();
  await act(() => result.current.handleStartEditTimelineItem('note', note));
  act(() => result.current.setEditTimelineValues({ noteContent: 'Changed', noteDate: '2026-03-20' }));
  await act(() => result.current.handleUpdateNote(1));
  expect(notes.updateNote).toHaveBeenCalledWith(1, {
    content: 'Changed',
    date: '2026-03-20T00:00:00.000Z',
    contact_id: 5,
  });
  expect(onRefresh).toHaveBeenCalledTimes(1);
  expect(result.current.editingTimelineItem).toBeNull();
});

test('saving a note with its date cleared falls back to now', async () => {
  vi.useFakeTimers({ toFake: ['Date'] });
  vi.setSystemTime(new Date('2026-06-15T12:00:00Z'));
  const { result } = setup();
  await act(() => result.current.handleStartEditTimelineItem('note', { ...note, date: '' }));
  await act(() => result.current.handleUpdateNote(1));
  expect(notes.updateNote).toHaveBeenCalledWith(1, expect.objectContaining({ date: '2026-06-15T12:00:00.000Z' }));
});

test('a failed note save notifies and keeps the edit open', async () => {
  quietLogs();
  notes.updateNote.mockRejectedValue(new Error('boom'));
  const { result, onRefresh, showError } = setup();
  await act(() => result.current.handleStartEditTimelineItem('note', note));
  await act(() => result.current.handleUpdateNote(1));
  expect(showError).toHaveBeenCalledWith('boom');
  expect(onRefresh).not.toHaveBeenCalled();
  expect(result.current.editTimelineValues.noteContent).toBe('Hello');
});

test('an activity with a blank title is not saved', async () => {
  const { result } = setup();
  await act(() => result.current.handleStartEditTimelineItem('activity', { ...activity, title: ' ' }));
  await act(() => result.current.handleUpdateActivity(2));
  expect(activities.updateActivity).not.toHaveBeenCalled();
});

test('saving an activity reduces its contacts to ids', async () => {
  const { result, onRefresh } = setup();
  await act(() => result.current.handleStartEditTimelineItem('activity', activity));
  await act(() => result.current.handleUpdateActivity(2));
  expect(activities.updateActivity).toHaveBeenCalledWith(2, {
    title: 'Lunch',
    description: 'Pizza',
    location: 'Berlin',
    date: '2026-04-01T00:00:00.000Z',
    contact_ids: [9],
  });
  expect(onRefresh).toHaveBeenCalledTimes(1);
  expect(result.current.editingTimelineItem).toBeNull();
});

test('a failed activity save notifies and keeps the edit open', async () => {
  quietLogs();
  activities.updateActivity.mockRejectedValue(new Error('boom'));
  const { result, showError } = setup();
  await act(() => result.current.handleStartEditTimelineItem('activity', activity));
  await act(() => result.current.handleUpdateActivity(2));
  expect(showError).toHaveBeenCalledWith('boom');
  expect(result.current.editingTimelineItem).toEqual({ type: 'activity', id: 2 });
});

test('deleting a note or activity refreshes and leaves edit mode', async () => {
  const { result, onRefresh } = setup();
  await act(() => result.current.handleStartEditTimelineItem('note', note));
  await act(() => result.current.handleDeleteNote(1));
  expect(notes.deleteNote).toHaveBeenCalledWith(1);
  expect(result.current.editingTimelineItem).toBeNull();
  await act(() => result.current.handleStartEditTimelineItem('activity', activity));
  await act(() => result.current.handleDeleteActivity(2));
  expect(activities.deleteActivity).toHaveBeenCalledWith(2);
  expect(result.current.editingTimelineItem).toBeNull();
  expect(onRefresh).toHaveBeenCalledTimes(2);
});

test('a failed delete notifies and keeps the edit open', async () => {
  quietLogs();
  notes.deleteNote.mockRejectedValue(new Error('n'));
  activities.deleteActivity.mockRejectedValue(new Error('a'));
  const { result, onRefresh, showError } = setup();
  await act(() => result.current.handleStartEditTimelineItem('note', note));
  await act(() => result.current.handleDeleteNote(1));
  await act(() => result.current.handleDeleteActivity(2));
  expect(showError.mock.calls).toEqual([['n'], ['a']]);
  expect(onRefresh).not.toHaveBeenCalled();
  expect(result.current.editingTimelineItem).toEqual({ type: 'note', id: 1 });
});
