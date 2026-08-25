// frontend/src/components/AddEmploymentHistoryDialog.tsx
import { useState, useEffect } from 'react';
import { DialogTitle, DialogContent, DialogActions, TextField, Button, Box } from '@mui/material';
import AppDialog from './AppDialog';
import { useTranslation } from 'react-i18next';
import { EmploymentHistoryEntry, EmploymentHistoryFormData } from '../api/employmentHistory';
import { handleError, getErrorMessage } from '../utils/errorHandler';
import { useSnackbar } from '../context/SnackbarContext';

interface AddEmploymentHistoryDialogProps {
  open: boolean;
  onClose: () => void;
  onSave: (data: EmploymentHistoryFormData) => Promise<void>;
  entry?: EmploymentHistoryEntry | null;
}

const PARTIAL_DATE_RE = /^\d{4}(-\d{2}(-\d{2})?)?$/;

export default function AddEmploymentHistoryDialog({
  open,
  onClose,
  onSave,
  entry,
}: AddEmploymentHistoryDialogProps) {
  const { t } = useTranslation();
  const { showError } = useSnackbar();
  const [organization, setOrganization] = useState('');
  const [department, setDepartment] = useState('');
  const [jobTitle, setJobTitle] = useState('');
  const [role, setRole] = useState('');
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');
  const [notes, setNotes] = useState('');
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);

  const resetForm = () => {
    setOrganization('');
    setDepartment('');
    setJobTitle('');
    setRole('');
    setStartDate('');
    setEndDate('');
    setNotes('');
    setError('');
  };

  useEffect(() => {
    if (entry) {
      setOrganization(entry.organization || '');
      setDepartment(entry.department || '');
      setJobTitle(entry.job_title || '');
      setRole(entry.role || '');
      setStartDate(entry.start_date || '');
      setEndDate(entry.end_date || '');
      setNotes(entry.notes || '');
      setError('');
    } else {
      resetForm();
    }
  }, [entry, open]);

  const handleClose = () => {
    resetForm();
    onClose();
  };

  const validDate = (v: string) => v === '' || PARTIAL_DATE_RE.test(v);

  const handleSave = async () => {
    if (!organization.trim()) {
      setError(t('employmentHistory.organizationRequired'));
      return;
    }
    if (!validDate(startDate) || !validDate(endDate)) {
      setError(t('employmentHistory.dateFormatError'));
      return;
    }

    setSaving(true);
    try {
      await onSave({
        organization: organization.trim(),
        department: department.trim() || undefined,
        job_title: jobTitle.trim() || undefined,
        role: role.trim() || undefined,
        start_date: startDate.trim() || undefined,
        end_date: endDate.trim() || undefined,
        notes: notes.trim() || undefined,
      });
      handleClose();
    } catch (err) {
      handleError(err, { operation: 'saving employment history entry' }, { showError });
      setError(getErrorMessage(err));
    } finally {
      setSaving(false);
    }
  };

  const isEditing = !!entry;

  return (
    <AppDialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
      <DialogTitle>
        {isEditing ? t('employmentHistory.editEntry') : t('employmentHistory.addEntry')}
      </DialogTitle>
      <DialogContent>
        <Box sx={{ pt: 1, display: 'flex', flexDirection: 'column', gap: 2 }}>
          <TextField
            label={t('contacts.organization')}
            value={organization}
            onChange={(e) => { setOrganization(e.target.value); setError(''); }}
            fullWidth
            required
            autoFocus
            error={!!error && !organization.trim()}
          />
          <TextField
            label={t('contacts.jobTitle')}
            value={jobTitle}
            onChange={(e) => setJobTitle(e.target.value)}
            fullWidth
          />
          <TextField
            label={t('contacts.department')}
            value={department}
            onChange={(e) => setDepartment(e.target.value)}
            fullWidth
          />
          <TextField
            label={t('contacts.role')}
            value={role}
            onChange={(e) => setRole(e.target.value)}
            fullWidth
          />
          <Box sx={{ display: 'flex', gap: 2 }}>
            <TextField
              label={t('employmentHistory.startDate')}
              placeholder="YYYY, YYYY-MM, YYYY-MM-DD"
              helperText={t('employmentHistory.dateFormatHelp')}
              value={startDate}
              onChange={(e) => { setStartDate(e.target.value); setError(''); }}
              fullWidth
            />
            <TextField
              label={t('employmentHistory.endDate')}
              placeholder="YYYY, YYYY-MM, YYYY-MM-DD"
              helperText={t('employmentHistory.endDateHelp')}
              value={endDate}
              onChange={(e) => { setEndDate(e.target.value); setError(''); }}
              fullWidth
            />
          </Box>
          <TextField
            label={t('employmentHistory.notes')}
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            multiline
            rows={3}
            fullWidth
          />
          {error && (
            <Box sx={{ color: 'error.main', fontSize: '0.875rem' }}>{error}</Box>
          )}
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose} disabled={saving}>{t('common.cancel')}</Button>
        <Button onClick={handleSave} variant="contained" disabled={saving}>
          {saving ? t('common.saving') : t('common.save')}
        </Button>
      </DialogActions>
    </AppDialog>
  );
}
