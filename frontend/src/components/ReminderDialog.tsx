import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import {
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  FormControlLabel,
  Checkbox,
  MenuItem,
  Box,
  Alert
} from '@mui/material';
import AppDialog from './AppDialog';
import { Reminder, ReminderFormData } from '../api/reminders';
import { getErrorMessage } from '../utils/errorHandler';

interface InitialReminderValues {
  message?: string;
  recurrence?: ReminderFormData['recurrence'];
}

interface ReminderDialogProps {
  open: boolean;
  onClose: () => void;
  onSave: (reminderData: ReminderFormData) => Promise<void>;
  reminder?: Reminder | null;
  contactId: number;
  initialValues?: InitialReminderValues;
}

export default function ReminderDialog({
  open,
  onClose,
  onSave,
  reminder,
  contactId,
  initialValues
}: ReminderDialogProps) {
  const { t } = useTranslation();
  const [message, setMessage] = useState('');
  const [byMail, setByMail] = useState(true);
  const [remindAt, setRemindAt] = useState('');
  const [recurrence, setRecurrence] = useState<ReminderFormData['recurrence']>('once');
  const [reoccurFromCompletion, setReoccurFromCompletion] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const getDateForRecurrence = (rec: ReminderFormData['recurrence']): string => {
    const d = new Date();
    switch (rec) {
      case 'weekly':      d.setDate(d.getDate() + 7); break;
      case 'monthly':     d.setMonth(d.getMonth() + 1); break;
      case 'quarterly':   d.setMonth(d.getMonth() + 3); break;
      case 'six-months':  d.setMonth(d.getMonth() + 6); break;
      case 'yearly':      d.setFullYear(d.getFullYear() + 1); break;
    }
    return d.toISOString().split('T')[0];
  };

  useEffect(() => {
    if (reminder) {
      setMessage(reminder.message);
      setByMail(reminder.by_mail);
      setRemindAt(reminder.remind_at.split('T')[0]); // Extract date part
      setRecurrence(reminder.recurrence);
      setReoccurFromCompletion(reminder.reoccur_from_completion);
    } else {
      // Reset form for new reminder, using initialValues if provided
      const initialRec = initialValues?.recurrence || 'once';
      setMessage(initialValues?.message || '');
      setByMail(true);
      setRecurrence(initialRec);
      setRemindAt(getDateForRecurrence(initialRec));
      setReoccurFromCompletion(true);
    }
    setError(null);
  }, [reminder, open, initialValues]);

  const handleRecurrenceChange = (newRecurrence: ReminderFormData['recurrence']) => {
    setRecurrence(newRecurrence);
    if (!reminder) {
      setRemindAt(getDateForRecurrence(newRecurrence));
    }
  };

  const handleSave = async () => {
    // Validation
    if (!message.trim()) {
      setError(t('reminders.messageRequired'));
      return;
    }
    if (!remindAt) {
      setError(t('reminders.dateRequired'));
      return;
    }

    try {
      setLoading(true);
      setError(null);

      const reminderData: ReminderFormData = {
        message: message.trim(),
        by_mail: byMail,
        remind_at: new Date(remindAt).toISOString(),
        recurrence,
        reoccur_from_completion: reoccurFromCompletion,
        contact_id: contactId
      };

      await onSave(reminderData);
      onClose();
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  const recurrenceOptions = [
    { value: 'once', label: t('reminders.recurrence.once') },
    { value: 'weekly', label: t('reminders.recurrence.weekly') },
    { value: 'monthly', label: t('reminders.recurrence.monthly') },
    { value: 'quarterly', label: t('reminders.recurrence.quarterly') },
    { value: 'six-months', label: t('reminders.recurrence.six-months') },
    { value: 'yearly', label: t('reminders.recurrence.yearly') }
  ];

  return (
    <AppDialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>
        {reminder ? t('reminders.editReminder') : t('reminders.addReminder')}
      </DialogTitle>
      <DialogContent>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
          {error && <Alert severity="error">{error}</Alert>}

          <TextField
            label={t('reminders.message')}
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            multiline
            rows={3}
            required
            fullWidth
            placeholder={t('reminders.messagePlaceholder')}
            inputProps={{ maxLength: 500 }}
            helperText={`${message.length}/500`}
          />

          <TextField
            label={t('reminders.recurrence.label')}
            select
            value={recurrence}
            onChange={(e) => handleRecurrenceChange(e.target.value as ReminderFormData['recurrence'])}
            required
            fullWidth
          >
            {recurrenceOptions.map((option) => (
              <MenuItem key={option.value} value={option.value}>
                {option.label}
              </MenuItem>
            ))}
          </TextField>

          <TextField
            label={t('reminders.date')}
            type="date"
            value={remindAt}
            onChange={(e) => setRemindAt(e.target.value)}
            required
            fullWidth
            slotProps={{ inputLabel: { shrink: true }, htmlInput: { min: new Date().toISOString().split('T')[0] } }}
          />

          {recurrence !== 'once' && (
            <FormControlLabel
              control={
                <Checkbox
                  checked={reoccurFromCompletion}
                  onChange={(e) => setReoccurFromCompletion(e.target.checked)}
                />
              }
              label={t('reminders.reoccurFromCompletion')}
            />
          )}

          <FormControlLabel
            control={
              <Checkbox
                checked={byMail}
                onChange={(e) => setByMail(e.target.checked)}
              />
            }
            label={t('reminders.sendEmail')}
          />
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={loading}>
          {t('common.cancel')}
        </Button>
        <Button onClick={handleSave} variant="contained" disabled={loading}>
          {loading ? t('common.saving') : t('common.save')}
        </Button>
      </DialogActions>
    </AppDialog>
  );
}
