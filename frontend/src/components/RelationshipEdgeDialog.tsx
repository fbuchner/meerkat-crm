import { useState, useEffect, useCallback, useMemo } from 'react';
import {
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  Box,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  RadioGroup,
  FormControlLabel,
  Radio,
  Autocomplete,
  FormLabel,
  CircularProgress,
  Typography,
  Link,
} from '@mui/material';
import AppDialog from './AppDialog';
import { useTranslation } from 'react-i18next';
import {
  RelationshipEdge,
  RelationshipEdgeInput,
  RelationshipEdgeType,
  RelationshipEdgeSensitivity,
  RELATIONSHIP_EDGE_TYPE_TOKENS,
  getEffectiveType,
  toBackendType,
} from '../api/relationshipEdges';
import { Contact, getContacts } from '../api/contacts';
import { useSnackbar } from '../context/SnackbarContext';
import { handleError, handleFetchError, getErrorMessage } from '../utils/errorHandler';
import { useDateFormat } from '../DateFormatProvider';

interface RelationshipEdgeDialogProps {
  open: boolean;
  onClose: () => void;
  onSave: (input: RelationshipEdgeInput) => Promise<void>;
  edge?: RelationshipEdge | null; // undefined/null = create mode
  viewedContactUid: string; // the contact whose detail page this is opened from
  otherPartyContact?: Contact; // resolved other-party Contact; edit mode only
}

type EntryMode = 'manual' | 'linked';

export default function RelationshipEdgeDialog({
  open,
  onClose,
  onSave,
  edge,
  viewedContactUid,
  otherPartyContact,
}: RelationshipEdgeDialogProps) {
  const { t } = useTranslation();
  const { showError } = useSnackbar();
  const { parseBirthdayInput, getBirthdayPlaceholder, autoFormatBirthdayInput } = useDateFormat();
  const [entryMode, setEntryMode] = useState<EntryMode>('manual');
  const [name, setName] = useState('');
  const [type, setType] = useState<RelationshipEdgeType | ''>('');
  const [gender, setGender] = useState('');
  const [birthday, setBirthday] = useState('');
  const [sensitivity, setSensitivity] = useState<RelationshipEdgeSensitivity>('normal');
  const [selectedContact, setSelectedContact] = useState<Contact | null>(null);
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [contactsLoading, setContactsLoading] = useState(false);
  const [searchInput, setSearchInput] = useState('');
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);

  const isEditing = !!edge;
  // Only meaningful in edit mode: which side of the edge the viewed contact
  // occupies, needed to convert the dropdown's "what is the other party to
  // me" selection back into the backend's Source-relative `type`.
  const viewedIsSource = edge ? edge.source_id === viewedContactUid : false;

  const loadContacts = useCallback(async (search: string = '') => {
    setContactsLoading(true);
    try {
      const response = await getContacts({ limit: 100, search });
      const filteredContacts = response.contacts.filter((c) => c.uid !== viewedContactUid);
      setContacts(filteredContacts);
    } catch (err) {
      handleFetchError(err, 'loading contacts');
    } finally {
      setContactsLoading(false);
    }
  }, [viewedContactUid]);

  useEffect(() => {
    if (open && entryMode === 'linked' && !isEditing) {
      loadContacts();
    }
  }, [open, entryMode, isEditing, loadContacts]);

  // Populate form when editing.
  useEffect(() => {
    if (edge) {
      setType(getEffectiveType(edge, viewedContactUid));
      setSensitivity(edge.sensitivity);
      setEntryMode('manual');
      setSelectedContact(null);
      setName('');
      setGender('');
      setBirthday('');
    } else {
      resetForm();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [edge, open]);

  // Debounced search effect (create + linked mode only).
  useEffect(() => {
    if (isEditing || entryMode !== 'linked') return;
    const timeoutId = setTimeout(() => {
      loadContacts(searchInput);
    }, 300);
    return () => clearTimeout(timeoutId);
  }, [searchInput, entryMode, isEditing, loadContacts]);

  const resetForm = () => {
    setEntryMode('manual');
    setName('');
    setType('');
    setGender('');
    setBirthday('');
    setSensitivity('normal');
    setSelectedContact(null);
    setSearchInput('');
    setContacts([]);
    setError('');
  };

  const handleClose = () => {
    resetForm();
    onClose();
  };

  const handleModeChange = (mode: EntryMode) => {
    setEntryMode(mode);
    if (mode === 'linked') {
      loadContacts('');
    }
  };

  const handleContactSelect = (contact: Contact | null) => {
    setSelectedContact(contact);
    // Don't copy name/gender/birthday - not stored on the edge, only used to
    // resolve source_id/target_id.
  };

  const otherPartyLabel = useMemo(() => {
    if (!otherPartyContact) return '';
    return `${otherPartyContact.firstname} ${otherPartyContact.lastname}`.trim();
  }, [otherPartyContact]);

  const handleSave = async () => {
    if (!type) {
      setError(t('relationships.typeRequired'));
      return;
    }
    if (!isEditing && entryMode === 'manual' && !name.trim()) {
      setError(t('relationships.nameRequired'));
      return;
    }
    if (!isEditing && entryMode === 'linked' && !selectedContact) {
      setError(t('relationships.contactRequired'));
      return;
    }

    let birthdayISO: string | undefined;
    if (!isEditing && entryMode === 'manual' && birthday.trim()) {
      const parsed = parseBirthdayInput(birthday);
      if (parsed === null) {
        setError(t('contactDetail.birthdayError'));
        return;
      }
      birthdayISO = parsed || undefined;
    }

    setSaving(true);
    try {
      let input: RelationshipEdgeInput;
      if (isEditing && edge) {
        // Edit mode: always resend the edge's own already-resolved
        // source_id/target_id verbatim, never *_thin -- resolveRelationshipEndpoint
        // (backend) always inserts a new Contact for a non-nil *_thin input,
        // even on update, so resubmitting manual-entry fields here would
        // silently orphan a new thin Contact on every edit. The other
        // party's identity is read-only in this dialog for that reason.
        input = {
          source_id: edge.source_id,
          target_id: edge.target_id,
          type: toBackendType(type as RelationshipEdgeType, viewedIsSource),
          sensitivity,
          metadata: edge.metadata,
        };
      } else if (entryMode === 'linked' && selectedContact?.uid) {
        input = {
          target_id: viewedContactUid,
          source_id: selectedContact.uid,
          type: type as RelationshipEdgeType,
          sensitivity,
        };
      } else {
        input = {
          target_id: viewedContactUid,
          source_thin: { name: name.trim(), gender: gender || undefined, birthday: birthdayISO },
          type: type as RelationshipEdgeType,
          sensitivity,
        };
      }
      await onSave(input);
      handleClose();
    } catch (err) {
      handleError(err, { operation: 'saving relationship' }, { showError });
      setError(getErrorMessage(err));
    } finally {
      setSaving(false);
    }
  };

  return (
    <AppDialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
      <DialogTitle>
        {isEditing ? t('relationships.editRelationship') : t('relationships.addRelationship')}
      </DialogTitle>
      <DialogContent>
        <Box sx={{ pt: 1, display: 'flex', flexDirection: 'column', gap: 2 }}>
          {!isEditing && (
            <FormControl component="fieldset">
              <FormLabel component="legend">{t('relationships.entryMode')}</FormLabel>
              <RadioGroup row value={entryMode} onChange={(e) => handleModeChange(e.target.value as EntryMode)}>
                <FormControlLabel value="manual" control={<Radio />} label={t('relationships.enterManually')} />
                <FormControlLabel value="linked" control={<Radio />} label={t('relationships.linkToContact')} />
              </RadioGroup>
            </FormControl>
          )}

          {isEditing && (
            <Box>
              <Typography variant="body2" color="text.secondary">
                {otherPartyContact ? (
                  <Link href={`/contacts/${otherPartyContact.ID}`}>{otherPartyLabel}</Link>
                ) : (
                  t('relationships.unknownContact')
                )}
              </Typography>
              <Typography variant="caption" color="text.secondary">
                {t('relationships.otherPartyReadOnlyHint')}
              </Typography>
            </Box>
          )}

          {!isEditing && entryMode === 'linked' && (
            <Autocomplete
              options={contacts}
              getOptionLabel={(option) => `${option.firstname} ${option.lastname}`}
              value={selectedContact}
              onChange={(_, value) => handleContactSelect(value)}
              onInputChange={(_, value, reason) => {
                if (reason === 'input') setSearchInput(value);
              }}
              loading={contactsLoading}
              filterOptions={(x) => x}
              renderInput={(params) => (
                <TextField
                  {...params}
                  label={t('relationships.selectContact')}
                  placeholder={t('relationships.searchContacts')}
                  required
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
              noOptionsText={searchInput ? t('relationships.noContactsFound') : t('relationships.typeToSearch')}
            />
          )}

          {!isEditing && entryMode === 'manual' && (
            <TextField
              label={t('relationships.name')}
              value={name}
              onChange={(e) => {
                setName(e.target.value);
                setError('');
              }}
              fullWidth
              required
              error={!!error && !name.trim()}
            />
          )}

          <FormControl fullWidth required>
            <InputLabel>{t('relationships.type')}</InputLabel>
            <Select
              value={type}
              label={t('relationships.type')}
              onChange={(e) => {
                setType(e.target.value as RelationshipEdgeType);
                setError('');
              }}
            >
              {RELATIONSHIP_EDGE_TYPE_TOKENS.map((token) => (
                <MenuItem key={token} value={token}>
                  {t(`relationships.types.${token}`)}
                </MenuItem>
              ))}
            </Select>
          </FormControl>

          {!isEditing && entryMode === 'manual' && (
            <FormControl fullWidth>
              <InputLabel>{t('contacts.gender')}</InputLabel>
              <Select value={gender} label={t('contacts.gender')} onChange={(e) => setGender(e.target.value)}>
                <MenuItem value="">{t('contacts.selectGender')}</MenuItem>
                <MenuItem value="male">{t('contacts.male')}</MenuItem>
                <MenuItem value="female">{t('contacts.female')}</MenuItem>
                <MenuItem value="other">{t('contacts.other')}</MenuItem>
              </Select>
            </FormControl>
          )}

          {!isEditing && entryMode === 'manual' && (
            <TextField
              label={t('contacts.birthday')}
              value={birthday}
              onChange={(e) => setBirthday(autoFormatBirthdayInput(e.target.value, birthday))}
              placeholder={getBirthdayPlaceholder()}
              fullWidth
              helperText={t('contacts.birthdayFormat')}
            />
          )}

          <FormControl fullWidth>
            <InputLabel>{t('relationships.sensitivity')}</InputLabel>
            <Select
              value={sensitivity}
              label={t('relationships.sensitivity')}
              onChange={(e) => setSensitivity(e.target.value as RelationshipEdgeSensitivity)}
            >
              <MenuItem value="normal">{t('relationships.sensitivityNormal')}</MenuItem>
              <MenuItem value="private">{t('relationships.sensitivityPrivate')}</MenuItem>
              <MenuItem value="secret">{t('relationships.sensitivitySecret')}</MenuItem>
            </Select>
          </FormControl>

          {error && (
            <Box sx={{ color: 'error.main', fontSize: '0.875rem' }}>
              {error}
            </Box>
          )}
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose} disabled={saving}>
          {t('common.cancel')}
        </Button>
        <Button onClick={handleSave} variant="contained" disabled={saving}>
          {saving ? t('common.saving') : t('common.save')}
        </Button>
      </DialogActions>
    </AppDialog>
  );
}
