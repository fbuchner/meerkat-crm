import { useState, useEffect, useCallback } from 'react';
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
} from '@mui/material';
import AppDialog from './AppDialog';
import { useTranslation } from 'react-i18next';
import { Relationship, RelationshipFormData, RELATIONSHIP_TYPES } from '../api/relationships';
import { Contact, getContacts, createContact } from '../api/contacts';
import { useSnackbar } from '../context/SnackbarContext';
import { handleError, handleFetchError, getErrorMessage } from '../utils/errorHandler';
import { useDateFormat } from '../DateFormatProvider';

interface AddRelationshipDialogProps {
  open: boolean;
  onClose: () => void;
  onSave: (data: RelationshipFormData) => Promise<void>;
  relationship?: Relationship | null;
  currentContactId: number;
}

type EntryMode = 'manual' | 'linked' | 'new';

// Splits a free-text name into first/last name parts (everything after the
// first token becomes the last name)
function splitName(fullName: string): { firstname: string; lastname: string } {
  const [first = '', ...rest] = fullName.trim().split(/\s+/);
  return { firstname: first, lastname: rest.join(' ') };
}

export default function AddRelationshipDialog({
  open,
  onClose,
  onSave,
  relationship,
  currentContactId,
}: AddRelationshipDialogProps) {
  const { t } = useTranslation();
  const { showError, showSuccess } = useSnackbar();
  const { parseBirthdayInput, getBirthdayPlaceholder, formatBirthdayForInput, autoFormatBirthdayInput } = useDateFormat();
  const [entryMode, setEntryMode] = useState<EntryMode>('manual');
  const [name, setName] = useState('');
  const [firstname, setFirstname] = useState('');
  const [lastname, setLastname] = useState('');
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
  // Contact created by "create new contact" mode. Kept so a failed link step can
  // be retried without creating the contact twice.
  const [createdContact, setCreatedContact] = useState<Contact | null>(null);

  const loadContacts = useCallback(async (search: string = '') => {
    setContactsLoading(true);
    try {
      const response = await getContacts({ limit: 100, search });
      // Filter out the current contact
      const filteredContacts = response.contacts.filter(c => c.ID !== currentContactId);
      setContacts(filteredContacts);
    } catch (err) {
      handleFetchError(err, 'loading contacts');
    } finally {
      setContactsLoading(false);
    }
  }, [currentContactId]);

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
      const { firstname: first, lastname: last } = splitName(relationship.name || '');
      setFirstname(first);
      setLastname(last);
      setCreatedContact(null);
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
    setFirstname('');
    setLastname('');
    setType('');
    setCustomType('');
    setGender('');
    setBirthday('');
    setSelectedContact(null);
    setSearchInput('');
    setContacts([]);
    setError('');
    setCreatedContact(null);
  };

  const handleClose = () => {
    resetForm();
    onClose();
  };

  const handleModeChange = (mode: EntryMode) => {
    // Carry the typed name across between the two "type it in yourself" modes
    if (mode === 'new' && entryMode === 'manual' && name.trim() && !firstname.trim() && !lastname.trim()) {
      const { firstname: first, lastname: last } = splitName(name);
      setFirstname(first);
      setLastname(last);
    }
    if (mode === 'manual' && entryMode === 'new' && !name.trim()) {
      setName(`${firstname.trim()} ${lastname.trim()}`.trim());
    }
    setEntryMode(mode);
    setError('');
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
    if (entryMode === 'new' && !firstname.trim()) {
      setError(t('relationships.firstnameRequired'));
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
    if (entryMode !== 'linked' && birthday.trim()) {
      const parsed = parseBirthdayInput(birthday);
      if (parsed === null) {
        setError(t('contactDetail.birthdayError'));
        return;
      }
      birthdayISO = parsed || undefined;
    }

    setSaving(true);
    // Reused on retry so a failed link step doesn't create the contact twice
    let newContact: Contact | null = createdContact;
    try {
      if (entryMode === 'new' && !newContact) {
        newContact = await createContact({
          firstname: firstname.trim(),
          lastname: lastname.trim(),
          gender: gender || undefined,
          birthday: birthdayISO || undefined,
        });
        setCreatedContact(newContact);
        showSuccess(t('relationships.contactCreated', {
          name: `${newContact.firstname} ${newContact.lastname}`.trim(),
        }));
      }

      // The linked contact (existing or freshly created) owns name/gender/birthday,
      // so those stay on the contact rather than being duplicated on the relationship
      const linkedContact = entryMode === 'new' ? newContact : (entryMode === 'linked' ? selectedContact : null);
      const data: RelationshipFormData = {
        name: linkedContact
          ? `${linkedContact.firstname} ${linkedContact.lastname}`.trim()
          : name.trim(),
        type: effectiveType,
        // Only include gender/birthday for manual mode
        gender: entryMode === 'manual' ? (gender || undefined) : undefined,
        birthday: entryMode === 'manual' ? birthdayISO : undefined,
        related_contact_id: linkedContact ? linkedContact.ID : null,
      };
      await onSave(data);
      handleClose();
    } catch (err) {
      const operation = entryMode === 'new' && !newContact ? 'creating contact' : 'saving relationship';
      handleError(err, { operation }, { showError });
      const errorMessage = getErrorMessage(err);
      // The contact exists but the relationship didn't save - say so, otherwise
      // the user has no idea a contact was left behind
      setError(newContact && entryMode === 'new'
        ? `${t('relationships.contactCreatedLinkFailed')} ${errorMessage}`
        : errorMessage);
    } finally {
      setSaving(false);
    }
  };

  const isEditing = !!relationship;

  return (
    <AppDialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
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
                value="new"
                control={<Radio />}
                label={t('relationships.createNewContact')}
              />
              <FormControlLabel
                value="linked"
                control={<Radio />}
                label={t('relationships.linkToContact')}
              />
            </RadioGroup>
          </FormControl>

          {entryMode === 'new' && (
            <Typography variant="body2" color="text.secondary">
              {t('relationships.createNewContactHelp')}
            </Typography>
          )}

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

          {/* First/last name - the new contact is created from these */}
          {entryMode === 'new' && (
            <Box sx={{ display: 'flex', gap: 2 }}>
              <TextField
                label={t('contacts.firstname')}
                value={firstname}
                onChange={(e) => {
                  setFirstname(e.target.value);
                  setError('');
                }}
                fullWidth
                required
                disabled={!!createdContact}
                error={!!error && !firstname.trim()}
              />
              <TextField
                label={t('contacts.lastname')}
                value={lastname}
                onChange={(e) => {
                  setLastname(e.target.value);
                  setError('');
                }}
                fullWidth
                disabled={!!createdContact}
              />
            </Box>
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
              <MenuItem value="custom">{t('relationships.customType')}</MenuItem>
              {RELATIONSHIP_TYPES.map((relType) => (
                <MenuItem key={relType} value={relType}>
                  {t(`relationships.types.${relType.toLowerCase().replace(' ', '_')}`, relType)}
                </MenuItem>
              ))}
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

          {/* Gender - stored on the relationship (manual) or on the new contact */}
          {entryMode !== 'linked' && (
            <FormControl fullWidth disabled={!!createdContact}>
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

          {/* Birthday - stored on the relationship (manual) or on the new contact */}
          {entryMode !== 'linked' && (
            <TextField
              label={t('contacts.birthday')}
              value={birthday}
              onChange={(e) => setBirthday(autoFormatBirthdayInput(e.target.value, birthday))}
              placeholder={getBirthdayPlaceholder()}
              fullWidth
              disabled={!!createdContact}
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
    </AppDialog>
  );
}
