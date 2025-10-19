import React, { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  Box,
} from '@mui/material';
import { useTranslation } from 'react-i18next';

interface AddNoteDialogProps {
  open: boolean;
  onClose: () => void;
  onSave: (content: string, date: string) => Promise<void>;
}

export default function AddNoteDialog({ open, onClose, onSave }: AddNoteDialogProps) {
  const { t } = useTranslation();
  const [content, setContent] = useState('');
  const [date, setDate] = useState(new Date().toISOString().split('T')[0]);
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);

  const handleSave = async () => {
    if (!content.trim()) {
      setError(t('noteDialog.required'));
      return;
    }

    setSaving(true);
    try {
      await onSave(content, date);
      handleClose();
    } catch (err) {
      setError('Failed to save note');
    } finally {
      setSaving(false);
    }
  };

  const handleClose = () => {
    setContent('');
    setDate(new Date().toISOString().split('T')[0]);
    setError('');
    onClose();
  };

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
      <DialogTitle>{t('noteDialog.title')}</DialogTitle>
      <DialogContent>
        <Box sx={{ pt: 1, display: 'flex', flexDirection: 'column', gap: 2 }}>
          <TextField
            label={t('noteDialog.content')}
            placeholder={t('noteDialog.contentPlaceholder')}
            multiline
            rows={4}
            value={content}
            onChange={(e) => {
              setContent(e.target.value);
              setError('');
            }}
            error={!!error}
            helperText={error}
            fullWidth
            required
            autoFocus
          />
          <TextField
            label={t('noteDialog.date')}
            type="date"
            value={date}
            onChange={(e) => setDate(e.target.value)}
            fullWidth
            InputLabelProps={{
              shrink: true,
            }}
          />
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose} disabled={saving}>
          {t('noteDialog.cancel')}
        </Button>
        <Button onClick={handleSave} variant="contained" disabled={saving}>
          {t('noteDialog.save')}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
