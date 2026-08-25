import { useState, useCallback } from 'react';
import {
  getEmploymentHistory,
  createEmploymentHistory,
  updateEmploymentHistory,
  deleteEmploymentHistory,
  endEmploymentHistory,
  EmploymentHistoryEntry,
  EmploymentHistoryFormData,
} from '../api/employmentHistory';
import { handleFetchError, handleError, ErrorNotifier } from '../utils/errorHandler';

export function useEmploymentHistory(
  contactId: string | undefined,
  notifier?: ErrorNotifier,
  onContactChanged?: () => void | Promise<void>
) {
  const [entries, setEntries] = useState<EmploymentHistoryEntry[]>([]);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingEntry, setEditingEntry] = useState<EmploymentHistoryEntry | null>(null);
  const [loading, setLoading] = useState(false);

  const refreshEmploymentHistory = useCallback(async () => {
    if (!contactId) return;
    setLoading(true);
    try {
      const response = await getEmploymentHistory(contactId);
      setEntries(response.employment_history || []);
    } catch (err) {
      handleFetchError(err, 'fetching employment history');
    } finally {
      setLoading(false);
    }
  }, [contactId]);

  const handleSaveEntry = async (data: EmploymentHistoryFormData) => {
    if (!contactId) return;
    try {
      if (editingEntry) {
        await updateEmploymentHistory(contactId, editingEntry.ID, data);
      } else {
        await createEmploymentHistory(contactId, data);
      }
      await refreshEmploymentHistory();
      await onContactChanged?.();
      setDialogOpen(false);
      setEditingEntry(null);
    } catch (err) {
      handleError(err, { operation: 'saving employment history entry' }, notifier);
      throw err;
    }
  };

  const handleEditEntry = (entry: EmploymentHistoryEntry) => {
    setEditingEntry(entry);
    setDialogOpen(true);
  };

  const handleDeleteEntry = async (entryId: number) => {
    if (!contactId) return;
    try {
      await deleteEmploymentHistory(contactId, entryId);
      await refreshEmploymentHistory();
      await onContactChanged?.();
    } catch (err) {
      handleError(err, { operation: 'deleting employment history entry' }, notifier);
      throw err;
    }
  };

  const handleEndEntry = async (entryId: number) => {
    if (!contactId) return;
    try {
      await endEmploymentHistory(contactId, entryId);
      await refreshEmploymentHistory();
      await onContactChanged?.();
    } catch (err) {
      handleError(err, { operation: 'ending employment history entry' }, notifier);
      throw err;
    }
  };

  const handleAddEntry = () => {
    setEditingEntry(null);
    setDialogOpen(true);
  };

  return {
    entries,
    dialogOpen,
    editingEntry,
    loading,
    refreshEmploymentHistory,
    handleSaveEntry,
    handleEditEntry,
    handleDeleteEntry,
    handleEndEntry,
    handleAddEntry,
    setDialogOpen,
    setEditingEntry,
  };
}
