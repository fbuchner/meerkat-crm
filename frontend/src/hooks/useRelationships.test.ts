import { test, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useRelationships } from './useRelationships';
import * as api from '../api/relationships';
import type { Relationship, IncomingRelationship } from '../api/relationships';

vi.mock('../api/relationships');

const mocked = vi.mocked(api);
const outgoing: Relationship = { ID: 3, CreatedAt: '', UpdatedAt: '', name: 'Ben', type: 'Friend', contact_id: 1 };
const incoming = { ID: 4, type: 'Friend' } as unknown as IncomingRelationship;
const data = { name: 'Ben', type: 'Friend' };

const quietLogs = () => vi.spyOn(console, 'error').mockImplementation(() => {});

beforeEach(() => {
  vi.resetAllMocks();
  mocked.getRelationships.mockResolvedValue({ relationships: [outgoing] });
  mocked.getIncomingRelationships.mockResolvedValue({ incoming_relationships: [incoming] });
});

afterEach(() => {
  vi.restoreAllMocks();
});

test('refresh loads both directions of relationships', async () => {
  const { result } = renderHook(() => useRelationships('1'));
  await act(() => result.current.refreshRelationships());
  expect(mocked.getRelationships).toHaveBeenCalledWith('1');
  expect(mocked.getIncomingRelationships).toHaveBeenCalledWith('1');
  expect(result.current.relationships).toEqual([outgoing]);
  expect(result.current.incomingRelationships).toEqual([incoming]);
  expect(result.current.loading).toBe(false);
});

test('null lists from the server become empty lists', async () => {
  mocked.getRelationships.mockResolvedValue({ relationships: null } as unknown as api.RelationshipsResponse);
  mocked.getIncomingRelationships.mockResolvedValue({ incoming_relationships: null } as never);
  const { result } = renderHook(() => useRelationships('1'));
  await act(() => result.current.refreshRelationships());
  expect(result.current.relationships).toEqual([]);
  expect(result.current.incomingRelationships).toEqual([]);
});

test('a failure in either direction sets the error and ends loading', async () => {
  quietLogs();
  mocked.getIncomingRelationships.mockRejectedValueOnce(new Error('offline'));
  const { result } = renderHook(() => useRelationships('1'));
  await act(() => result.current.refreshRelationships());
  expect(result.current.error).toBe('offline');
  expect(result.current.loading).toBe(false);
  await act(() => result.current.refreshRelationships());
  expect(result.current.error).toBeNull();
});

test('without a contact id nothing is fetched or written', async () => {
  const { result } = renderHook(() => useRelationships(undefined));
  await act(async () => {
    await result.current.refreshRelationships();
    await result.current.handleSaveRelationship(data);
    await result.current.handleDeleteRelationship(3);
  });
  expect(mocked.getRelationships).not.toHaveBeenCalled();
  expect(mocked.createRelationship).not.toHaveBeenCalled();
  expect(mocked.deleteRelationship).not.toHaveBeenCalled();
});

test('save creates when not editing, refreshes and closes the dialog', async () => {
  const { result } = renderHook(() => useRelationships('1'));
  act(() => result.current.handleAddRelationship());
  expect(result.current.relationshipDialogOpen).toBe(true);
  await act(() => result.current.handleSaveRelationship(data));
  expect(mocked.createRelationship).toHaveBeenCalledWith('1', data);
  expect(mocked.updateRelationship).not.toHaveBeenCalled();
  expect(result.current.relationships).toEqual([outgoing]);
  expect(result.current.relationshipDialogOpen).toBe(false);
});

test('save updates the relationship being edited, then leaves edit mode', async () => {
  const { result } = renderHook(() => useRelationships('1'));
  act(() => result.current.handleEditRelationship(outgoing));
  expect(result.current.relationshipDialogOpen).toBe(true);
  await act(() => result.current.handleSaveRelationship(data));
  expect(mocked.updateRelationship).toHaveBeenCalledWith('1', 3, data);
  expect(mocked.createRelationship).not.toHaveBeenCalled();
  expect(result.current.editingRelationship).toBeNull();
});

test('add discards a relationship that was being edited', () => {
  const { result } = renderHook(() => useRelationships('1'));
  act(() => result.current.handleEditRelationship(outgoing));
  act(() => result.current.handleAddRelationship());
  expect(result.current.editingRelationship).toBeNull();
});

test('a failed save notifies, rethrows and keeps the dialog open', async () => {
  quietLogs();
  const showError = vi.fn();
  mocked.createRelationship.mockRejectedValue(new Error('boom'));
  const { result } = renderHook(() => useRelationships('1', { showError }));
  act(() => result.current.handleAddRelationship());
  await act(async () => {
    await expect(result.current.handleSaveRelationship(data)).rejects.toThrow('boom');
  });
  expect(showError).toHaveBeenCalledWith('boom');
  expect(result.current.relationshipDialogOpen).toBe(true);
});

test('delete calls the API and refreshes; failure notifies and rethrows', async () => {
  quietLogs();
  const showError = vi.fn();
  const { result } = renderHook(() => useRelationships('1', { showError }));
  await act(() => result.current.handleDeleteRelationship(3));
  expect(mocked.deleteRelationship).toHaveBeenCalledWith('1', 3);
  expect(mocked.getRelationships).toHaveBeenCalledTimes(1);

  mocked.deleteRelationship.mockRejectedValue(new Error('nope'));
  await act(async () => {
    await expect(result.current.handleDeleteRelationship(3)).rejects.toThrow('nope');
  });
  expect(showError).toHaveBeenCalledWith('nope');
  expect(mocked.getRelationships).toHaveBeenCalledTimes(1);
});
