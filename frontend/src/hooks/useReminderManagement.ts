import { useState, useCallback } from 'react';
import {
  getRemindersForContact,
  createReminder,
  updateReminder,
  deleteReminder,
  completeReminder,
  Reminder,
  ReminderFormData
} from '../api/reminders';
import { handleFetchError, handleError, ErrorNotifier } from '../utils/errorHandler';

export function useReminderManagement(
  contactId: string | undefined,
  token: string,
  notifier?: ErrorNotifier
) {
  const [reminders, setReminders] = useState<Reminder[]>([]);
  const [reminderDialogOpen, setReminderDialogOpen] = useState(false);
  const [editingReminder, setEditingReminder] = useState<Reminder | null>(null);
  const [error, setError] = useState<string | null>(null);

  const refreshReminders = useCallback(async () => {
    if (!contactId) return;
    setError(null);
    try {
      const fetchedReminders = await getRemindersForContact(parseInt(contactId), token);
      setReminders(fetchedReminders);
    } catch (err) {
      const message = handleFetchError(err, 'fetching reminders');
      setError(message);
    }
  }, [contactId, token]);

  const handleSaveReminder = async (reminderData: ReminderFormData) => {
    if (!contactId) return;

    try {
      if (editingReminder) {
        await updateReminder(editingReminder.ID, reminderData, token);
      } else {
        await createReminder(parseInt(contactId), reminderData, token);
      }
      await refreshReminders();
      setReminderDialogOpen(false);
      setEditingReminder(null);
    } catch (err) {
      handleError(err, { operation: 'saving reminder' }, notifier);
      throw err;
    }
  };

  const handleCompleteReminder = async (reminderId: number) => {
    try {
      await completeReminder(reminderId, token);
      await refreshReminders();
    } catch (err) {
      handleError(err, { operation: 'completing reminder' }, notifier);
      throw err;
    }
  };

  const handleEditReminder = (reminder: Reminder) => {
    setEditingReminder(reminder);
    setReminderDialogOpen(true);
  };

  const handleDeleteReminder = async (reminderId: number) => {
    try {
      await deleteReminder(reminderId, token);
      await refreshReminders();
    } catch (err) {
      handleError(err, { operation: 'deleting reminder' }, notifier);
      throw err;
    }
  };

  const handleAddReminder = () => {
    setEditingReminder(null);
    setReminderDialogOpen(true);
  };

  return {
    reminders,
    reminderDialogOpen,
    editingReminder,
    error,
    refreshReminders,
    handleSaveReminder,
    handleCompleteReminder,
    handleEditReminder,
    handleDeleteReminder,
    handleAddReminder,
    setReminderDialogOpen,
    setEditingReminder,
    setReminders
  };
}
