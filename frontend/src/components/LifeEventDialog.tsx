import { useState, useCallback, useEffect } from 'react';
import {
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  Box,
  MenuItem,
  Autocomplete,
  Chip,
  CircularProgress,
  FormControlLabel,
  Checkbox,
  Tooltip,
  Typography,
} from '@mui/material';
import AppDialog from './AppDialog';
import { useTranslation } from 'react-i18next';
import { Contact, getContacts, getContactsByUid } from '../api/contacts';
import {
  LIFE_EVENT_TYPES,
  LifeEventType,
  PartialDate,
  partialDateHasMonthDay,
  partialDateIsYearOnly,
} from '../api/lifeEvents';
import { handleFetchError } from '../utils/errorHandler';

interface ContactBrief {
  uid: string;
  firstname: string;
  lastname: string;
  nickname?: string;
}

function toContactBrief(c: Contact): ContactBrief {
  return { uid: c.uid!, firstname: c.firstname, lastname: c.lastname, nickname: c.nickname };
}

function contactLabel(c: ContactBrief): string {
  if (c.nickname) {
    return `${c.firstname} "${c.nickname}" ${c.lastname}`;
  }
  return `${c.firstname} ${c.lastname}`;
}

export interface LifeEventFormData {
  type: string;
  date?: PartialDate;
  description?: string;
  relatedEntityIds?: string[];
  remind?: boolean;
}

interface LifeEventDialogProps {
  open: boolean;
  onClose: () => void;
  onSave: (data: LifeEventFormData) => Promise<void>;
  initial?: LifeEventFormData;
  excludeContactUid?: string;
}

export default function LifeEventDialog({
  open,
  onClose,
  onSave,
  initial,
  excludeContactUid,
}: LifeEventDialogProps) {
  const { t } = useTranslation();
  const [type, setType] = useState<LifeEventType | ''>('');
  const [dateYear, setDateYear] = useState('');
  const [dateMonth, setDateMonth] = useState('');
  const [dateDay, setDateDay] = useState('');
  const [description, setDescription] = useState('');
  const [remind, setRemind] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const [contacts, setContacts] = useState<ContactBrief[]>([]);
  const [contactsLoading, setContactsLoading] = useState(false);
  const [searchInput, setSearchInput] = useState('');
  const [relatedContacts, setRelatedContacts] = useState<ContactBrief[]>([]);

  const isEditing = !!initial;

  const loadContacts = useCallback(
    async (search: string = '') => {
      setContactsLoading(true);
      try {
        const response = await getContacts({ limit: 40, search });
        let filtered = (response.contacts || [])
          .filter((c) => c.uid)
          .map(toContactBrief);
        if (excludeContactUid) {
          filtered = filtered.filter((c) => c.uid !== excludeContactUid);
        }
        setContacts(filtered);
      } catch (err) {
        handleFetchError(err, 'loading contacts');
      } finally {
        setContactsLoading(false);
      }
    },
    [excludeContactUid]
  );

  useEffect(() => {
    if (!open || !isEditing) return;
    const timeoutId = setTimeout(() => {
      loadContacts(searchInput);
    }, 300);
    return () => clearTimeout(timeoutId);
  }, [searchInput, open, isEditing, loadContacts]);

  useEffect(() => {
    if (open && !isEditing) {
      loadContacts();
    }
  }, [open, isEditing, loadContacts]);

  useEffect(() => {
    if (open) {
      if (initial) {
        setType((initial.type as LifeEventType) || '');
        if (initial.date) {
          setDateYear(initial.date.year != null ? String(initial.date.year) : '');
          setDateMonth(initial.date.month != null ? String(initial.date.month) : '');
          setDateDay(initial.date.day != null ? String(initial.date.day) : '');
        } else {
          setDateYear('');
          setDateMonth('');
          setDateDay('');
        }
        setDescription(initial.description || '');
        setRemind(initial.remind || false);
        setRelatedContacts([]);
        // Pre-populate related contacts from their VCardUIDs.
        if (initial.relatedEntityIds && initial.relatedEntityIds.length > 0) {
          getContactsByUid(initial.relatedEntityIds).then((byUid) => {
            const briefs: ContactBrief[] = [];
            byUid.forEach((c) => { if (c.uid) briefs.push(toContactBrief(c)); });
            setRelatedContacts(briefs);
          }).catch(() => {});
        }
      } else {
        setType('');
        setDateYear('');
        setDateMonth('');
        setDateDay('');
        setDescription('');
        setRemind(false);
        setRelatedContacts([]);
      }
      setError('');
      setSearchInput('');
    }
  }, [open, initial]);

  const currentDate: PartialDate | undefined = (() => {
    const y = dateYear ? parseInt(dateYear, 10) : undefined;
    const m = dateMonth ? parseInt(dateMonth, 10) : undefined;
    const d = dateDay ? parseInt(dateDay, 10) : undefined;
    if (y == null && m == null && d == null) return undefined;
    return { year: y, month: m, day: d };
  })();

  const canRemind = partialDateHasMonthDay(currentDate);
  const isYearOnly = partialDateIsYearOnly(currentDate);

  const handleSave = async () => {
    if (!type) {
      setError(t('lifeEvent.validation.typeRequired'));
      return;
    }

    setSaving(true);
    try {
      await onSave({
        type,
        date: currentDate,
        description: description || undefined,
        relatedEntityIds: relatedContacts.map((c) => c.uid),
        remind: canRemind ? remind : false,
      });
      handleClose();
    } catch {
      setError(t('lifeEvent.validation.saveFailed'));
    } finally {
      setSaving(false);
    }
  };

  const handleClose = () => {
    onClose();
  };

  return (
    <AppDialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
      <DialogTitle>
        {isEditing ? t('lifeEvent.editTitle') : t('lifeEvent.createTitle')}
      </DialogTitle>
      <DialogContent>
        <Box sx={{ pt: 1, display: 'flex', flexDirection: 'column', gap: 2 }}>
          <TextField
            select
            label={t('lifeEvent.type')}
            value={type}
            onChange={(e) => { setType(e.target.value as LifeEventType); setError(''); }}
            fullWidth
            required
          >
            {LIFE_EVENT_TYPES.map((tkn) => (
              <MenuItem key={tkn} value={tkn}>
                {t(`lifeEvent.types.${tkn}`, tkn)}
              </MenuItem>
            ))}
          </TextField>

          <Box display="flex" gap={1}>
            <TextField
              label={t('lifeEvent.dateYear')}
              type="number"
              value={dateYear}
              onChange={(e) => setDateYear(e.target.value)}
              fullWidth
              slotProps={{ inputLabel: { shrink: true }, htmlInput: { min: 1900, max: 2100 } }}
            />
            <TextField
              label={t('lifeEvent.dateMonth')}
              type="number"
              value={dateMonth}
              onChange={(e) => setDateMonth(e.target.value)}
              fullWidth
              slotProps={{ inputLabel: { shrink: true }, htmlInput: { min: 1, max: 12 } }}
            />
            <TextField
              label={t('lifeEvent.dateDay')}
              type="number"
              value={dateDay}
              onChange={(e) => setDateDay(e.target.value)}
              fullWidth
              slotProps={{ inputLabel: { shrink: true }, htmlInput: { min: 1, max: 31 } }}
            />
          </Box>

          <TextField
            label={t('lifeEvent.description')}
            multiline
            rows={3}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            fullWidth
          />

          <Autocomplete
            multiple
            options={contacts}
            getOptionLabel={contactLabel}
            value={relatedContacts}
            onChange={(_, value) => setRelatedContacts(value)}
            onInputChange={(_, value, reason) => {
              if (reason === 'input') setSearchInput(value);
            }}
            loading={contactsLoading}
            filterOptions={(x) => x}
            renderInput={(params) => (
              <TextField
                {...params}
                label={t('lifeEvent.relatedEntities')}
                placeholder={t('lifeEvent.searchContacts')}
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
            renderTags={(value, getTagProps) =>
              value.map((contact, index) => (
                <Chip
                  label={contactLabel(contact)}
                  {...getTagProps({ index })}
                  size="small"
                  key={contact.uid}
                />
              ))
            }
            isOptionEqualToValue={(option, value) => option.uid === value.uid}
            noOptionsText={searchInput ? t('lifeEvent.noContactsFound') : t('lifeEvent.typeToSearch')}
          />

          <Tooltip
            title={
              isYearOnly
                ? t('lifeEvent.remindDisabledYearOnly')
                : !currentDate
                  ? t('lifeEvent.remindDisabledNoDate')
                  : ''
            }
          >
            <FormControlLabel
              control={
                <Checkbox
                  checked={remind}
                  onChange={(e) => setRemind(e.target.checked)}
                  disabled={!canRemind}
                />
              }
              label={t('lifeEvent.remind')}
            />
          </Tooltip>

          {error && (
            <Typography color="error" variant="body2">
              {error}
            </Typography>
          )}
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose} disabled={saving}>
          {t('lifeEvent.cancel')}
        </Button>
        <Button onClick={handleSave} variant="contained" disabled={saving}>
          {t('lifeEvent.save')}
        </Button>
      </DialogActions>
    </AppDialog>
  );
}
