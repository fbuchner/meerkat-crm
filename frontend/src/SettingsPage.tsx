import { FormEvent, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Box,
  Card,
  CardContent,
  Typography,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Divider,
  TextField,
  Button,
  Stack,
  Alert
} from '@mui/material';
import LanguageIcon from '@mui/icons-material/Language';
import LockResetIcon from '@mui/icons-material/LockReset';
import { changePassword } from './api/auth';

export default function SettingsPage() {
  const { t, i18n } = useTranslation();
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [passwordError, setPasswordError] = useState('');
  const [passwordSuccess, setPasswordSuccess] = useState('');
  const [changingPassword, setChangingPassword] = useState(false);

  const handleLanguageChange = (event: any) => {
    i18n.changeLanguage(event.target.value);
  };

  const handlePasswordChange = async (event: FormEvent) => {
    event.preventDefault();
    setPasswordError('');
    setPasswordSuccess('');

    if (newPassword !== confirmPassword) {
      setPasswordError(t('settings.password.mismatch'));
      return;
    }

    setChangingPassword(true);

    try {
      const message = await changePassword(currentPassword, newPassword);
      setPasswordSuccess(message || t('settings.password.success'));
      setCurrentPassword('');
      setNewPassword('');
      setConfirmPassword('');
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : t('settings.password.error');
      setPasswordError(errorMessage);
    } finally {
      setChangingPassword(false);
    }
  };

  return (
    <Box sx={{ maxWidth: 800, mx: 'auto', mt: 4, p: 2 }}>
      <Typography variant="h4" sx={{ mb: 3 }}>
        {t('settings.title')}
      </Typography>

      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
            <LanguageIcon sx={{ mr: 1, color: 'text.secondary' }} />
            <Typography variant="h6" sx={{ fontWeight: 500 }}>
              {t('settings.language.title')}
            </Typography>
          </Box>
          <Divider sx={{ mb: 3 }} />
          
          <FormControl fullWidth>
            <InputLabel id="language-select-label">
              {t('settings.language.label')}
            </InputLabel>
            <Select
              labelId="language-select-label"
              value={i18n.language}
              label={t('settings.language.label')}
              onChange={handleLanguageChange}
            >
              <MenuItem value="en">English</MenuItem>
              <MenuItem value="de">Deutsch</MenuItem>
            </Select>
          </FormControl>
          
          <Typography variant="caption" color="text.secondary" sx={{ mt: 2, display: 'block' }}>
            {t('settings.language.description')}
          </Typography>
        </CardContent>
      </Card>

      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
            <LockResetIcon sx={{ mr: 1, color: 'text.secondary' }} />
            <Typography variant="h6" sx={{ fontWeight: 500 }}>
              {t('settings.password.title')}
            </Typography>
          </Box>
          <Divider sx={{ mb: 3 }} />

          <form onSubmit={handlePasswordChange}>
            <Stack spacing={2}>
              <Typography variant="body2" color="text.secondary">
                {t('settings.password.description')}
              </Typography>
              {passwordError && <Alert severity="error">{passwordError}</Alert>}
              {passwordSuccess && <Alert severity="success">{passwordSuccess}</Alert>}
              <TextField
                label={t('settings.password.current')}
                type="password"
                value={currentPassword}
                onChange={event => setCurrentPassword(event.target.value)}
                fullWidth
                required
              />
              <TextField
                label={t('settings.password.new')}
                type="password"
                value={newPassword}
                onChange={event => setNewPassword(event.target.value)}
                fullWidth
                required
              />
              <TextField
                label={t('settings.password.confirm')}
                type="password"
                value={confirmPassword}
                onChange={event => setConfirmPassword(event.target.value)}
                fullWidth
                required
              />
              <Button type="submit" variant="contained" disabled={changingPassword}>
                {changingPassword ? t('settings.password.changing') : t('settings.password.changeButton')}
              </Button>
            </Stack>
          </form>
        </CardContent>
      </Card>
    </Box>
  );
}
