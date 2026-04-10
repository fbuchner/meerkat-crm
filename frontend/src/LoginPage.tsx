import { useState } from 'react';
import { loginUser, isAuthenticated, API_BASE_URL } from './auth';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import i18n from './i18n/config';
import { initializeDateFormatFromBackend } from './DateFormatProvider';
import {
  Box,
  TextField,
  Button,
  Typography,
  Alert,
  Paper,
  Stack,
  Divider
} from '@mui/material';
import ForgotPasswordDialog from './components/ForgotPasswordDialog';
import { useOIDCConfig } from './hooks/useOIDCConfig';

type LoginPageProps = {
  setToken?: (token: string | null) => void;
};

const OIDC_ERROR_MAP: Record<string, string> = {
  oidc_denied: 'login.oidcDenied',
  oidc_no_account: 'login.oidcNoAccount',
  oidc_error: 'login.oidcError',
};

export default function LoginPage({ setToken }: LoginPageProps) {
  const { t } = useTranslation();
  const [searchParams] = useSearchParams();
  const [identifier, setIdentifier] = useState(searchParams.get('username') || '');
  const [password, setPassword] = useState(searchParams.get('password') || '');
  const [error, setError] = useState(() => {
    const oidcError = searchParams.get('error');
    return oidcError && OIDC_ERROR_MAP[oidcError] ? t(OIDC_ERROR_MAP[oidcError]) : '';
  });
  const [loading, setLoading] = useState(false);
  const [forgotOpen, setForgotOpen] = useState(false);
  const navigate = useNavigate();
  const oidcConfig = useOIDCConfig();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');
    try {
      const { language, date_format } = await loginUser(identifier, password);
      // Signal that user is now authenticated (token is in httpOnly cookie)
      if (setToken) setToken(isAuthenticated() ? 'authenticated' : null);

      // Sync language preference from backend if available
      if (language && language !== i18n.language) {
        i18n.changeLanguage(language);
      }

      // Sync date format preference from backend
      initializeDateFormatFromBackend(date_format);

      navigate('/');
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : t('login.loginFailed');
      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box sx={{ maxWidth: 400, mx: 'auto', mt: 8 }}>
      <Paper sx={{ p: 4 }}>
        <Box sx={{ display: 'flex', justifyContent: 'center' }}>
          <Box
            component="img"
            src="/meerkat-crm-logo.svg"
            alt="Meerkat CRM"
            sx={{ width: 150, height: 'auto' }}
          />
        </Box>
        <Typography variant="h5" mb={2}>{t('login.title')}</Typography>
        <form onSubmit={handleSubmit}>
          <Stack spacing={2}>
            <TextField
              label={t('login.identifier')}
              type="text"
              value={identifier}
              onChange={e => setIdentifier(e.target.value)}
              required
              fullWidth
            />
            <TextField
              label={t('login.password')}
              type="password"
              value={password}
              onChange={e => setPassword(e.target.value)}
              required
              fullWidth
            />
            {error && <Alert severity="error">{error}</Alert>}
            <Button type="submit" variant="contained" color="primary" disabled={loading}>
              {loading ? t('login.loggingIn') : t('login.loginButton')}
            </Button>
            <Button variant="text" color="secondary" onClick={() => setForgotOpen(true)}>
              {t('login.forgotPassword')}
            </Button>
            <Button component={Link} to="/register" color="secondary" variant="text">
              {t('login.noAccount')}
            </Button>
            {oidcConfig.enabled && (
              <>
                <Divider>{t('login.orSeparator')}</Divider>
                <Button
                  variant="outlined"
                  color="primary"
                  onClick={() => { window.location.href = `${API_BASE_URL}/auth/oidc/login`; }}
                >
                  {t('login.ssoButton', { provider: oidcConfig.provider_name })}
                </Button>
              </>
            )}
          </Stack>
        </form>
      </Paper>
      <ForgotPasswordDialog open={forgotOpen} onClose={() => setForgotOpen(false)} />
    </Box>
  );
}
