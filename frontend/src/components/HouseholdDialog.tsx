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
import { HOUSEHOLD_TYPES, Household, HouseholdType } from '../api/households';

export interface HouseholdFormData {
  name: string;
  type: HouseholdType;
}

interface HouseholdDialogProps {
  open: boolean;
  onClose: () => void;
  onSave: (data: HouseholdFormData) => Promise<void>;
  household?: Household | null;
}

export default function HouseholdDialog({ open, onClose, onSave, household }: HouseholdDialogProps) {
  const { t } = useTranslation();
  const isEditing = !!household;

  const [name, setName] = useState('');
  const [type, setType] = useState<HouseholdType>('family_unit');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (open) {
      setName(household?.name || '');
      setType(household?.type || 'family_unit');
      setError('');
    }
  }, [open, household]);

  const handleSave = async () => {
    if (!name.trim()) {
      setError(t('household.nameRequired'));
      return;
    }
    setSaving(true);
    try {
      await onSave({ name: name.trim(), type });
      onClose();
    } catch {
      setError(t('household.saveFailed'));
    } finally {
      setSaving(false);
    }
  };

  return (
    <AppDialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>
        {isEditing ? t('household.editHousehold') : t('household.createHousehold')}
      </DialogTitle>
      <DialogContent>
        <Box sx={{ pt: 1, display: 'flex', flexDirection: 'column', gap: 2 }}>
          <TextField
            label={t('household.name')}
            value={name}
            onChange={(e) => { setName(e.target.value); setError(''); }}
            fullWidth
            required
            autoFocus
          />
          <TextField
            select
            label={t('household.type')}
            value={type}
            onChange={(e) => { setType(e.target.value as HouseholdType); setError(''); }}
            fullWidth
            required
          >
            {HOUSEHOLD_TYPES.map((token) => (
              <MenuItem key={token} value={token}>
                {t(`household.types.${token}`, token)}
              </MenuItem>
            ))}
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
          {t('household.cancel')}
        </Button>
        <Button onClick={handleSave} variant="contained" disabled={saving}>
          {t('household.save')}
        </Button>
      </DialogActions>
    </AppDialog>
  );
}
