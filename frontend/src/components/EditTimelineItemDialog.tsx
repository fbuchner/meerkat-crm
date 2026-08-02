import { useTranslation } from 'react-i18next';
import {
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  Box,
  Autocomplete,
  Chip,
  IconButton,
  CircularProgress,
} from '@mui/material';
import AppDialog from './AppDialog';
import DeleteIcon from '@mui/icons-material/Delete';
import { useState, useCallback, useEffect } from 'react';
import { Contact, getContacts } from '../api/contacts';
import { handleFetchError } from '../utils/errorHandler';

export interface TimelineItemValues {
  noteContent?: string;
  noteDate?: string;
  noteContactId?: number;
  noteContactName?: string;
  activityTitle?: string;
  activityDescription?: string;
  activityLocation?: string;
  activityDate?: string;
  activityContacts?: { ID: number; firstname: string; lastname: string; nickname?: string }[];
}

interface EditTimelineItemDialogProps {
  open: boolean;
  onClose: () => void;
  onSave: () => void;
  onDelete: () => void;
  type: 'note' | 'activity';
  values: TimelineItemValues;
  onChange: (values: TimelineItemValues) => void;
  allContacts: { ID: number; firstname: string; lastname: string; nickname?: string }[];
}

interface ContactBrief {
  ID: number;
  firstname: string;
  lastname: string;
  nickname?: string;
}

function toContactBrief(c: Contact): ContactBrief {
  return { ID: c.ID, firstname: c.firstname, lastname: c.lastname, nickname: c.nickname };
}

function contactLabel(contact: ContactBrief): string {
  if (contact.nickname) {
    return `${contact.firstname} "${contact.nickname}" ${contact.lastname}`;
  }
  return `${contact.firstname} ${contact.lastname}`;
}

export default function EditTimelineItemDialog({
  open,
  onClose,
  onSave,
  onDelete,
  type,
  values,
  onChange,
  allContacts
}: EditTimelineItemDialogProps) {
  const { t } = useTranslation();

  const [noteContacts, setNoteContacts] = useState<ContactBrief[]>([]);
  const [noteContactsLoading, setNoteContactsLoading] = useState(false);
  const [noteSearchInput, setNoteSearchInput] = useState('');
  const [noteSelectedContact, setNoteSelectedContact] = useState<ContactBrief | null>(null);

  const loadNoteContacts = useCallback(async (search: string = '') => {
    setNoteContactsLoading(true);
    try {
      const response = await getContacts({ limit: 40, search });
      setNoteContacts((response.contacts || []).map(toContactBrief));
    } catch (err) {
      handleFetchError(err, 'loading contacts');
    } finally {
      setNoteContactsLoading(false);
    }
  }, []);

  // Load initial contacts when dialog opens in note mode.
  useEffect(() => {
    if (open && type === 'note') {
      loadNoteContacts();
    }
  }, [open, type, loadNoteContacts]);

  // Debounced contact search (note mode only).
  useEffect(() => {
    if (!open || type !== 'note') return;
    const timeoutId = setTimeout(() => {
      loadNoteContacts(noteSearchInput);
    }, 300);
    return () => clearTimeout(timeoutId);
  }, [noteSearchInput, open, type, loadNoteContacts]);

  // When the contact name is resolved, sync it into the Autocomplete value
  // so the picker shows the currently assigned contact.
  useEffect(() => {
    if (type === 'note' && values.noteContactId != null && values.noteContactName && !noteSelectedContact) {
      setNoteSelectedContact({
        ID: values.noteContactId,
        firstname: values.noteContactName,
        lastname: '',
      });
    }
  }, [type, values.noteContactId, values.noteContactName, noteSelectedContact]);

  // When a noteContactId is set but no resolved name yet, try to resolve it
  // from the loaded contact list.
  useEffect(() => {
    if (type === 'note' && values.noteContactId != null && !values.noteContactName && noteContacts.length > 0) {
      const found = noteContacts.find((c) => c.ID === values.noteContactId);
      if (found) {
        onChange({ ...values, noteContactName: contactLabel(found) });
      }
    }
  }, [type, values.noteContactId, values.noteContactName, noteContacts, onChange]);

  // Resolve contact name on first open if not already resolved (do a broader
  // fetch so the user immediately sees who the note is assigned to).
  useEffect(() => {
    if (open && type === 'note' && values.noteContactId != null && !values.noteContactName) {
      getContacts({ limit: 200 }).then((resp) => {
        const found = (resp.contacts || []).find((c) => c.ID === values.noteContactId);
        if (found) {
          onChange({ ...values, noteContactName: contactLabel(toContactBrief(found)) });
        }
      }).catch(() => {});
    }
    // Only run on open — values.noteContactId / noteContactName are deliberately
    // excluded so the fetch only fires once per open.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, type]);

  const handleSave = () => {
    onSave();
  };

  const handleDelete = () => {
    if (window.confirm(t('contactDetail.confirmDelete'))) {
      onDelete();
    }
  };

  return (
    <AppDialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>
        {type === 'note' ? t('contactDetail.editNote') : t('contactDetail.editActivity')}
      </DialogTitle>
      <DialogContent>
        {type === 'note' ? (
          // Edit Note
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
            {values.noteContactName ? (
              <Box display="flex" alignItems="center" gap={1}>
                <Chip
                  label={`${t('inbox.assignedTo')} ${values.noteContactName}`}
                  onDelete={() => { setNoteSelectedContact(null); onChange({ ...values, noteContactId: undefined, noteContactName: undefined }); }}
                  size="small"
                />
              </Box>
            ) : null}
            <Autocomplete
              options={noteContacts}
              getOptionLabel={contactLabel}
              value={noteSelectedContact}
              onChange={(_, value) => {
                setNoteSelectedContact(value);
                if (value) {
                  onChange({ ...values, noteContactId: value.ID, noteContactName: contactLabel(value) });
                }
              }}
              onInputChange={(_, value, reason) => {
                if (reason === 'input') setNoteSearchInput(value);
              }}
              loading={noteContactsLoading}
              filterOptions={(x) => x}
              renderInput={(params) => (
                <TextField
                  {...params}
                  label={t('inbox.assignContact')}
                  placeholder={t('inbox.searchContacts')}
                  InputProps={{
                    ...params.InputProps,
                    endAdornment: (
                      <>
                        {noteContactsLoading ? <CircularProgress color="inherit" size={20} /> : null}
                        {params.InputProps.endAdornment}
                      </>
                    ),
                  }}
                />
              )}
              isOptionEqualToValue={(option, value) => option.ID === value.ID}
              noOptionsText={noteSearchInput ? t('inbox.noContactsFound') : t('inbox.typeToSearch')}
            />
            <TextField
              fullWidth
              multiline
              rows={6}
              value={values.noteContent || ''}
              onChange={(e) => onChange({ ...values, noteContent: e.target.value })}
              label={t('noteDialog.content')}
              placeholder={t('noteDialog.contentPlaceholder')}
              autoFocus
            />
            <TextField
              fullWidth
              type="date"
              value={values.noteDate || ''}
              onChange={(e) => onChange({ ...values, noteDate: e.target.value })}
              label={t('noteDialog.date')}
              InputLabelProps={{ shrink: true }}
            />
          </Box>
        ) : (
          // Edit Activity
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
            <TextField
              fullWidth
              value={values.activityTitle || ''}
              onChange={(e) => onChange({ ...values, activityTitle: e.target.value })}
              label={t('activityDialog.activityTitle')}
              autoFocus
            />
            <TextField
              fullWidth
              multiline
              rows={3}
              value={values.activityDescription || ''}
              onChange={(e) => onChange({ ...values, activityDescription: e.target.value })}
              label={t('activityDialog.description')}
            />
            <TextField
              fullWidth
              value={values.activityLocation || ''}
              onChange={(e) => onChange({ ...values, activityLocation: e.target.value })}
              label={t('activityDialog.location')}
            />
            <TextField
              fullWidth
              type="date"
              value={values.activityDate || ''}
              onChange={(e) => onChange({ ...values, activityDate: e.target.value })}
              label={t('activityDialog.date')}
              InputLabelProps={{ shrink: true }}
            />
            <Autocomplete
              multiple
              options={allContacts}
              getOptionLabel={(contact) => `${contact.firstname}${contact.nickname ? ` "${contact.nickname}"` : ''} ${contact.lastname}`}
              value={values.activityContacts || []}
              onChange={(_, newValue) => onChange({ ...values, activityContacts: newValue })}
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
                    label={`${contact.firstname}${contact.nickname ? ` "${contact.nickname}"` : ''} ${contact.lastname}`}
                    {...getTagProps({ index })}
                    size="small"
                    key={contact.ID}
                  />
                ))
              }
            />
          </Box>
        )}
      </DialogContent>
      <DialogActions sx={{ justifyContent: 'space-between', px: 3, pb: 2 }}>
        <IconButton 
          color="error"
          onClick={handleDelete}
          title={t('contactDetail.delete')}
        >
          <DeleteIcon />
        </IconButton>
        <Box>
          <Button onClick={onClose} sx={{ mr: 1 }}>
            {t('contactDetail.cancel')}
          </Button>
          <Button onClick={handleSave} variant="contained">
            {t('contactDetail.save')}
          </Button>
        </Box>
      </DialogActions>
    </AppDialog>
  );
}
