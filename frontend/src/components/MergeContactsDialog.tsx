import { useState, useCallback, useEffect } from 'react';
import {
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  Box,
  Autocomplete,
  CircularProgress,
  Typography,
  RadioGroup,
  FormControlLabel,
  Radio,
  Divider,
  Alert,
} from '@mui/material';
import { useTranslation } from 'react-i18next';
import AppDialog from './AppDialog';
import { Contact, getContacts } from '../api/contacts';
import { useContactMerge } from '../hooks/useContactMerge';
import { useSnackbar } from '../context/SnackbarContext';
import { getErrorMessage } from '../utils/errorHandler';

interface MergeContactsDialogProps {
  open: boolean;
  onClose: () => void;
  // Called after a successful commit with the surviving (keeper) contact's
  // id, since the currently-viewed contact (the loser) no longer exists --
  // the parent is expected to navigate there.
  onMerged: (keeperId: number) => void;
  currentContactId: number;
  currentContactUid: string;
  currentContactName: string;
}

// MergeContactsDialog is ticket N1's frontend entry point
// (docs/fork-plan/tickets/01-N1-contact-merge.md): opened from the
// currently-viewed contact's page, "merge into another contact" always
// treats the viewed contact as the loser and the picked contact as the
// keeper -- the viewed contact is the one that will disappear.
export default function MergeContactsDialog({
  open,
  onClose,
  onMerged,
  currentContactId,
  currentContactUid,
  currentContactName,
}: MergeContactsDialogProps) {
  const { t } = useTranslation();
  const { showError } = useSnackbar();
  const [selectedContact, setSelectedContact] = useState<Contact | null>(null);
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [contactsLoading, setContactsLoading] = useState(false);
  const [searchInput, setSearchInput] = useState('');

  const { preview, loading, committing, error, allConflictsResolved, loadPreview, setResolution, resolutions, commit, reset } =
    useContactMerge();

  const loadContacts = useCallback(
    async (search: string = '') => {
      setContactsLoading(true);
      try {
        const response = await getContacts({ limit: 100, search });
        setContacts(response.contacts.filter((c) => c.uid !== currentContactUid));
      } catch (err) {
        showError(getErrorMessage(err));
      } finally {
        setContactsLoading(false);
      }
    },
    [currentContactUid, showError]
  );

  useEffect(() => {
    if (open) loadContacts();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const timeoutId = setTimeout(() => loadContacts(searchInput), 300);
    return () => clearTimeout(timeoutId);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchInput]);

  const handleSelectContact = (contact: Contact | null) => {
    setSelectedContact(contact);
    if (contact) {
      loadPreview(contact.ID, currentContactId);
    } else {
      reset();
    }
  };

  const handleClose = () => {
    setSelectedContact(null);
    setSearchInput('');
    reset();
    onClose();
  };

  const handleCommit = async () => {
    if (!selectedContact) return;
    try {
      await commit(selectedContact.ID, currentContactId);
      onMerged(selectedContact.ID);
    } catch {
      // useContactMerge's commit already surfaced the error via the snackbar.
    }
  };

  const counts = preview?.association_counts;
  const hasAssociations =
    counts &&
    (counts.notes || counts.activities || counts.reminders || counts.reminder_completions ||
      counts.relationship_edges || counts.household_memberships || counts.circle_memberships ||
      counts.tags || counts.life_events || counts.life_event_references || counts.field_values ||
      counts.contact_sync_links);

  return (
    <AppDialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
      <DialogTitle>{t('contactMerge.title')}</DialogTitle>
      <DialogContent>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          {t('contactMerge.description', { name: currentContactName })}
        </Typography>

        <Autocomplete
          options={contacts}
          getOptionLabel={(option) => `${option.firstname} ${option.lastname}`}
          value={selectedContact}
          onChange={(_, value) => handleSelectContact(value)}
          onInputChange={(_, value, reason) => {
            if (reason === 'input') setSearchInput(value);
          }}
          loading={contactsLoading}
          filterOptions={(x) => x}
          renderInput={(params) => (
            <TextField
              {...params}
              label={t('contactMerge.selectTarget')}
              placeholder={t('contactMerge.searchContacts')}
              InputProps={{
                ...params.InputProps,
                endAdornment: (
                  <>
                    {contactsLoading ? <CircularProgress color="inherit" size={20} /> : null}
                    {params.InputProps.endAdornment}
                  </>
                ),
              }}
            />
          )}
          isOptionEqualToValue={(option, value) => option.uid === value.uid}
          noOptionsText={searchInput ? t('contactMerge.noContactsFound') : t('contactMerge.typeToSearch')}
        />

        {loading && (
          <Box sx={{ display: 'flex', justifyContent: 'center', my: 2 }}>
            <CircularProgress size={24} />
          </Box>
        )}

        {error && (
          <Alert severity="error" sx={{ mt: 2 }}>
            {error}
          </Alert>
        )}

        {preview && selectedContact && (
          <Box sx={{ mt: 2 }}>
            <Divider sx={{ mb: 2 }} />

            {preview.resolution.conflicts.length === 0 && preview.resolution.field_value_conflicts.length === 0 ? (
              <Alert severity="info" sx={{ mb: 2 }}>
                {t('contactMerge.noConflicts')}
              </Alert>
            ) : (
              <>
                <Typography variant="subtitle2" sx={{ mb: 1 }}>
                  {t('contactMerge.resolveConflicts')}
                </Typography>
                {preview.resolution.conflicts.map((conflict) => (
                  <Box key={conflict.field} sx={{ mb: 2 }}>
                    <Typography variant="body2" sx={{ fontWeight: 500 }}>
                      {conflict.label}
                    </Typography>
                    <RadioGroup
                      value={resolutions[conflict.field] ?? ''}
                      onChange={(e) => setResolution(conflict.field, e.target.value)}
                    >
                      <FormControlLabel
                        value={conflict.keeper_value}
                        control={<Radio size="small" />}
                        label={t('contactMerge.keepValue', { name: selectedContact.firstname, value: conflict.keeper_value })}
                      />
                      <FormControlLabel
                        value={conflict.loser_value}
                        control={<Radio size="small" />}
                        label={t('contactMerge.keepValue', { name: currentContactName, value: conflict.loser_value })}
                      />
                    </RadioGroup>
                  </Box>
                ))}
                {preview.resolution.field_value_conflicts.length > 0 && (
                  <>
                    <Typography variant="subtitle2" sx={{ mb: 1, mt: 2 }}>
                      {t('contactMerge.resolveFieldValueConflicts')}
                    </Typography>
                    {preview.resolution.field_value_conflicts.map((conflict) => (
                      <Box key={conflict.field} sx={{ mb: 2 }}>
                        <Typography variant="body2" sx={{ fontWeight: 500 }}>
                          {conflict.label}
                        </Typography>
                        <RadioGroup
                          value={resolutions[conflict.field] ?? ''}
                          onChange={(e) => setResolution(conflict.field, e.target.value)}
                        >
                          <FormControlLabel
                            value={conflict.keeper_value}
                            control={<Radio size="small" />}
                            label={t('contactMerge.keepValue', { name: selectedContact.firstname, value: conflict.keeper_value })}
                          />
                          <FormControlLabel
                            value={conflict.loser_value}
                            control={<Radio size="small" />}
                            label={t('contactMerge.keepValue', { name: currentContactName, value: conflict.loser_value })}
                          />
                        </RadioGroup>
                      </Box>
                    ))}
                  </>
                )}
              </>
            )}

            {hasAssociations && counts && (
              <Alert severity="info" sx={{ mt: 1 }}>
                {t('contactMerge.associationSummary', {
                  notes: counts.notes,
                  activities: counts.activities,
                  reminders: counts.reminders,
                  edges: counts.relationship_edges,
                })}
              </Alert>
            )}
          </Box>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose}>{t('common.cancel')}</Button>
        <Button
          variant="contained"
          color="primary"
          disabled={!preview || !allConflictsResolved || committing}
          onClick={handleCommit}
        >
          {committing ? <CircularProgress size={20} /> : t('contactMerge.mergeButton')}
        </Button>
      </DialogActions>
    </AppDialog>
  );
}
