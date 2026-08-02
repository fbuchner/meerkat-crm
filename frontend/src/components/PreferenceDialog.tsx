import { useState, useEffect } from 'react';
import {
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  Box,
  MenuItem,
  Typography,
} from '@mui/material';
import AppDialog from './AppDialog';
import { useTranslation } from 'react-i18next';
import {
  PREFERENCE_CATEGORIES,
  Preference,
  PreferenceInput,
  PreferenceSensitivity,
} from '../api/preferences';

export interface PreferenceFormData {
  category: string;
  key?: string;
  value: string;
  sensitivity: PreferenceSensitivity;
}

interface PreferenceDialogProps {
  open: boolean;
  onClose: () => void;
  onSave: (data: PreferenceFormData) => Promise<void>;
  preference?: Preference | null;
}

export default function PreferenceDialog({
  open,
  onClose,
  onSave,
  preference,
}: PreferenceDialogProps) {
  const { t } = useTranslation();
  const isEditing = !!preference;

  const [category, setCategory] = useState('food');
  const [key, setKey] = useState('');
  const [value, setValue] = useState('');
  const [sensitivity, setSensitivity] = useState<PreferenceSensitivity>('normal');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (open) {
      setCategory(preference?.category || 'food');
      setKey(preference?.key || '');
      setValue(preference?.value || '');
      setSensitivity(preference?.sensitivity || 'normal');
      setError('');
    }
  }, [open, preference]);

  const handleSave = async () => {
    if (!value.trim()) {
      setError(t('preference.validation.valueRequired'));
      return;
    }
    setSaving(true);
    try {
      const data: PreferenceFormData = {
        category,
        key: key.trim() || undefined,
        value: value.trim(),
        sensitivity,
      };
      await onSave(data);
      onClose();
    } catch {
      setError(t('preference.validation.saveFailed'));
    } finally {
      setSaving(false);
    }
  };

  return (
    <AppDialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>
        {isEditing ? t('preference.editTitle') : t('preference.createTitle')}
      </DialogTitle>
      <DialogContent>
        <Box sx={{ pt: 1, display: 'flex', flexDirection: 'column', gap: 2 }}>
          <TextField
            select
            label={t('preference.category')}
            value={category}
            onChange={(e) => { setCategory(e.target.value); setError(''); }}
            fullWidth
            required
          >
            {PREFERENCE_CATEGORIES.map((token) => (
              <MenuItem key={token} value={token}>
                {t(`preference.categories.${token}`, token)}
              </MenuItem>
            ))}
          </TextField>
          <TextField
            label={t('preference.value')}
            value={value}
            onChange={(e) => { setValue(e.target.value); setError(''); }}
            fullWidth
            required
          />
          <TextField
            label={t('preference.key')}
            value={key}
            onChange={(e) => setKey(e.target.value)}
            fullWidth
            helperText={t('preference.keyHint')}
          />
          <TextField
            select
            label={t('preference.sensitivity')}
            value={sensitivity}
            onChange={(e) => setSensitivity(e.target.value as PreferenceSensitivity)}
            fullWidth
          >
            <MenuItem value="normal">{t('preference.sensitivities.normal')}</MenuItem>
            <MenuItem value="private">{t('preference.sensitivities.private')}</MenuItem>
            <MenuItem value="secret">{t('preference.sensitivities.secret')}</MenuItem>
          </TextField>
          {error && (
            <Typography color="error" variant="body2">
              {error}
            </Typography>
          )}
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={saving}>
          {t('preference.cancel')}
        </Button>
        <Button onClick={handleSave} variant="contained" disabled={saving}>
          {t('preference.save')}
        </Button>
      </DialogActions>
    </AppDialog>
  );
}

// Rebuilds a PreferenceInput for the wire from a form payload + the target
// entity. source defaults to user, confidence to 1.0 — the same defaults the
// backend applies to a user-created preference.
export function toPreferenceInput(entityId: string, data: PreferenceFormData): PreferenceInput {
  return {
    entity_id: entityId,
    category: data.category,
    key: data.key,
    value: data.value,
    source: 'user',
    confidence: 1.0,
    sensitivity: data.sensitivity,
  };
}
