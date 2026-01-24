import { useState, useEffect, useCallback } from 'react';
import {
  Dialog,
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
} from '@mui/material';
import { useTranslation } from 'react-i18next';
import { Relationship, RelationshipFormData, RELATIONSHIP_TYPES } from '../api/relationships';
import { Contact, getContacts } from '../api/contacts';
import { useSnackbar } from '../context/SnackbarContext';
import { handleError, handleFetchError, getErrorMessage } from '../utils/errorHandler';
import { useDateFormat } from '../DateFormatProvider';

interface AddRelationshipDialogProps {
  open: boolean;
  onClose: () => void;
  onSave: (data: RelationshipFormData) => Promise<void>;
  relationship?: Relationship | null;
  token: string;
  currentContactId: number;
}

type EntryMode = 'manual' | 'linked';

export default function AddRelationshipDialog({
  open,
  onClose,
  onSave,
  relationship,
  token,
  currentContactId,
}: AddRelationshipDialogProps) {
  const { t } = useTranslation();
  const { showError } = useSnackbar();
  const { parseBirthdayInput, getBirthdayPlaceholder, formatBirthdayForInput } = useDateFormat();
  const [entryMode, setEntryMode] = useState<EntryMode>('manual');
  const [name, setName] = useState('');
  const [type, setType] = useState('');
  const [customType, setCustomType] = useState('');
  const [gender, setGender] = useState('');
  const [birthday, setBirthday] = useState('');
  const [selectedContact, setSelectedContact] = useState<Contact | null>(null);
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [contactsLoading, setContactsLoading] = useState(false);
  const [searchInput, setSearchInput] = useState('');
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);

  const loadContacts = useCallback(async (search: string = '') => {
    setContactsLoading(true);
    try {
      const response = await getContacts({ limit: 100, search }, token);
      // Filter out the current contact
      const filteredContacts = response.contacts.filter(c => c.ID !== currentContactId);
      setContacts(filteredContacts);
    } catch (err) {
      handleFetchError(err, 'loading contacts');
    } finally {
      setContactsLoading(false);
    }
  }, [token, currentContactId]);

  // Load contacts for linking
  useEffect(() => {
    if (open && entryMode === 'linked') {
      loadContacts();
    }
  }, [open, entryMode, loadContacts]);

  // Populate form when editing
  useEffect(() => {
    if (relationship) {
      setName(relationship.name || '');
      // Check if type is in presets
      if (RELATIONSHIP_TYPES.includes(relationship.type as typeof RELATIONSHIP_TYPES[number])) {
        setType(relationship.type);
        setCustomType('');
      } else {
        setType('custom');
        setCustomType(relationship.type || '');
      }
      setGender(relationship.gender || '');
      // Format birthday from ISO to display format based on user's date preferences
      setBirthday(relationship.birthday ? formatBirthdayForInput(relationship.birthday) : '');
      if (relationship.related_contact_id) {
        setEntryMode('linked');
        // We'll need to find the contact
        if (relationship.related_contact) {
          setSelectedContact({
            ID: relationship.related_contact.ID,
            firstname: relationship.related_contact.firstname,
            lastname: relationship.related_contact.lastname,
          } as Contact);
        }
      } else {
        setEntryMode('manual');
        setSelectedContact(null);
      }
    } else {
      resetForm();
    }
  }, [relationship, open, formatBirthdayForInput]);

  // Debounced search effect
  useEffect(() => {
    if (entryMode !== 'linked') return;
    
    const timeoutId = setTimeout(() => {
      loadContacts(searchInput);
    }, 300);

    return () => clearTimeout(timeoutId);
  }, [searchInput, entryMode, loadContacts]);

  const resetForm = () => {
    setEntryMode('manual');
    setName('');
    setType('');
    setCustomType('');
    setGender('');
    setBirthday('');
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
      // Load initial contacts when switching to linked mode
      loadContacts('');
    }
  };

  const handleContactSelect = (contact: Contact | null) => {
    setSelectedContact(contact);
    // Don't copy name/gender/birthday - they will be inferred from the linked contact
  };

  const getEffectiveType = () => {
    if (type === 'custom') {
      return customType.trim();
    }
    return type;
  };

  const handleSave = async () => {
    const effectiveType = getEffectiveType();
    
    // For manual mode, name is required
    if (entryMode === 'manual' && !name.trim()) {
      setError(t('relationships.nameRequired'));
      return;
    }
    if (!effectiveType) {
      setError(t('relationships.typeRequired'));
      return;
    }
    if (entryMode === 'linked' && !selectedContact) {
      setError(t('relationships.contactRequired'));
      return;
    }

    // Parse birthday from user's preferred format to ISO format
    let birthdayISO: string | undefined = undefined;
    if (entryMode === 'manual' && birthday.trim()) {
      const parsed = parseBirthdayInput(birthday);
      if (parsed === null) {
        setError(t('contactDetail.birthdayError'));
        return;
      }
      birthdayISO = parsed || undefined;
    }

    setSaving(true);
    try {
      // For linked mode, derive name from contact but don't store gender/birthday
      const data: RelationshipFormData = {
        name: entryMode === 'linked' && selectedContact 
          ? `${selectedContact.firstname} ${selectedContact.lastname}` 
          : name.trim(),
        type: effectiveType,
        // Only include gender/birthday for manual mode
        gender: entryMode === 'manual' ? (gender || undefined) : undefined,
        birthday: birthdayISO,
        related_contact_id: entryMode === 'linked' && selectedContact ? selectedContact.ID : null,
      };
      await onSave(data);
      handleClose();
    } catch (err) {
      handleError(err, { operation: 'saving relationship' }, { showError });
      const errorMessage = getErrorMessage(err);
      setError(errorMessage);
    } finally {
      setSaving(false);
    }
  };

  const isEditing = !!relationship;

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
      <DialogTitle>
        {isEditing ? t('relationships.editRelationship') : t('relationships.addRelationship')}
      </DialogTitle>
      <DialogContent>
        <Box sx={{ pt: 1, display: 'flex', flexDirection: 'column', gap: 2 }}>
          {/* Entry mode selection */}
          <FormControl component="fieldset">
            <FormLabel component="legend">{t('relationships.entryMode')}</FormLabel>
            <RadioGroup
              row
              value={entryMode}
              onChange={(e) => handleModeChange(e.target.value as EntryMode)}
            >
              <FormControlLabel
                value="manual"
                control={<Radio />}
                label={t('relationships.enterManually')}
              />
              <FormControlLabel
                value="linked"
                control={<Radio />}
                label={t('relationships.linkToContact')}
              />
            </RadioGroup>
          </FormControl>

          {/* Contact selector for linked mode */}
          {entryMode === 'linked' && (
            <Autocomplete
              options={contacts}
              getOptionLabel={(option) => `${option.firstname} ${option.lastname}`}
              value={selectedContact}
              onChange={(_, value) => handleContactSelect(value)}
              onInputChange={(_, value, reason) => {
                if (reason === 'input') {
                  setSearchInput(value);
                }
              }}
              loading={contactsLoading}
              filterOptions={(x) => x} // Disable client-side filtering, server handles it
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
              isOptionEqualToValue={(option, value) => option.ID === value.ID}
              noOptionsText={searchInput ? t('relationships.noContactsFound') : t('relationships.typeToSearch')}
            />
          )}

          {/* Name field - only shown for manual entry */}
          {entryMode === 'manual' && (
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

          {/* Relationship type */}
          <FormControl fullWidth required>
            <InputLabel>{t('relationships.type')}</InputLabel>
            <Select
              value={type}
              label={t('relationships.type')}
              onChange={(e) => {
                setType(e.target.value);
                setError('');
              }}
            >
              {RELATIONSHIP_TYPES.map((relType) => (
                <MenuItem key={relType} value={relType}>
                  {t(`relationships.types.${relType.toLowerCase().replace(' ', '_')}`, relType)}
                </MenuItem>
              ))}
              <MenuItem value="custom">{t('relationships.customType')}</MenuItem>
            </Select>
          </FormControl>

          {/* Custom type input */}
          {type === 'custom' && (
            <TextField
              label={t('relationships.customTypeLabel')}
              value={customType}
              onChange={(e) => {
                setCustomType(e.target.value);
                setError('');
              }}
              fullWidth
              required
              autoFocus
            />
          )}

          {/* Gender - only shown for manual entry */}
          {entryMode === 'manual' && (
            <FormControl fullWidth>
              <InputLabel>{t('contacts.gender')}</InputLabel>
              <Select
                value={gender}
                label={t('contacts.gender')}
                onChange={(e) => setGender(e.target.value)}
              >
                <MenuItem value="">{t('contacts.selectGender')}</MenuItem>
                <MenuItem value="male">{t('contacts.male')}</MenuItem>
                <MenuItem value="female">{t('contacts.female')}</MenuItem>
                <MenuItem value="other">{t('contacts.other')}</MenuItem>
              </Select>
            </FormControl>
          )}

          {/* Birthday - only shown for manual entry */}
          {entryMode === 'manual' && (
            <TextField
              label={t('contacts.birthday')}
              value={birthday}
              onChange={(e) => setBirthday(e.target.value)}
              placeholder={getBirthdayPlaceholder()}
              fullWidth
              helperText={t('contacts.birthdayFormat')}
            />
          )}

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
    </Dialog>
  );
}
