import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  MenuItem,
  Chip,
  Box,
  Typography,
  Stack,
  FormControlLabel,
  Switch
} from '@mui/material';
import AppDialog from './AppDialog';
import MultiValueField from './MultiValueField';
import AddressFields from './AddressFields';
import { createContact, ContactValue, ContactAddress } from '../api/contacts';
import { createReminder } from '../api/reminders';
import { useSnackbar } from '../context/SnackbarContext';
import { handleError, getErrorMessage } from '../utils/errorHandler';
import { useDateFormat } from '../DateFormatProvider';
import { ContactFieldKey, resolveEnabledFields } from '../contactFields';

interface AddContactDialogProps {
  open: boolean;
  onClose: () => void;
  onContactAdded: (contactId: number) => void;
  availableCircles: string[];
  customFieldNames?: string[];
  enabledFields?: Set<ContactFieldKey>;
}

const emptyForm = {
  firstname: '',
  lastname: '',
  prefix: '',
  middle_name: '',
  suffix: '',
  nickname: '',
  gender: '',
  birthday: '',
  anniversary: '',
  organization: '',
  department: '',
  job_title: '',
  role: '',
  how_we_met: '',
  food_preference: '',
  work_information: '',
  contact_information: ''
};

export default function AddContactDialog({
  open,
  onClose,
  onContactAdded,
  availableCircles,
  customFieldNames = [],
  enabledFields
}: AddContactDialogProps) {
  const { t } = useTranslation();
  const { showError, showSuccess } = useSnackbar();
  const { parseBirthdayInput, getBirthdayPlaceholder, autoFormatBirthdayInput } = useDateFormat();
  const enabled = enabledFields ?? resolveEnabledFields(null);
  const isOn = (key: ContactFieldKey) => enabled.has(key);

  const [formData, setFormData] = useState({ ...emptyForm });
  const [emails, setEmails] = useState<ContactValue[]>([]);
  const [phones, setPhones] = useState<ContactValue[]>([]);
  const [addresses, setAddresses] = useState<ContactAddress[]>([]);
  const [urls, setUrls] = useState<ContactValue[]>([]);
  const [impps, setImpps] = useState<ContactValue[]>([]);
  const [customFieldValues, setCustomFieldValues] = useState<Record<string, string>>({});
  const [selectedCircles, setSelectedCircles] = useState<string[]>([]);
  const [newCircle, setNewCircle] = useState('');
  const [createBirthdayReminder, setCreateBirthdayReminder] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleChange = (field: string) => (event: React.ChangeEvent<HTMLInputElement>) => {
    if (field === 'birthday') {
      setFormData({ ...formData, birthday: autoFormatBirthdayInput(event.target.value, formData.birthday) });
    } else if (field === 'anniversary') {
      setFormData({ ...formData, anniversary: autoFormatBirthdayInput(event.target.value, formData.anniversary) });
    } else {
      setFormData({ ...formData, [field]: event.target.value });
    }
  };

  const handleCustomFieldChange = (fieldName: string) => (event: React.ChangeEvent<HTMLInputElement>) => {
    setCustomFieldValues({ ...customFieldValues, [fieldName]: event.target.value });
  };

  const handleAddCircle = () => {
    const trimmed = newCircle.trim();
    if (trimmed && !selectedCircles.includes(trimmed)) {
      setSelectedCircles([...selectedCircles, trimmed]);
      setNewCircle('');
    }
  };

  const handleRemoveCircle = (circle: string) => {
    setSelectedCircles(selectedCircles.filter(c => c !== circle));
  };

  const handleSubmit = async () => {
    if (!formData.firstname.trim()) {
      setError(t('contacts.add.requiredFields'));
      return;
    }

    // Parse birthday from user's preferred format to ISO format
    let birthdayISO = '';
    if (formData.birthday.trim()) {
      const parsed = parseBirthdayInput(formData.birthday);
      if (parsed === null) {
        setError(t('contactDetail.birthdayError'));
        return;
      }
      birthdayISO = parsed;
    }

    let anniversaryISO = '';
    if (formData.anniversary.trim()) {
      const parsed = parseBirthdayInput(formData.anniversary);
      if (parsed === null) {
        setError(t('contactDetail.birthdayError'));
        return;
      }
      anniversaryISO = parsed;
    }

    setLoading(true);
    setError('');

    try {
      const filteredCustomFields: Record<string, string> = {};
      for (const [key, value] of Object.entries(customFieldValues)) {
        if (value.trim()) {
          filteredCustomFields[key] = value;
        }
      }

      // Drop empty rows from the multi-value fields
      const cleanEmails = emails.filter(e => e.value.trim());
      const cleanPhones = phones.filter(p => p.value.trim());
      const cleanUrls = urls.filter(u => u.value.trim());
      const cleanImpps = impps.filter(i => i.value.trim());
      const cleanAddresses = addresses.filter(
        a => a.street.trim() || a.city.trim() || a.region.trim() || a.postal.trim() || a.country.trim()
      );

      const contactData = {
        firstname: formData.firstname,
        lastname: formData.lastname,
        prefix: formData.prefix,
        middle_name: formData.middle_name,
        suffix: formData.suffix,
        nickname: formData.nickname,
        gender: formData.gender,
        birthday: birthdayISO,
        anniversary: anniversaryISO,
        organization: formData.organization,
        department: formData.department,
        job_title: formData.job_title,
        role: formData.role,
        how_we_met: formData.how_we_met,
        food_preference: formData.food_preference,
        work_information: formData.work_information,
        contact_information: formData.contact_information,
        emails: cleanEmails,
        phones: cleanPhones,
        addresses: cleanAddresses,
        urls: cleanUrls,
        impps: cleanImpps,
        // Derived primary scalars keep search/list and the backend in sync
        email: cleanEmails[0]?.value || '',
        phone: cleanPhones[0]?.value || '',
        circles: selectedCircles.length > 0 ? selectedCircles : undefined,
        custom_fields: Object.keys(filteredCustomFields).length > 0 ? filteredCustomFields : undefined
      };

      const newContact = await createContact(contactData);

      if (createBirthdayReminder && birthdayISO) {
        let day: number | undefined;
        let month: number | undefined;

        if (birthdayISO.startsWith('--')) {
          month = parseInt(birthdayISO.substring(2, 4), 10) - 1;
          day = parseInt(birthdayISO.substring(5, 7), 10);
        } else {
          const parts = birthdayISO.split('-');
          if (parts.length === 3) {
            month = parseInt(parts[1], 10) - 1;
            day = parseInt(parts[2], 10);
          }
        }

        if (day !== undefined && month !== undefined && !isNaN(day) && !isNaN(month)) {
          const today = new Date();
          let nextBirthday = new Date(today.getFullYear(), month, day);

          if (nextBirthday < today) {
            nextBirthday.setFullYear(today.getFullYear() + 1);
          }

          nextBirthday.setHours(9, 0, 0, 0);

          await createReminder(newContact.ID, {
            message: t('reminders.birthdayMessage', { name: `${newContact.firstname} ${newContact.lastname}` }),
            by_mail: true,
            remind_at: nextBirthday.toISOString(),
            recurrence: 'yearly',
            reoccur_from_completion: false,
            contact_id: newContact.ID
          });
        }
      }

      onContactAdded(newContact.ID);
      showSuccess(t('contacts.add.success'));
      handleClose();
    } catch (err) {
      handleError(err, { operation: 'creating contact' }, { showError });
      const errorMessage = getErrorMessage(err);
      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    setFormData({ ...emptyForm });
    setEmails([]);
    setPhones([]);
    setAddresses([]);
    setUrls([]);
    setImpps([]);
    setCustomFieldValues({});
    setSelectedCircles([]);
    setNewCircle('');
    setCreateBirthdayReminder(false);
    setError('');
    onClose();
  };

  return (
    <AppDialog open={open} onClose={handleClose} maxWidth="md" fullWidth>
      <DialogTitle>{t('contacts.add.title')}</DialogTitle>
      <DialogContent>
        {error && (
          <Typography color="error" sx={{ mb: 2 }}>
            {error}
          </Typography>
        )}
        <Stack spacing={2} sx={{ mt: 1 }}>
          {(isOn('prefix') || isOn('suffix')) && (
            <Stack direction="row" spacing={2}>
              {isOn('prefix') && (
                <TextField label={t('contacts.prefix')} fullWidth value={formData.prefix} onChange={handleChange('prefix')} />
              )}
              {isOn('suffix') && (
                <TextField label={t('contacts.suffix')} fullWidth value={formData.suffix} onChange={handleChange('suffix')} />
              )}
            </Stack>
          )}
          <Stack direction="row" spacing={2}>
            <TextField
              label={t('contacts.firstname')}
              fullWidth
              value={formData.firstname}
              onChange={handleChange('firstname')}
              required
            />
            <TextField
              label={t('contacts.lastname')}
              fullWidth
              value={formData.lastname}
              onChange={handleChange('lastname')}
            />
          </Stack>
          {isOn('middle_name') && (
            <TextField label={t('contacts.middleName')} fullWidth value={formData.middle_name} onChange={handleChange('middle_name')} />
          )}
          <Stack direction="row" spacing={2}>
            {isOn('nickname') && (
              <TextField
                label={t('contacts.nickname')}
                fullWidth
                value={formData.nickname}
                onChange={handleChange('nickname')}
              />
            )}
            {isOn('gender') && (
              <TextField
                select
                label={t('contacts.gender')}
                fullWidth
                value={formData.gender}
                onChange={handleChange('gender')}
              >
                <MenuItem value="">{t('contacts.selectGender')}</MenuItem>
                <MenuItem value="male">{t('contacts.male')}</MenuItem>
                <MenuItem value="female">{t('contacts.female')}</MenuItem>
                <MenuItem value="other">{t('contacts.other')}</MenuItem>
              </TextField>
            )}
          </Stack>

          {isOn('emails') && (
            <MultiValueField label={t('contacts.email')} value={emails} onChange={setEmails} valueType="email" defaultType="home" />
          )}
          {isOn('phones') && (
            <MultiValueField label={t('contacts.phone')} value={phones} onChange={setPhones} valueType="tel" defaultType="cell" />
          )}
          {isOn('addresses') && (
            <AddressFields label={t('contacts.address')} value={addresses} onChange={setAddresses} />
          )}
          {isOn('urls') && (
            <MultiValueField label={t('contacts.urls')} value={urls} onChange={setUrls} valueType="url" defaultType="home" />
          )}
          {isOn('impps') && (
            <MultiValueField label={t('contacts.impps')} value={impps} onChange={setImpps} defaultType="" freeTextType />
          )}

          {isOn('birthday') && (
            <>
              <TextField
                label={t('contacts.birthday')}
                fullWidth
                value={formData.birthday}
                onChange={handleChange('birthday')}
                placeholder={getBirthdayPlaceholder()}
                helperText={t('contacts.birthdayFormat')}
              />
              {formData.birthday && (
                <FormControlLabel
                  control={
                    <Switch
                      checked={createBirthdayReminder}
                      onChange={(e) => setCreateBirthdayReminder(e.target.checked)}
                    />
                  }
                  label={t('contacts.add.createBirthdayReminder')}
                />
              )}
            </>
          )}
          {isOn('anniversary') && (
            <TextField
              label={t('contacts.anniversary')}
              fullWidth
              value={formData.anniversary}
              onChange={handleChange('anniversary')}
              placeholder={getBirthdayPlaceholder()}
              helperText={t('contacts.birthdayFormat')}
            />
          )}

          {isOn('organization') && (
            <TextField label={t('contacts.organization')} fullWidth value={formData.organization} onChange={handleChange('organization')} />
          )}
          {isOn('department') && (
            <TextField label={t('contacts.department')} fullWidth value={formData.department} onChange={handleChange('department')} />
          )}
          {isOn('job_title') && (
            <TextField label={t('contacts.jobTitle')} fullWidth value={formData.job_title} onChange={handleChange('job_title')} />
          )}
          {isOn('role') && (
            <TextField label={t('contacts.role')} fullWidth value={formData.role} onChange={handleChange('role')} />
          )}
          {isOn('work_information') && (
            <TextField
              label={t('contacts.workInformation')}
              fullWidth
              multiline
              rows={2}
              value={formData.work_information}
              onChange={handleChange('work_information')}
            />
          )}

          {isOn('how_we_met') && (
            <TextField
              label={t('contacts.howWeMet')}
              fullWidth
              multiline
              rows={2}
              value={formData.how_we_met}
              onChange={handleChange('how_we_met')}
            />
          )}
          {isOn('food_preference') && (
            <TextField
              label={t('contacts.foodPreference')}
              fullWidth
              value={formData.food_preference}
              onChange={handleChange('food_preference')}
            />
          )}
          {isOn('contact_information') && (
            <TextField
              label={t('contacts.contactInformation')}
              fullWidth
              multiline
              rows={2}
              value={formData.contact_information}
              onChange={handleChange('contact_information')}
            />
          )}

          {/* Custom Fields */}
          {customFieldNames.map((fieldName) => (
            <TextField
              key={fieldName}
              label={fieldName}
              fullWidth
              multiline
              rows={2}
              value={customFieldValues[fieldName] || ''}
              onChange={handleCustomFieldChange(fieldName)}
            />
          ))}
          <Box>
            <Typography variant="subtitle2" gutterBottom>
              {t('contacts.circles')}
            </Typography>
            <Box sx={{ display: 'flex', gap: 1, mb: 1, flexWrap: 'wrap' }}>
              {selectedCircles.map(circle => (
                <Chip
                  key={circle}
                  label={circle}
                  onDelete={() => handleRemoveCircle(circle)}
                  color="primary"
                  size="small"
                />
              ))}
            </Box>
            <Stack direction="row" spacing={1}>
              <TextField
                select
                label={t('contacts.selectCircle')}
                fullWidth
                value=""
                onChange={(e) => {
                  const value = e.target.value;
                  if (value && !selectedCircles.includes(value)) {
                    setSelectedCircles([...selectedCircles, value]);
                  }
                }}
              >
                <MenuItem value="">{t('contacts.selectCircle')}</MenuItem>
                {availableCircles
                  .filter(c => !selectedCircles.includes(c))
                  .map(circle => (
                    <MenuItem key={circle} value={circle}>
                      {circle}
                    </MenuItem>
                  ))}
              </TextField>
              <TextField
                label={t('contacts.newCircle')}
                value={newCircle}
                onChange={(e) => setNewCircle(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault();
                    handleAddCircle();
                  }
                }}
                sx={{ minWidth: 150 }}
              />
              <Button onClick={handleAddCircle} variant="outlined">
                {t('contacts.add.addCircle')}
              </Button>
            </Stack>
          </Box>
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose} disabled={loading}>
          {t('common.cancel')}
        </Button>
        <Button onClick={handleSubmit} variant="contained" disabled={loading}>
          {loading ? t('common.saving') : t('contacts.add.create')}
        </Button>
      </DialogActions>
    </AppDialog>
  );
}
