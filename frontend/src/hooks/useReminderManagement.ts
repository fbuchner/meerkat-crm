import { useState } from 'react';
import {
  getRemindersForContact,
  createReminder,
  updateReminder,
  deleteReminder,
  completeReminder,
  Reminder,
  ReminderFormData
} from '../api/reminders';

export function useReminderManagement(contactId: string | undefined, token: string) {
  const [reminders, setReminders] = useState<Reminder[]>([]);
  const [reminderDialogOpen, setReminderDialogOpen] = useState(false);
  const [editingReminder, setEditingReminder] = useState<Reminder | null>(null);

  const refreshReminders = async () => {
    if (!contactId) return;
    try {
      const fetchedReminders = await getRemindersForContact(parseInt(contactId), token);
      setReminders(fetchedReminders);
    } catch (err) {
      console.error('Error fetching reminders:', err);
    }
  };

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
      console.error('Error saving reminder:', err);
      throw err;
    }
  };

  const handleCompleteReminder = async (reminderId: number) => {
    try {
      await completeReminder(reminderId, token);
      await refreshReminders();
    } catch (err) {
      console.error('Error completing reminder:', err);
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
      console.error('Error deleting reminder:', err);
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
