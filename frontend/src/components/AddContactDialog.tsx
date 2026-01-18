import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Dialog,
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
import { createContact } from '../api/contacts';
import { createReminder } from '../api/reminders';
import { useSnackbar } from '../context/SnackbarContext';
import { handleError, getErrorMessage } from '../utils/errorHandler';
import { useDateFormat } from '../DateFormatProvider';

interface AddContactDialogProps {
  open: boolean;
  onClose: () => void;
  onContactAdded: (contactId: number) => void;
  token: string;
  availableCircles: string[];
}

export default function AddContactDialog({
  open,
  onClose,
  onContactAdded,
  token,
  availableCircles
}: AddContactDialogProps) {
  const { t } = useTranslation();
  const { showError, showSuccess } = useSnackbar();
  const { parseBirthdayInput, getBirthdayPlaceholder } = useDateFormat();
  const [formData, setFormData] = useState({
    firstname: '',
    lastname: '',
    nickname: '',
    gender: '',
    email: '',
    phone: '',
    birthday: '',
    address: '',
    how_we_met: '',
    food_preference: '',
    work_information: '',
    contact_information: ''
  });
  const [selectedCircles, setSelectedCircles] = useState<string[]>([]);
  const [newCircle, setNewCircle] = useState('');
  const [createBirthdayReminder, setCreateBirthdayReminder] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleChange = (field: string) => (event: React.ChangeEvent<HTMLInputElement>) => {
    setFormData({ ...formData, [field]: event.target.value });
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
    // Validate required fields
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

    setLoading(true);
    setError('');

    try {
      const contactData = {
        ...formData,
        birthday: birthdayISO,
        circles: selectedCircles.length > 0 ? selectedCircles : undefined
      };

      const newContact = await createContact(contactData, token);

      if (createBirthdayReminder && birthdayISO) {
        // Birthday format is now YYYY-MM-DD or --MM-DD (ISO 8601)
        let day: number | undefined;
        let month: number | undefined;

        if (birthdayISO.startsWith('--')) {
          // --MM-DD format
          month = parseInt(birthdayISO.substring(2, 4), 10) - 1;
          day = parseInt(birthdayISO.substring(5, 7), 10);
        } else {
          // YYYY-MM-DD format
          const parts = birthdayISO.split('-');
          if (parts.length === 3) {
            month = parseInt(parts[1], 10) - 1;
            day = parseInt(parts[2], 10);
          }
        }

        if (day !== undefined && month !== undefined && !isNaN(day) && !isNaN(month)) {
          const today = new Date();
          let nextBirthday = new Date(today.getFullYear(), month, day);

          // If birthday has passed this year, set for next year
          if (nextBirthday < today) {
            nextBirthday.setFullYear(today.getFullYear() + 1);
          }

          // Set reminder for 9 AM
          nextBirthday.setHours(9, 0, 0, 0);

          await createReminder(newContact.ID, {
            message: t('reminders.birthdayMessage', { name: `${newContact.firstname} ${newContact.lastname}` }),
            by_mail: true,
            remind_at: nextBirthday.toISOString(),
            recurrence: 'yearly',
            reoccur_from_completion: false,
            contact_id: newContact.ID
          }, token);
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
    setFormData({
      firstname: '',
      lastname: '',
      nickname: '',
      gender: '',
      email: '',
      phone: '',
      birthday: '',
      address: '',
      how_we_met: '',
      food_preference: '',
      work_information: '',
      contact_information: ''
    });
    setSelectedCircles([]);
    setNewCircle('');
    setError('');
    onClose();
  };

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="md" fullWidth>
      <DialogTitle>{t('contacts.add.title')}</DialogTitle>
      <DialogContent>
        {error && (
          <Typography color="error" sx={{ mb: 2 }}>
            {error}
          </Typography>
        )}
        <Stack spacing={2} sx={{ mt: 1 }}>
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
          <Stack direction="row" spacing={2}>
            <TextField
              label={t('contacts.nickname')}
              fullWidth
              value={formData.nickname}
              onChange={handleChange('nickname')}
            />
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
          </Stack>
          <Stack direction="row" spacing={2}>
            <TextField
              label={t('contacts.email')}
              fullWidth
              type="email"
              value={formData.email}
              onChange={handleChange('email')}
            />
            <TextField
              label={t('contacts.phone')}
              fullWidth
              value={formData.phone}
              onChange={handleChange('phone')}
            />
          </Stack>
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
          <TextField
            label={t('contacts.address')}
            fullWidth
            multiline
            rows={2}
            value={formData.address}
            onChange={handleChange('address')}
          />
          <TextField
            label={t('contacts.howWeMet')}
            fullWidth
            multiline
            rows={2}
            value={formData.how_we_met}
            onChange={handleChange('how_we_met')}
          />
          <TextField
            label={t('contacts.foodPreference')}
            fullWidth
            value={formData.food_preference}
            onChange={handleChange('food_preference')}
          />
          <TextField
            label={t('contacts.workInformation')}
            fullWidth
            multiline
            rows={2}
            value={formData.work_information}
            onChange={handleChange('work_information')}
          />
          <TextField
            label={t('contacts.contactInformation')}
            fullWidth
            multiline
            rows={2}
            value={formData.contact_information}
            onChange={handleChange('contact_information')}
          />
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
                onKeyPress={(e) => {
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
    </Dialog>
  );
}
