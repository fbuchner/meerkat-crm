import React, { useState, useEffect, useCallback } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  Box,
  Autocomplete,
  Chip,
} from '@mui/material';
import { useTranslation } from 'react-i18next';
import { API_BASE_URL, apiFetch } from '../api';

interface Contact {
  ID: number;
  firstname: string;
  lastname: string;
  nickname?: string;
}

interface AddActivityDialogProps {
  open: boolean;
  onClose: () => void;
  onSave: (activity: {
    title: string;
    description: string;
    location: string;
    date: string;
    contact_ids: number[];
  }) => Promise<void>;
  token: string;
  preselectedContactId?: number;
}

export default function AddActivityDialog({
  open,
  onClose,
  onSave,
  token,
  preselectedContactId,
}: AddActivityDialogProps) {
  const { t } = useTranslation();
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [location, setLocation] = useState('');
  const [date, setDate] = useState(new Date().toISOString().split('T')[0]);
  const [selectedContacts, setSelectedContacts] = useState<Contact[]>([]);
  const [allContacts, setAllContacts] = useState<Contact[]>([]);
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(false);

  const fetchContacts = useCallback(async () => {
    setLoading(true);
    try {
      const response = await apiFetch(`${API_BASE_URL}/contacts?page=1&limit=1000`, {
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      });

      if (response.ok) {
        const data = await response.json();
        setAllContacts(data.contacts || []);
        
        // Preselect contact if provided
        if (preselectedContactId) {
          const preselected = data.contacts.find((c: Contact) => c.ID === preselectedContactId);
          if (preselected) {
            setSelectedContacts([preselected]);
          }
        }
      }
    } catch (err) {
      console.error('Failed to fetch contacts:', err);
    } finally {
      setLoading(false);
    }
  }, [token, preselectedContactId]);

  useEffect(() => {
    if (open) {
      fetchContacts();
    }
  }, [open, fetchContacts]);

  const handleSave = async () => {
    if (!title.trim() || !date) {
      setError(t('activityDialog.required'));
      return;
    }

    setSaving(true);
    try {
      await onSave({
        title,
        description,
        location,
        date,
        contact_ids: selectedContacts.map((c) => c.ID),
      });
      handleClose();
    } catch (err) {
      setError('Failed to save activity');
    } finally {
      setSaving(false);
    }
  };

  const handleClose = () => {
    setTitle('');
    setDescription('');
    setLocation('');
    setDate(new Date().toISOString().split('T')[0]);
    setSelectedContacts([]);
    setError('');
    onClose();
  };

  const getContactLabel = (contact: Contact) => {
    return `${contact.firstname}${contact.nickname ? ` "${contact.nickname}"` : ''} ${contact.lastname}`;
  };

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
      <DialogTitle>{t('activityDialog.title')}</DialogTitle>
      <DialogContent>
        <Box sx={{ pt: 1, display: 'flex', flexDirection: 'column', gap: 2 }}>
          <TextField
            label={t('activityDialog.activityTitle')}
            placeholder={t('activityDialog.titlePlaceholder')}
            value={title}
            onChange={(e) => {
              setTitle(e.target.value);
              setError('');
            }}
            error={!!error}
            helperText={error}
            fullWidth
            required
            autoFocus
          />
          
          <TextField
            label={t('activityDialog.description')}
            placeholder={t('activityDialog.descriptionPlaceholder')}
            multiline
            rows={3}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            fullWidth
          />
          
          <TextField
            label={t('activityDialog.location')}
            placeholder={t('activityDialog.locationPlaceholder')}
            value={location}
            onChange={(e) => setLocation(e.target.value)}
            fullWidth
          />
          
          <TextField
            label={t('activityDialog.date')}
            type="date"
            value={date}
            onChange={(e) => setDate(e.target.value)}
            fullWidth
            required
            InputLabelProps={{
              shrink: true,
            }}
          />
          
          <Autocomplete
            multiple
            options={allContacts}
            getOptionLabel={getContactLabel}
            value={selectedContacts}
            onChange={(_, newValue) => setSelectedContacts(newValue)}
            loading={loading}
            renderInput={(params) => (
              <TextField
                {...params}
                label={t('activityDialog.contacts')}
                placeholder={t('activityDialog.selectContacts')}
              />
            )}
            renderTags={(value, getTagProps) =>
              value.map((contact, index) => (
                <Chip
                  label={getContactLabel(contact)}
                  {...getTagProps({ index })}
                  key={contact.ID}
                />
              ))
            }
          />
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose} disabled={saving}>
          {t('activityDialog.cancel')}
        </Button>
        <Button onClick={handleSave} variant="contained" disabled={saving}>
          {t('activityDialog.save')}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
