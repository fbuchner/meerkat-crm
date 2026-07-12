import { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Divider,
  TextField,
  Button,
  Stack,
  Alert,
  IconButton,
  List,
  ListItem,
  ListItemText,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogContentText,
  DialogActions,
  Switch,
  FormControlLabel,
  Chip,
  CircularProgress,
  Tooltip,
} from '@mui/material';
import CalendarMonthIcon from '@mui/icons-material/CalendarMonth';
import AddIcon from '@mui/icons-material/Add';
import DeleteIcon from '@mui/icons-material/Delete';
import EditIcon from '@mui/icons-material/Edit';
import SyncIcon from '@mui/icons-material/Sync';
import {
  getCalendarSubscriptions,
  createCalendarSubscription,
  updateCalendarSubscription,
  deleteCalendarSubscription,
  syncCalendarSubscription,
  CalendarSubscription,
} from '../api/calendars';

interface CalendarFormState {
  name: string;
  url: string;
  username: string;
  password: string;
  sync_enabled: boolean;
  past_days: string;
  future_days: string;
}

const emptyForm = (): CalendarFormState => ({
  name: '',
  url: '',
  username: '',
  password: '',
  sync_enabled: true,
  past_days: '5',
  future_days: '10',
});

export default function CalendarSyncSettings() {
  const { t } = useTranslation();
  const [calendars, setCalendars] = useState<CalendarSubscription[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [editingHasPassword, setEditingHasPassword] = useState(false);
  const [editingUsername, setEditingUsername] = useState('');
  const [form, setForm] = useState<CalendarFormState>(emptyForm());
  const [saving, setSaving] = useState(false);

  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [calendarToDelete, setCalendarToDelete] = useState<CalendarSubscription | null>(null);
  const [deleting, setDeleting] = useState(false);

  const [syncing, setSyncing] = useState<Record<number, boolean>>({});

  const loadCalendars = useCallback(async () => {
    try {
      setLoading(true);
      const data = await getCalendarSubscriptions();
      setCalendars(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : t('settings.calendarSync.loadError'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    loadCalendars();
  }, [loadCalendars]);

  const openCreate = () => {
    setEditingId(null);
    setEditingHasPassword(false);
    setEditingUsername('');
    setForm(emptyForm());
    setDialogOpen(true);
  };

  const openEdit = (cal: CalendarSubscription) => {
    setEditingId(cal.id);
    setEditingHasPassword(cal.has_password);
    setEditingUsername(cal.username);
    setForm({
      name: cal.name,
      url: cal.url,
      username: cal.username,
      password: '',
      sync_enabled: cal.sync_enabled,
      past_days: String(cal.past_days),
      future_days: String(cal.future_days),
    });
    setDialogOpen(true);
  };

  const parseDays = (value: string, fallback: number) => {
    const parsed = Number.parseInt(value, 10);
    return Number.isFinite(parsed) && parsed >= 0 ? parsed : fallback;
  };

  const handleSave = async () => {
    setSaving(true);
    setError('');
    setSuccess('');
    const input = {
      name: form.name.trim(),
      url: form.url.trim(),
      username: form.username.trim(),
      password: form.password,
      // Clear a stored password when the user removed the username of a
      // previously protected calendar and left the password empty. Calendars
      // that never had a username (password-only auth) are not affected.
      clear_password:
        editingId !== null &&
        editingHasPassword &&
        editingUsername !== '' &&
        form.username.trim() === '' &&
        form.password === '',
      sync_enabled: form.sync_enabled,
      past_days: parseDays(form.past_days, 5),
      future_days: parseDays(form.future_days, 10),
    };
    try {
      if (editingId !== null) {
        const updated = await updateCalendarSubscription(editingId, input);
        setCalendars(prev => prev.map(c => (c.id === editingId ? updated : c)));
        setDialogOpen(false);
      } else {
        const created = await createCalendarSubscription(input);
        setCalendars(prev => [...prev, created]);
        setDialogOpen(false);
        // Trigger the first sync right away so the user gets feedback.
        await handleSync(created);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : t('settings.calendarSync.saveError'));
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteClick = (cal: CalendarSubscription) => {
    setCalendarToDelete(cal);
    setDeleteDialogOpen(true);
  };

  const handleDeleteConfirm = async () => {
    if (!calendarToDelete) return;
    setDeleting(true);
    try {
      await deleteCalendarSubscription(calendarToDelete.id);
      setCalendars(prev => prev.filter(c => c.id !== calendarToDelete.id));
      setDeleteDialogOpen(false);
      setCalendarToDelete(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : t('settings.calendarSync.deleteError'));
      setDeleteDialogOpen(false);
    } finally {
      setDeleting(false);
    }
  };

  const handleSync = async (cal: CalendarSubscription) => {
    setSyncing(prev => ({ ...prev, [cal.id]: true }));
    setSuccess('');
    setError('');
    try {
      const result = await syncCalendarSubscription(cal.id);
      setSuccess(
        t('settings.calendarSync.syncSuccess', {
          created: result.created,
          updated: result.updated,
          skipped: result.skipped,
        })
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : t('settings.calendarSync.syncError'));
    } finally {
      setSyncing(prev => ({ ...prev, [cal.id]: false }));
      await loadCalendars();
    }
  };

  const formatLastSync = (cal: CalendarSubscription) => {
    if (!cal.last_synced_at) return t('settings.calendarSync.neverSynced');
    return t('settings.calendarSync.lastSynced', {
      date: new Date(cal.last_synced_at).toLocaleString(),
    });
  };

  return (
    <>
      <Card sx={{ mb: 2 }}>
        <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
            <Box sx={{ display: 'flex', alignItems: 'center' }}>
              <CalendarMonthIcon sx={{ mr: 1, color: 'text.secondary', fontSize: 20 }} />
              <Typography variant="subtitle1" sx={{ fontWeight: 500 }}>
                {t('settings.calendarSync.title')}
              </Typography>
            </Box>
            <Button variant="outlined" size="small" startIcon={<AddIcon />} onClick={openCreate}>
              {t('settings.calendarSync.add')}
            </Button>
          </Box>
          <Divider sx={{ mb: 1.5 }} />

          <Stack spacing={1.5}>
            <Typography variant="body2" color="text.secondary">
              {t('settings.calendarSync.description')}
            </Typography>
            {error && <Alert severity="error" sx={{ py: 0 }} onClose={() => setError('')}>{error}</Alert>}
            {success && <Alert severity="success" sx={{ py: 0 }} onClose={() => setSuccess('')}>{success}</Alert>}

            {loading ? (
              <Box sx={{ display: 'flex', justifyContent: 'center', py: 2 }}>
                <CircularProgress size={24} />
              </Box>
            ) : calendars.length === 0 ? (
              <Typography variant="body2" color="text.secondary" sx={{ fontStyle: 'italic' }}>
                {t('settings.calendarSync.empty')}
              </Typography>
            ) : (
              <List dense disablePadding>
                {calendars.map(cal => (
                  <ListItem
                    key={cal.id}
                    divider
                    secondaryAction={
                      <Stack direction="row" spacing={0.5}>
                        <Tooltip title={t('settings.calendarSync.syncNow')}>
                          <span>
                            <IconButton
                              size="small"
                              onClick={() => handleSync(cal)}
                              disabled={!!syncing[cal.id]}
                            >
                              {syncing[cal.id] ? (
                                <CircularProgress size={18} />
                              ) : (
                                <SyncIcon fontSize="small" />
                              )}
                            </IconButton>
                          </span>
                        </Tooltip>
                        <IconButton size="small" onClick={() => openEdit(cal)}>
                          <EditIcon fontSize="small" />
                        </IconButton>
                        <IconButton size="small" onClick={() => handleDeleteClick(cal)}>
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      </Stack>
                    }
                  >
                    <ListItemText
                      primary={
                        <Stack direction="row" spacing={1} alignItems="center">
                          <Typography variant="body2" sx={{ fontWeight: 500 }}>
                            {cal.name}
                          </Typography>
                          {!cal.sync_enabled && (
                            <Chip size="small" label={t('settings.calendarSync.disabled')} />
                          )}
                          {cal.last_sync_status === 'error' && (
                            <Tooltip title={cal.last_sync_error}>
                              <Chip size="small" color="error" label={t('settings.calendarSync.syncFailed')} />
                            </Tooltip>
                          )}
                        </Stack>
                      }
                      secondary={
                        <Typography variant="caption" color="text.secondary" component="span">
                          {cal.url} — {formatLastSync(cal)}
                        </Typography>
                      }
                    />
                  </ListItem>
                ))}
              </List>
            )}
          </Stack>
        </CardContent>
      </Card>

      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>
          {editingId !== null
            ? t('settings.calendarSync.editTitle')
            : t('settings.calendarSync.addTitle')}
        </DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1 }}>
            <Alert severity="info" sx={{ py: 0.5 }}>
              {t('settings.calendarSync.dialogInfo')}
            </Alert>
            <TextField
              label={t('settings.calendarSync.name')}
              value={form.name}
              onChange={e => setForm(prev => ({ ...prev, name: e.target.value }))}
              fullWidth
              required
              size="small"
            />
            <TextField
              label={t('settings.calendarSync.url')}
              value={form.url}
              onChange={e => setForm(prev => ({ ...prev, url: e.target.value }))}
              fullWidth
              required
              size="small"
              placeholder="https://cloud.example.com/remote.php/dav/calendars/user/personal/"
              helperText={t('settings.calendarSync.urlHelp')}
            />
            {form.url.trim().toLowerCase().startsWith('http://') &&
              (form.username.trim() !== '' || form.password !== '' || editingHasPassword) && (
                <Alert severity="warning" sx={{ py: 0.5 }}>
                  {t('settings.calendarSync.insecureUrlWarning')}
                </Alert>
              )}
            <Box sx={{ display: 'grid', gap: 1.5, gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' } }}>
              <TextField
                label={t('settings.calendarSync.username')}
                value={form.username}
                onChange={e => setForm(prev => ({ ...prev, username: e.target.value }))}
                fullWidth
                size="small"
                helperText={t('settings.calendarSync.credentialsOptional')}
              />
              <TextField
                label={t('settings.calendarSync.password')}
                type="password"
                value={form.password}
                onChange={e => setForm(prev => ({ ...prev, password: e.target.value }))}
                fullWidth
                size="small"
                autoComplete="new-password"
                helperText={
                  editingId !== null && editingHasPassword
                    ? t('settings.calendarSync.passwordKeep')
                    : undefined
                }
              />
            </Box>
            <Box sx={{ display: 'grid', gap: 1.5, gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' } }}>
              <TextField
                label={t('settings.calendarSync.pastDays')}
                type="number"
                value={form.past_days}
                onChange={e => setForm(prev => ({ ...prev, past_days: e.target.value }))}
                inputProps={{ min: 0, max: 3650 }}
                size="small"
              />
              <TextField
                label={t('settings.calendarSync.futureDays')}
                type="number"
                value={form.future_days}
                onChange={e => setForm(prev => ({ ...prev, future_days: e.target.value }))}
                inputProps={{ min: 0, max: 3650 }}
                size="small"
              />
            </Box>
            <FormControlLabel
              control={
                <Switch
                  checked={form.sync_enabled}
                  onChange={e => setForm(prev => ({ ...prev, sync_enabled: e.target.checked }))}
                />
              }
              label={t('settings.calendarSync.autoSync')}
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDialogOpen(false)}>{t('common.cancel')}</Button>
          <Button
            variant="contained"
            onClick={handleSave}
            disabled={saving || !form.name.trim() || !form.url.trim()}
            startIcon={saving ? <CircularProgress size={16} color="inherit" /> : undefined}
          >
            {t('common.save')}
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={deleteDialogOpen} onClose={() => setDeleteDialogOpen(false)}>
        <DialogTitle>{t('settings.calendarSync.deleteTitle')}</DialogTitle>
        <DialogContent>
          <DialogContentText>
            {t('settings.calendarSync.deleteConfirm', { name: calendarToDelete?.name })}
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteDialogOpen(false)}>{t('common.cancel')}</Button>
          <Button color="error" variant="contained" onClick={handleDeleteConfirm} disabled={deleting}>
            {t('common.delete')}
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}
