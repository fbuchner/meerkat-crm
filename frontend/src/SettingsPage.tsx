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
  Alert,
  CircularProgress
} from '@mui/material';
import { SelectChangeEvent } from '@mui/material/Select';
import LanguageIcon from '@mui/icons-material/Language';
import LockResetIcon from '@mui/icons-material/LockReset';
import DarkModeIcon from '@mui/icons-material/DarkMode';
import CalendarMonthIcon from '@mui/icons-material/CalendarMonth';
import DownloadIcon from '@mui/icons-material/Download';
import InfoIcon from '@mui/icons-material/Info';
import GitHubIcon from '@mui/icons-material/GitHub';
import Link from '@mui/material/Link';
import { changePassword, updateLanguage } from './api/auth';
import { exportDataAsCsv } from './api/export';
import { ThemePreference, useThemePreference } from './AppThemeProvider';
import { DateFormat, useDateFormat } from './DateFormatProvider';

export default function SettingsPage() {
  const { t, i18n } = useTranslation();
  const { preference: themePreference, setPreference: setThemePreference } = useThemePreference();
  const { dateFormat, setDateFormat } = useDateFormat();
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [passwordError, setPasswordError] = useState('');
  const [passwordSuccess, setPasswordSuccess] = useState('');
  const [changingPassword, setChangingPassword] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [exportError, setExportError] = useState('');
  const [exportSuccess, setExportSuccess] = useState('');

  const handleLanguageChange = async (event: any) => {
    const newLang = event.target.value;
    // Update frontend i18n immediately for responsive UI
    i18n.changeLanguage(newLang);
    
    // Sync to backend for email language preferences (fire and forget)
    try {
      await updateLanguage(newLang);
    } catch (error) {
      // Silently fail - the frontend language is still updated
      // Backend sync failure doesn't affect UI language
      console.error('Failed to sync language to backend:', error);
    }
  };

  const handleThemeChange = (event: SelectChangeEvent<ThemePreference>) => {
    setThemePreference(event.target.value as ThemePreference);
  };

  const handleDateFormatChange = (event: SelectChangeEvent<DateFormat>) => {
    setDateFormat(event.target.value as DateFormat);
  };

  const handleExportData = async () => {
    setExportError('');
    setExportSuccess('');
    setExporting(true);

    try {
      await exportDataAsCsv();
      setExportSuccess(t('settings.export.success'));
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : t('settings.export.error');
      setExportError(errorMessage);
    } finally {
      setExporting(false);
    }
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
    <Box sx={{ maxWidth: 1200, mx: 'auto', mt: 2, p: 2 }}>
      <Typography variant="h5" gutterBottom sx={{ mb: 1.5 }}>
        {t('settings.title')}
      </Typography>

      <Card sx={{ mb: 2 }}>
        <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
          <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
            <InfoIcon sx={{ mr: 1, color: 'text.secondary', fontSize: 20 }} />
            <Typography variant="subtitle1" sx={{ fontWeight: 500 }}>
              {t('settings.about.title')}
            </Typography>
          </Box>
          <Divider sx={{ mb: 1.5 }} />

          <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 2 }}>
            <img
              src="/meerkat-crm-logo.svg"
              alt="Meerkat CRM Logo"
              style={{ height: 100, flexShrink: 0 }}
            />
            <Stack spacing={1}>
              <Typography variant="body2" color="text.secondary">
                {t('settings.about.description')}
              </Typography>
              <Typography variant="body2" color="text.secondary">
                {t('settings.about.contribute')}
              </Typography>
              <Box sx={{ display: 'flex', alignItems: 'center', mt: 0.5 }}>
                <GitHubIcon sx={{ mr: 1, fontSize: 18, color: 'text.secondary' }} />
                <Link
                  href="https://github.com/fbuchner/meerkat-crm"
                  target="_blank"
                  rel="noopener noreferrer"
                  underline="hover"
                >
                  github.com/fbuchner/meerkat-crm
                </Link>
              </Box>
            </Stack>
          </Box>
        </CardContent>
      </Card>

      <Card sx={{ mb: 2 }}>
        <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
          <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
            <LanguageIcon sx={{ mr: 1, color: 'text.secondary', fontSize: 20 }} />
            <Typography variant="subtitle1" sx={{ fontWeight: 500 }}>
              {t('settings.language.title')}
            </Typography>
          </Box>
          <Divider sx={{ mb: 1.5 }} />
          
          <FormControl fullWidth size="small">
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
          
          <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: 'block' }}>
            {t('settings.language.description')}
          </Typography>
        </CardContent>
      </Card>

      <Card sx={{ mb: 2 }}>
        <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
          <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
            <CalendarMonthIcon sx={{ mr: 1, color: 'text.secondary', fontSize: 20 }} />
            <Typography variant="subtitle1" sx={{ fontWeight: 500 }}>
              {t('settings.dateFormat.title')}
            </Typography>
          </Box>
          <Divider sx={{ mb: 1.5 }} />

          <FormControl fullWidth size="small">
            <InputLabel id="date-format-select-label">
              {t('settings.dateFormat.label')}
            </InputLabel>
            <Select
              labelId="date-format-select-label"
              value={dateFormat}
              label={t('settings.dateFormat.label')}
              onChange={handleDateFormatChange}
            >
              <MenuItem value="eu">{t('settings.dateFormat.options.eu')}</MenuItem>
              <MenuItem value="us">{t('settings.dateFormat.options.us')}</MenuItem>
            </Select>
          </FormControl>

          <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: 'block' }}>
            {t('settings.dateFormat.description')}
          </Typography>
        </CardContent>
      </Card>

      <Card sx={{ mb: 2 }}>
        <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
          <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
            <DarkModeIcon sx={{ mr: 1, color: 'text.secondary', fontSize: 20 }} />
            <Typography variant="subtitle1" sx={{ fontWeight: 500 }}>
              {t('settings.theme.title')}
            </Typography>
          </Box>
          <Divider sx={{ mb: 1.5 }} />

          <FormControl fullWidth size="small">
            <InputLabel id="theme-select-label">
              {t('settings.theme.label')}
            </InputLabel>
            <Select
              labelId="theme-select-label"
              value={themePreference}
              label={t('settings.theme.label')}
              onChange={handleThemeChange}
            >
              <MenuItem value="system">{t('settings.theme.options.system')}</MenuItem>
              <MenuItem value="light">{t('settings.theme.options.light')}</MenuItem>
              <MenuItem value="dark">{t('settings.theme.options.dark')}</MenuItem>
            </Select>
          </FormControl>

          <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: 'block' }}>
            {t('settings.theme.description')}
          </Typography>
        </CardContent>
      </Card>

      <Card sx={{ mb: 2 }}>
        <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
          <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
            <LockResetIcon sx={{ mr: 1, color: 'text.secondary', fontSize: 20 }} />
            <Typography variant="subtitle1" sx={{ fontWeight: 500 }}>
              {t('settings.password.title')}
            </Typography>
          </Box>
          <Divider sx={{ mb: 1.5 }} />

          <form onSubmit={handlePasswordChange}>
            <Stack spacing={1.5}>
              <Typography variant="body2" color="text.secondary">
                {t('settings.password.description')}
              </Typography>
              {passwordError && <Alert severity="error" sx={{ py: 0 }}>{passwordError}</Alert>}
              {passwordSuccess && <Alert severity="success" sx={{ py: 0 }}>{passwordSuccess}</Alert>}
              <TextField
                label={t('settings.password.current')}
                type="password"
                value={currentPassword}
                onChange={event => setCurrentPassword(event.target.value)}
                fullWidth
                required
                size="small"
              />
              <TextField
                label={t('settings.password.new')}
                type="password"
                value={newPassword}
                onChange={event => setNewPassword(event.target.value)}
                fullWidth
                required
                size="small"
              />
              <TextField
                label={t('settings.password.confirm')}
                type="password"
                value={confirmPassword}
                onChange={event => setConfirmPassword(event.target.value)}
                fullWidth
                required
                size="small"
              />
              <Button type="submit" variant="contained" size="small" disabled={changingPassword}>
                {changingPassword ? t('settings.password.changing') : t('settings.password.changeButton')}
              </Button>
            </Stack>
          </form>
        </CardContent>
      </Card>

      <Card sx={{ mb: 2 }}>
        <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
          <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
            <DownloadIcon sx={{ mr: 1, color: 'text.secondary', fontSize: 20 }} />
            <Typography variant="subtitle1" sx={{ fontWeight: 500 }}>
              {t('settings.export.title')}
            </Typography>
          </Box>
          <Divider sx={{ mb: 1.5 }} />

          <Stack spacing={1.5}>
            <Typography variant="body2" color="text.secondary">
              {t('settings.export.description')}
            </Typography>
            {exportError && <Alert severity="error" sx={{ py: 0 }}>{exportError}</Alert>}
            {exportSuccess && <Alert severity="success" sx={{ py: 0 }}>{exportSuccess}</Alert>}
            <Button
              variant="contained"
              size="small"
              startIcon={exporting ? <CircularProgress size={16} color="inherit" /> : <DownloadIcon />}
              onClick={handleExportData}
              disabled={exporting}
            >
              {exporting ? t('settings.export.exporting') : t('settings.export.downloadButton')}
            </Button>
          </Stack>
        </CardContent>
      </Card>
    </Box>
  );
}
