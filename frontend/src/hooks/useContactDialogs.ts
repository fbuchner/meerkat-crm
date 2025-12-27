import { useState } from 'react';
import { createNote } from '../api/notes';
import { createActivity } from '../api/activities';
import { handleError, ErrorNotifier, getErrorMessage } from '../utils/errorHandler';

export function useContactDialogs(
  contactId: string | undefined,
  token: string,
  onRefresh: () => Promise<void>,
  notifier?: ErrorNotifier
) {
  const [noteDialogOpen, setNoteDialogOpen] = useState(false);
  const [activityDialogOpen, setActivityDialogOpen] = useState(false);

  const handleSaveNote = async (content: string, date: string) => {
    if (!contactId) return;

    try {
      await createNote(contactId, {
        content,
        date: new Date(date).toISOString(),
        contact_id: parseInt(contactId)
      }, token);
      await onRefresh();
    } catch (err) {
      handleError(err, { operation: 'saving note' }, notifier);
      throw new Error(getErrorMessage(err));
    }
  };

  const handleSaveActivity = async (activity: {
    title: string;
    description: string;
    location: string;
    date: string;
    contact_ids: number[];
  }) => {
    try {
      await createActivity({
        title: activity.title,
        description: activity.description,
        location: activity.location,
        date: new Date(activity.date).toISOString(),
        contact_ids: activity.contact_ids
      }, token);
      await onRefresh();
    } catch (err) {
      handleError(err, { operation: 'saving activity' }, notifier);
      throw new Error(getErrorMessage(err));
    }
  };

  return {
    noteDialogOpen,
    activityDialogOpen,
    setNoteDialogOpen,
    setActivityDialogOpen,
    handleSaveNote,
    handleSaveActivity
  };
}
