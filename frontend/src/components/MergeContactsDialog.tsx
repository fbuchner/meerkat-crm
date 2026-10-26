import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Alert,
  Autocomplete,
  Avatar,
  Box,
  Button,
  CircularProgress,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  IconButton,
  Radio,
  RadioGroup,
  Stack,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';
import SwapHorizIcon from '@mui/icons-material/SwapHoriz';
import AppDialog from './AppDialog';
import {
  Contact,
  getContacts,
  getMergePreview,
  mergeContacts,
  MergeConflict,
  MergePreview,
  MergeResolution,
} from '../api/contacts';
import { CONTACT_FIELDS } from '../contactFields';
import { useDateFormat } from '../DateFormatProvider';
import { useDebouncedValue } from '../hooks/useDebounce';
import { getErrorMessage, handleFetchError } from '../utils/errorHandler';

interface MergeContactsDialogProps {
  open: boolean;
  onClose: () => void;
  /** The contact the merge was started from; it is kept by default. */
  contactId: number;
  /** The contact to merge into it. When omitted the dialog asks for one first. */
  otherContactId?: number;
  onMerged: (merged: Contact, deletedId: number) => void;
}

const CUSTOM_FIELD_PREFIX = 'custom_fields.';
const DATE_FIELDS = new Set(['birthday', 'anniversary', 'deceased_date']);

// Label keys for conflict fields that are not in the toggleable CONTACT_FIELDS registry
const EXTRA_FIELD_LABELS: Record<string, string> = {
  firstname: 'contacts.firstname',
  lastname: 'contacts.lastname',
  deceased_date: 'contactDetail.deceasedDate',
  is_deceased: 'contacts.deceased',
  archived: 'contacts.archived',
  photo: 'contacts.merge.photo',
};

// Labels for the "true"/"false" values of the boolean conflict fields
const BOOLEAN_FIELD_VALUES: Record<string, { true: string; false: string }> = {
  is_deceased: { true: 'contacts.deceased', false: 'contacts.merge.notDeceased' },
  archived: { true: 'contacts.merge.archivedRemovesReminders', false: 'contacts.merge.active' },
};

export function contactDisplayName(contact: { firstname?: string; lastname?: string }): string {
  return [contact.firstname, contact.lastname].filter(Boolean).join(' ');
}

export default function MergeContactsDialog({
  open,
  onClose,
  contactId,
  otherContactId,
  onMerged,
}: MergeContactsDialogProps) {
  const { t } = useTranslation();
  const { formatBirthday } = useDateFormat();

  const [targetId, setTargetId] = useState(contactId);
  const [sourceId, setSourceId] = useState<number | null>(otherContactId ?? null);
  const [preview, setPreview] = useState<MergePreview | null>(null);
  const [resolutions, setResolutions] = useState<Record<string, MergeResolution>>({});
  const [loading, setLoading] = useState(false);
  const [merging, setMerging] = useState(false);
  const [error, setError] = useState('');

  const [searchInput, setSearchInput] = useState('');
  const debouncedSearch = useDebouncedValue(searchInput, 300);
  const [candidates, setCandidates] = useState<Contact[]>([]);
  const [candidatesLoading, setCandidatesLoading] = useState(false);

  // Start over whenever the dialog is (re)opened
  useEffect(() => {
    if (open) {
      setTargetId(contactId);
      setSourceId(otherContactId ?? null);
      setPreview(null);
      setError('');
      setSearchInput('');
    }
  }, [open, contactId, otherContactId]);

  // Contact picker for when no second contact was given
  useEffect(() => {
    if (!open || sourceId !== null) return;
    let cancelled = false;
    setCandidatesLoading(true);
    getContacts({ limit: 50, search: debouncedSearch, includeArchived: true })
      .then((response) => {
        if (!cancelled) setCandidates(response.contacts.filter((c) => c.ID !== contactId));
      })
      .catch((err) => handleFetchError(err, 'loading contacts'))
      .finally(() => {
        if (!cancelled) setCandidatesLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [open, sourceId, debouncedSearch, contactId]);

  useEffect(() => {
    if (!open || sourceId === null) return;
    // A swap starts a new request; a slower response for the old direction
    // must not land after it, or the merge would run the wrong way round.
    let cancelled = false;
    setLoading(true);
    setError('');
    getMergePreview(targetId, sourceId)
      .then((result) => {
        if (cancelled) return;
        setPreview(result);
        setResolutions(Object.fromEntries(result.conflicts.map((c) => [c.field, 'target' as MergeResolution])));
      })
      .catch((err) => {
        if (cancelled) return;
        setPreview(null);
        setError(getErrorMessage(err));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [open, targetId, sourceId]);

  const handleSwap = () => {
    if (sourceId === null) return;
    setPreview(null);
    setTargetId(sourceId);
    setSourceId(targetId);
  };

  const handleMerge = async () => {
    if (!preview) return;
    // Merge exactly what the preview on screen shows
    const keepId = preview.target.ID;
    const removeId = preview.source.ID;
    setMerging(true);
    setError('');
    try {
      const merged = await mergeContacts(keepId, removeId, resolutions);
      onMerged(merged, removeId);
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setMerging(false);
    }
  };

  const fieldLabel = (field: string): string => {
    if (field.startsWith(CUSTOM_FIELD_PREFIX)) {
      return field.slice(CUSTOM_FIELD_PREFIX.length);
    }
    const labelKey = EXTRA_FIELD_LABELS[field] ?? CONTACT_FIELDS.find((f) => f.key === field)?.labelKey;
    return labelKey ? t(labelKey) : field;
  };

  const renderValue = (conflict: MergeConflict, side: MergeResolution) => {
    if (conflict.field === 'photo' && preview) {
      const contact = side === 'target' ? preview.target : preview.source;
      return <Avatar src={contact.photo_thumbnail} sx={{ width: 40, height: 40 }} />;
    }
    const value = side === 'target' ? conflict.target_value : conflict.source_value;
    let display = value;
    const booleanLabels = BOOLEAN_FIELD_VALUES[conflict.field];
    if (booleanLabels) {
      display = t(value === 'true' ? booleanLabels.true : booleanLabels.false);
    } else if (DATE_FIELDS.has(conflict.field)) {
      display = formatBirthday(value);
    } else if (conflict.field === 'gender') {
      display = t(`contacts.${value}`, value);
    }
    return (
      <Typography variant="body2" sx={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
        {display}
      </Typography>
    );
  };

  const renderContactCard = (contact: Contact, role: 'keep' | 'remove') => (
    <Box
      sx={{
        flex: 1,
        minWidth: 0,
        p: 1.5,
        border: 1,
        borderColor: role === 'keep' ? 'primary.main' : 'divider',
        borderRadius: 1,
      }}
    >
      <Typography variant="overline" color={role === 'keep' ? 'primary' : 'text.secondary'}>
        {role === 'keep' ? t('contacts.merge.keep') : t('contacts.merge.remove')}
      </Typography>
      <Stack direction="row" spacing={1.5} alignItems="center">
        <Avatar src={contact.photo_thumbnail}>{contact.firstname?.[0]}</Avatar>
        <Typography variant="subtitle1" noWrap>
          {contactDisplayName(contact)}
        </Typography>
      </Stack>
    </Box>
  );

  const movedSummary = preview
    ? [
        preview.moved.notes > 0 && t('contacts.merge.notesCount', { count: preview.moved.notes }),
        preview.moved.activities > 0 && t('contacts.merge.activitiesCount', { count: preview.moved.activities }),
        preview.moved.reminders > 0 && t('contacts.merge.remindersCount', { count: preview.moved.reminders }),
        preview.moved.relationships > 0 && t('contacts.merge.relationshipsCount', { count: preview.moved.relationships }),
      ].filter(Boolean).join(', ')
    : '';

  return (
    <AppDialog open={open} onClose={merging ? undefined : onClose} maxWidth="md" fullWidth>
      <DialogTitle>{t('contacts.merge.title')}</DialogTitle>
      <DialogContent>
        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        {sourceId === null && (
          <Box sx={{ pt: 1 }}>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
              {t('contacts.merge.pickDescription')}
            </Typography>
            <Autocomplete
              options={candidates}
              getOptionLabel={(option) => contactDisplayName(option)}
              onChange={(_, value) => value && setSourceId(value.ID)}
              onInputChange={(_, value, reason) => {
                if (reason === 'input') setSearchInput(value);
              }}
              loading={candidatesLoading}
              filterOptions={(x) => x}
              noOptionsText={t('relationships.noContactsFound')}
              renderInput={(params) => (
                <TextField {...params} autoFocus label={t('contacts.merge.pickLabel')} placeholder={t('relationships.searchContacts')} />
              )}
            />
          </Box>
        )}

        {sourceId !== null && loading && (
          <Box display="flex" justifyContent="center" py={4}>
            <CircularProgress />
          </Box>
        )}

        {sourceId !== null && !loading && preview && (
          <Stack spacing={2} sx={{ pt: 1 }}>
            <Stack direction="row" spacing={1} alignItems="center">
              {renderContactCard(preview.target, 'keep')}
              <Tooltip title={t('contacts.merge.swap')}>
                <span>
                  <IconButton onClick={handleSwap} disabled={merging} aria-label={t('contacts.merge.swap')}>
                    <SwapHorizIcon />
                  </IconButton>
                </span>
              </Tooltip>
              {renderContactCard(preview.source, 'remove')}
            </Stack>

            {preview.conflicts.length > 0 ? (
              <Box>
                <Typography variant="subtitle2" sx={{ mb: 1 }}>
                  {t('contacts.merge.conflictsTitle')}
                </Typography>
                <Stack spacing={1.5}>
                  {preview.conflicts.map((conflict) => (
                    <Box key={conflict.field}>
                      <Typography variant="caption" color="text.secondary">
                        {fieldLabel(conflict.field)}
                      </Typography>
                      <RadioGroup
                        row
                        value={resolutions[conflict.field] ?? 'target'}
                        onChange={(e) =>
                          setResolutions((prev) => ({ ...prev, [conflict.field]: e.target.value as MergeResolution }))
                        }
                        sx={{ flexWrap: 'nowrap', gap: 1 }}
                      >
                        {(['target', 'source'] as MergeResolution[]).map((side) => (
                          <FormControlLabel
                            key={side}
                            value={side}
                            control={<Radio size="small" />}
                            label={renderValue(conflict, side)}
                            sx={{ flex: 1, minWidth: 0, mr: 0, alignItems: 'center' }}
                          />
                        ))}
                      </RadioGroup>
                    </Box>
                  ))}
                </Stack>
              </Box>
            ) : (
              <Typography variant="body2" color="text.secondary">
                {t('contacts.merge.noConflicts')}
              </Typography>
            )}

            <Box>
              <Typography variant="body2" color="text.secondary">
                {t('contacts.merge.combinedInfo')}
              </Typography>
              {movedSummary && (
                <Typography variant="body2" color="text.secondary">
                  {t('contacts.merge.movedInfo', { items: movedSummary })}
                </Typography>
              )}
            </Box>

            <Alert severity="warning">
              {t('contacts.merge.deleteWarning', { name: contactDisplayName(preview.source) })}
            </Alert>
          </Stack>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={merging}>
          {t('common.cancel')}
        </Button>
        <Button
          variant="contained"
          color="warning"
          onClick={handleMerge}
          disabled={!preview || loading || merging}
          startIcon={merging ? <CircularProgress size={16} color="inherit" /> : undefined}
        >
          {t('contacts.merge.confirm')}
        </Button>
      </DialogActions>
    </AppDialog>
  );
}
