import React, { useState } from 'react';
import { loginUser, saveToken } from './auth';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  Box,
  TextField,
  Button,
  Typography,
  Alert,
  Paper,
  Stack
} from '@mui/material';

type LoginPageProps = {
  setToken?: (token: string | null) => void;
};

export default function LoginPage({ setToken }: LoginPageProps) {
  const { t } = useTranslation();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');
    try {
      const token = await loginUser(email, password);
  saveToken(token);
  if (setToken) setToken(token);
  navigate('/');
    } catch (err: any) {
      setError(err.message || t('login.loginFailed'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box sx={{ maxWidth: 400, mx: 'auto', mt: 8 }}>
      <Paper sx={{ p: 4 }}>
        <Typography variant="h5" mb={2}>{t('login.title')}</Typography>
        <form onSubmit={handleSubmit}>
          <Stack spacing={2}>
            <TextField
              label={t('login.email')}
              type="email"
              value={email}
              onChange={e => setEmail(e.target.value)}
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
            <Button component={require('react-router-dom').Link} to="/register" color="secondary" variant="text">
              {t('login.noAccount')}
            </Button>
          </Stack>
        </form>
      </Paper>
    </Box>
  );
}
