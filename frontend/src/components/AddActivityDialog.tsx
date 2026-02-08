import { useState, useEffect, useCallback } from 'react';
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
  CircularProgress,
} from '@mui/material';
import { useTranslation } from 'react-i18next';
import { Contact, getContacts } from '../api/contacts';

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
  preselectedContactId?: number;
}

export default function AddActivityDialog({
  open,
  onClose,
  onSave,
  preselectedContactId,
}: AddActivityDialogProps) {
  const { t } = useTranslation();
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [location, setLocation] = useState('');
  const [date, setDate] = useState(new Date().toISOString().split('T')[0]);
  const [selectedContacts, setSelectedContacts] = useState<Contact[]>([]);
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [searchInput, setSearchInput] = useState('');
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(false);

  const loadContacts = useCallback(async (search: string = '') => {
    setLoading(true);
    try {
      const response = await getContacts({ limit: 100, search });
      setContacts(response.contacts || []);
    } catch (err) {
      console.error('Failed to fetch contacts:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  // Load preselected contact on open
  useEffect(() => {
    if (open && preselectedContactId) {
      const loadPreselected = async () => {
        try {
          const response = await getContacts({ limit: 100 });
          const preselected = response.contacts.find((c: Contact) => c.ID === preselectedContactId);
          if (preselected) {
            setSelectedContacts([preselected]);
          }
          setContacts(response.contacts || []);
        } catch (err) {
          console.error('Failed to load preselected contact:', err);
        }
      };
      loadPreselected();
    } else if (open) {
      loadContacts('');
    }
  }, [open, preselectedContactId, loadContacts]);

  // Debounced search effect
  useEffect(() => {
    if (!open) return;
    
    const timeoutId = setTimeout(() => {
      loadContacts(searchInput);
    }, 300);

    return () => clearTimeout(timeoutId);
  }, [searchInput, open, loadContacts]);

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
    setSearchInput('');
    setContacts([]);
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
            options={contacts}
            getOptionLabel={getContactLabel}
            value={selectedContacts}
            onChange={(_, newValue) => setSelectedContacts(newValue)}
            onInputChange={(_, value) => setSearchInput(value)}
            inputValue={searchInput}
            loading={loading}
            filterOptions={(x) => x}
            isOptionEqualToValue={(option, value) => option.ID === value.ID}
            renderInput={(params) => (
              <TextField
                {...params}
                label={t('activityDialog.contacts')}
                placeholder={t('activityDialog.selectContacts')}
                InputProps={{
                  ...params.InputProps,
                  endAdornment: (
                    <>
                      {loading ? <CircularProgress color="inherit" size={20} /> : null}
                      {params.InputProps.endAdornment}
                    </>
                  ),
                }}
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
