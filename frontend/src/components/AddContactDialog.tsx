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
import {
  createContactRecord,
  ContactValue,
  ContactAddress,
  NameComponent,
  valuesToCardEmails,
  valuesToCardPhones,
  valuesToCardLinks,
  valuesToCardImpp,
  valuesToCardAddresses,
  withAnniversary,
  withOrganization,
  withTitles,
} from '../api/contacts';
import { Circle } from '../api/circles';
import { addCircleMember, createCircle } from '../api/circles';
import { createReminder } from '../api/reminders';
import { useSnackbar } from '../context/SnackbarContext';
import { handleError, getErrorMessage } from '../utils/errorHandler';
import { useDateFormat } from '../DateFormatProvider';
import { ContactFieldKey, resolveEnabledFields } from '../contactFields';

interface AddContactDialogProps {
  open: boolean;
  onClose: () => void;
  onContactAdded: (contactId: number) => void;
  availableCircles: Circle[];
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
  const [selectedCircles, setSelectedCircles] = useState<Circle[]>([]);
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
    if (trimmed && !selectedCircles.find(sc => sc.name === trimmed)) {
      setSelectedCircles([...selectedCircles, { id: '', created_at: '', updated_at: '', name: trimmed }]);
      setNewCircle('');
    }
  };

  const handleRemoveCircle = (circle: Circle) => {
    setSelectedCircles(selectedCircles.filter(c => c.id !== circle.id && c.name !== circle.name));
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

      const nameComponents: NameComponent[] = [];
      if (formData.prefix.trim()) nameComponents.push({ kind: 'title', value: formData.prefix.trim() });
      nameComponents.push({ kind: 'given', value: formData.firstname.trim() });
      if (formData.middle_name.trim()) nameComponents.push({ kind: 'given2', value: formData.middle_name.trim() });
      if (formData.lastname.trim()) nameComponents.push({ kind: 'surname', value: formData.lastname.trim() });
      if (formData.suffix.trim()) nameComponents.push({ kind: 'generation', value: formData.suffix.trim() });

      const cardEmails = valuesToCardEmails(emails);
      const cardPhones = valuesToCardPhones(phones);
      const cardLinks = valuesToCardLinks(urls);
      const cardImpp = valuesToCardImpp(impps);
      const cardAddresses = valuesToCardAddresses(addresses);
      const anniversaries = withAnniversary(withAnniversary(undefined, 'birth', birthdayISO), 'wedding', anniversaryISO);
      const organizations = withOrganization(formData.organization, formData.department);
      const titles = withTitles(formData.job_title, formData.role);

      const newRecord = await createContactRecord({
        gender: formData.gender,
        card: {
          name: { components: nameComponents },
          nicknames: formData.nickname.trim() ? [{ name: formData.nickname.trim() }] : undefined,
          emails: cardEmails.length > 0 ? cardEmails : undefined,
          phones: cardPhones.length > 0 ? cardPhones : undefined,
          links: cardLinks.length > 0 ? cardLinks : undefined,
          imppAddresses: cardImpp.length > 0 ? cardImpp : undefined,
          addresses: cardAddresses.length > 0 ? cardAddresses : undefined,
          anniversaries: anniversaries.length > 0 ? anniversaries : undefined,
          organizations: organizations.length > 0 ? organizations : undefined,
          titles: titles.length > 0 ? titles : undefined,
        },
        crm: {
          how_we_met: formData.how_we_met,
          work_information: formData.work_information,
          contact_information: formData.contact_information,
          custom_fields: Object.keys(filteredCustomFields).length > 0 ? filteredCustomFields : undefined,
        },
      });

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

          await createReminder(newRecord.id, {
            message: t('reminders.birthdayMessage', { name: `${formData.firstname} ${formData.lastname}` }),
            by_mail: true,
            remind_at: nextBirthday.toISOString(),
            recurrence: 'yearly',
            reoccur_from_completion: false,
            contact_id: newRecord.id
          });
        }
      }

      // Add circle memberships for selected circles
      for (const circle of selectedCircles) {
        try {
          let circleId = circle.id;
          if (!circleId) {
            const created = await createCircle(circle.name);
            circleId = created.circle.id;
          }
          await addCircleMember(circleId, newRecord.uid);
        } catch {
          // Silently skip — the contact exists, memberships are best-effort
        }
      }

      onContactAdded(newRecord.id);
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
          {isOn('links') && (
            <MultiValueField label={t('contacts.urls')} value={urls} onChange={setUrls} valueType="url" defaultType="home" />
          )}
          {isOn('imppAddresses') && (
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

          {isOn('organizations') && (
            <>
              <TextField label={t('contacts.organization')} fullWidth value={formData.organization} onChange={handleChange('organization')} />
              <TextField label={t('contacts.department')} fullWidth value={formData.department} onChange={handleChange('department')} />
            </>
          )}
          {isOn('titles') && (
            <>
              <TextField label={t('contacts.jobTitle')} fullWidth value={formData.job_title} onChange={handleChange('job_title')} />
              <TextField label={t('contacts.role')} fullWidth value={formData.role} onChange={handleChange('role')} />
            </>
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
              {selectedCircles.map(c => (
                <Chip
                  key={c.id || c.name}
                  label={c.name}
                  onDelete={() => handleRemoveCircle(c)}
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
                  if (value) {
                    const circle = availableCircles.find(c => c.name === value);
                    if (circle && !selectedCircles.find(sc => sc.id === circle.id || sc.name === circle.name)) {
                      setSelectedCircles([...selectedCircles, circle]);
                    }
                  }
                }}
              >
                <MenuItem value="">{t('contacts.selectCircle')}</MenuItem>
                {availableCircles
                  .filter(c => !selectedCircles.find(sc => sc.id === c.id || sc.name === c.name))
                  .map(c => (
                    <MenuItem key={c.id || c.name} value={c.name}>
                      {c.name}
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
