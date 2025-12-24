import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Dialog,
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
import { Reminder, ReminderFormData } from '../api/reminders';

interface ReminderDialogProps {
  open: boolean;
  onClose: () => void;
  onSave: (reminderData: ReminderFormData) => Promise<void>;
  reminder?: Reminder | null;
  contactId: number;
}

export default function ReminderDialog({
  open,
  onClose,
  onSave,
  reminder,
  contactId
}: ReminderDialogProps) {
  const { t } = useTranslation();
  const [message, setMessage] = useState('');
  const [byMail, setByMail] = useState(true);
  const [remindAt, setRemindAt] = useState('');
  const [recurrence, setRecurrence] = useState<ReminderFormData['recurrence']>('once');
  const [reoccurFromCompletion, setReoccurFromCompletion] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (reminder) {
      setMessage(reminder.message);
      setByMail(reminder.by_mail);
      setRemindAt(reminder.remind_at.split('T')[0]); // Extract date part
      setRecurrence(reminder.recurrence);
      setReoccurFromCompletion(reminder.reoccur_from_completion);
    } else {
      // Reset form for new reminder
      setMessage('');
      setByMail(true);
      setRemindAt(new Date().toISOString().split('T')[0]);
      setRecurrence('once');
      setReoccurFromCompletion(true);
    }
    setError(null);
  }, [reminder, open]);

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
      console.error('Error saving reminder:', err);
      setError(err instanceof Error ? err.message : t('reminders.saveFailed'));
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
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
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
            label={t('reminders.date')}
            type="date"
            value={remindAt}
            onChange={(e) => setRemindAt(e.target.value)}
            required
            fullWidth
            InputLabelProps={{ shrink: true }}
            inputProps={{ min: new Date().toISOString().split('T')[0] }}
          />

          <TextField
            label={t('reminders.recurrence.label')}
            select
            value={recurrence}
            onChange={(e) => setRecurrence(e.target.value as ReminderFormData['recurrence'])}
            required
            fullWidth
          >
            {recurrenceOptions.map((option) => (
              <MenuItem key={option.value} value={option.value}>
                {option.label}
              </MenuItem>
            ))}
          </TextField>

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
    </Dialog>
  );
}
