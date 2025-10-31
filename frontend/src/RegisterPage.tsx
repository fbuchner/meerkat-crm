import { useState } from 'react';
import { API_BASE_URL } from './auth';
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

export default function RegisterPage() {
  const { t } = useTranslation();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [name, setName] = useState('');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');
    setSuccess('');
    try {
      const response = await fetch(`${API_BASE_URL}/register`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ email, password, name }),
      });
      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.error || t('register.registrationFailed'));
      }
      setSuccess(t('register.registrationSuccess'));
      setTimeout(() => navigate('/login'), 1500);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : t('register.registrationFailed');
      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box sx={{ maxWidth: 400, mx: 'auto', mt: 8 }}>
      <Paper sx={{ p: 4 }}>
        <Typography variant="h5" mb={2}>{t('register.title')}</Typography>
        <form onSubmit={handleSubmit}>
          <Stack spacing={2}>
            <TextField
              label={t('register.name')}
              value={name}
              onChange={e => setName(e.target.value)}
              required
              fullWidth
            />
            <TextField
              label={t('register.email')}
              type="email"
              value={email}
              onChange={e => setEmail(e.target.value)}
              required
              fullWidth
            />
            <TextField
              label={t('register.password')}
              type="password"
              value={password}
              onChange={e => setPassword(e.target.value)}
              required
              fullWidth
            />
            {error && <Alert severity="error">{error}</Alert>}
            {success && <Alert severity="success">{success}</Alert>}
            <Button type="submit" variant="contained" color="primary" disabled={loading}>
              {loading ? t('register.registering') : t('register.registerButton')}
            </Button>
          </Stack>
        </form>
      </Paper>
    </Box>
  );
}
