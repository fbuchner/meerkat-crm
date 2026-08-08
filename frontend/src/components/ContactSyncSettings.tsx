import { useState, useEffect, useCallback, useRef } from 'react';
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
  Dialog,
  DialogTitle,
  DialogContent,
  DialogContentText,
  DialogActions,
  Switch,
  Checkbox,
  FormControlLabel,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Radio,
  RadioGroup,
  Chip,
  CircularProgress,
  Tooltip,
} from '@mui/material';
import ContactsIcon from '@mui/icons-material/Contacts';
import SyncIcon from '@mui/icons-material/Sync';
import EditIcon from '@mui/icons-material/Edit';
import LinkOffIcon from '@mui/icons-material/LinkOff';
import AddIcon from '@mui/icons-material/Add';
import {
  getCardDAVConnection,
  saveCardDAVConnection,
  deleteCardDAVConnection,
  discoverCardDAVAddressBooks,
  startCardDAVSync,
  CardDAVConnection,
  CardDAVDirection,
  DiscoveredAddressBook,
} from '../api/carddav';

type DialogStep = 'server' | 'addressBook';

// How often to re-read the connection while a sync runs on the server.
const SYNC_POLL_INTERVAL_MS = 2000;

export default function ContactSyncSettings() {
  const { t } = useTranslation();
  const [connection, setConnection] = useState<CardDAVConnection | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  const [dialogOpen, setDialogOpen] = useState(false);
  const [step, setStep] = useState<DialogStep>('server');
  const [baseUrl, setBaseUrl] = useState('');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [clearPassword, setClearPassword] = useState(false);
  const [discovering, setDiscovering] = useState(false);
  const [dialogError, setDialogError] = useState('');
  const [books, setBooks] = useState<DiscoveredAddressBook[]>([]);
  const [selectedPath, setSelectedPath] = useState('');
  const [direction, setDirection] = useState<CardDAVDirection>('two_way');
  const [syncEnabled, setSyncEnabled] = useState(true);
  const [saving, setSaving] = useState(false);

  const [disconnectDialogOpen, setDisconnectDialogOpen] = useState(false);
  const [disconnecting, setDisconnecting] = useState(false);
  const [syncing, setSyncing] = useState(false);

  // `quiet` re-reads without the spinner, for the poll while a sync is running.
  const loadConnection = useCallback(
    async (quiet = false) => {
      try {
        if (!quiet) setLoading(true);
        setConnection(await getCardDAVConnection());
      } catch (err) {
        setError(err instanceof Error ? err.message : t('settings.contactSync.loadError'));
      } finally {
        if (!quiet) setLoading(false);
      }
    },
    [t]
  );

  useEffect(() => {
    loadConnection();
  }, [loadConnection]);

  // Syncs run in the background on the server, so poll for the outcome while
  // one is in flight. Cleared as soon as it finishes, and on unmount.
  const syncRunning = connection?.sync_running ?? false;
  useEffect(() => {
    if (!syncRunning) return;
    const timer = setInterval(() => loadConnection(true), SYNC_POLL_INTERVAL_MS);
    return () => clearInterval(timer);
  }, [syncRunning, loadConnection]);

  // Report the result once, on the transition out of "running".
  const wasSyncing = useRef(false);
  useEffect(() => {
    if (wasSyncing.current && !syncRunning && connection) {
      if (connection.last_sync_status === 'error') {
        setError(connection.last_sync_error || t('settings.contactSync.syncError'));
      } else {
        const stats = connection.last_sync_stats;
        setSuccess(
          t('settings.contactSync.syncSuccess', {
            pulled: stats
              ? stats.pulled_created + stats.pulled_updated + stats.pulled_deleted
              : 0,
            pushed: stats
              ? stats.pushed_created + stats.pushed_updated + stats.pushed_deleted
              : 0,
            conflicts: stats?.conflicts ?? 0,
          })
        );
      }
    }
    wasSyncing.current = syncRunning;
  }, [syncRunning, connection, t]);

  const openDialog = () => {
    setStep('server');
    setDialogError('');
    setBooks([]);
    if (connection) {
      setBaseUrl(connection.base_url);
      setUsername(connection.username);
      setSelectedPath(connection.address_book_path);
      setDirection(connection.direction);
      setSyncEnabled(connection.sync_enabled);
    } else {
      setBaseUrl('');
      setUsername('');
      setSelectedPath('');
      setDirection('two_way');
      setSyncEnabled(true);
    }
    setPassword('');
    setClearPassword(false);
    setDialogOpen(true);
  };

  const handleDiscover = async () => {
    setDiscovering(true);
    setDialogError('');
    try {
      // An empty password reuses the stored credentials server-side.
      const found = await discoverCardDAVAddressBooks({
        base_url: baseUrl.trim(),
        username: username.trim(),
        password,
      });
      setBooks(found);
      if (!found.some(book => book.path === selectedPath)) {
        setSelectedPath(found[0]?.path || '');
      }
      setStep('addressBook');
    } catch (err) {
      setDialogError(err instanceof Error ? err.message : t('settings.contactSync.discoverError'));
    } finally {
      setDiscovering(false);
    }
  };

  const handleSave = async () => {
    setSaving(true);
    setDialogError('');
    setSuccess('');
    const selectedBook = books.find(book => book.path === selectedPath);
    try {
      const saved = await saveCardDAVConnection({
        base_url: baseUrl.trim(),
        address_book_path: selectedPath,
        address_book_name: selectedBook?.name || '',
        username: username.trim(),
        password: clearPassword ? '' : password,
        clear_password: clearPassword,
        direction,
        sync_enabled: syncEnabled,
      });
      setConnection(saved);
      setDialogOpen(false);
      // Trigger the first sync right away so the user gets feedback.
      await handleSync();
    } catch (err) {
      setDialogError(err instanceof Error ? err.message : t('settings.contactSync.saveError'));
    } finally {
      setSaving(false);
    }
  };

  // Kicks off the run and returns; the poll above reports how it went.
  const handleSync = async () => {
    setSyncing(true);
    setError('');
    setSuccess('');
    try {
      await startCardDAVSync();
    } catch (err) {
      setError(err instanceof Error ? err.message : t('settings.contactSync.syncError'));
    } finally {
      // Refresh before clearing the local flag, so the button stays disabled
      // straight through the handover to the server-reported running state.
      await loadConnection(true);
      setSyncing(false);
    }
  };

  const handleDisconnect = async () => {
    setDisconnecting(true);
    try {
      await deleteCardDAVConnection();
      setConnection(null);
      setDisconnectDialogOpen(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : t('settings.contactSync.deleteError'));
      setDisconnectDialogOpen(false);
    } finally {
      setDisconnecting(false);
    }
  };

  const directionLabel = (value: CardDAVDirection) =>
    t(`settings.contactSync.direction.${value === 'two_way' ? 'twoWay' : value}`);

  const formatLastSync = (conn: CardDAVConnection) => {
    if (!conn.last_synced_at) return t('settings.contactSync.neverSynced');
    return t('settings.contactSync.lastSynced', {
      date: new Date(conn.last_synced_at).toLocaleString(),
    });
  };

  const statusChip = (conn: CardDAVConnection) => {
    if (conn.last_sync_status === 'error') {
      return (
        <Tooltip title={conn.last_sync_error}>
          <Chip size="small" color="error" label={t('settings.contactSync.syncFailed')} />
        </Tooltip>
      );
    }
    if (conn.last_sync_status === 'partial') {
      return (
        <Tooltip title={conn.last_sync_error}>
          <Chip size="small" color="warning" label={t('settings.contactSync.syncPartial')} />
        </Tooltip>
      );
    }
    return null;
  };

  return (
    <>
      <Card sx={{ mb: 2 }}>
        <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
            <Box sx={{ display: 'flex', alignItems: 'center' }}>
              <ContactsIcon sx={{ mr: 1, color: 'text.secondary', fontSize: 20 }} />
              <Typography variant="subtitle1" sx={{ fontWeight: 500 }}>
                {t('settings.contactSync.title')}
              </Typography>
            </Box>
            {!connection && !loading && (
              <Button variant="outlined" size="small" startIcon={<AddIcon />} onClick={openDialog}>
                {t('settings.contactSync.connect')}
              </Button>
            )}
          </Box>
          <Divider sx={{ mb: 1.5 }} />

          <Stack spacing={1.5}>
            <Typography variant="body2" color="text.secondary">
              {t('settings.contactSync.description')}
            </Typography>
            {error && <Alert severity="error" sx={{ py: 0 }} onClose={() => setError('')}>{error}</Alert>}
            {success && <Alert severity="success" sx={{ py: 0 }} onClose={() => setSuccess('')}>{success}</Alert>}

            {loading ? (
              <Box sx={{ display: 'flex', justifyContent: 'center', py: 2 }}>
                <CircularProgress size={24} />
              </Box>
            ) : !connection ? (
              <Typography variant="body2" color="text.secondary" sx={{ fontStyle: 'italic' }}>
                {t('settings.contactSync.empty')}
              </Typography>
            ) : (
              <Box
                sx={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  flexWrap: 'wrap',
                  gap: 1,
                }}
              >
                <Box>
                  <Stack direction="row" spacing={1} alignItems="center">
                    <Typography variant="body2" sx={{ fontWeight: 500 }}>
                      {connection.address_book_name || connection.address_book_path}
                    </Typography>
                    <Chip size="small" variant="outlined" label={directionLabel(connection.direction)} />
                    {!connection.sync_enabled && (
                      <Chip size="small" label={t('settings.contactSync.disabled')} />
                    )}
                    {syncRunning ? (
                      <Chip
                        size="small"
                        color="info"
                        icon={<CircularProgress size={12} color="inherit" />}
                        label={t('settings.contactSync.syncing')}
                      />
                    ) : (
                      statusChip(connection)
                    )}
                  </Stack>
                  <Typography variant="caption" color="text.secondary" component="span">
                    {connection.base_url} — {formatLastSync(connection)}
                  </Typography>
                </Box>
                <Stack direction="row" spacing={0.5}>
                  <Tooltip title={t('settings.contactSync.syncNow')}>
                    <span>
                      <Button
                        size="small"
                        onClick={handleSync}
                        disabled={syncing || syncRunning}
                        startIcon={
                          syncing || syncRunning ? (
                            <CircularProgress size={16} />
                          ) : (
                            <SyncIcon fontSize="small" />
                          )
                        }
                      >
                        {t('settings.contactSync.syncNow')}
                      </Button>
                    </span>
                  </Tooltip>
                  <Button size="small" onClick={openDialog} startIcon={<EditIcon fontSize="small" />}>
                    {t('common.edit')}
                  </Button>
                  <Button
                    size="small"
                    color="error"
                    onClick={() => setDisconnectDialogOpen(true)}
                    startIcon={<LinkOffIcon fontSize="small" />}
                  >
                    {t('settings.contactSync.disconnect')}
                  </Button>
                </Stack>
              </Box>
            )}
          </Stack>
        </CardContent>
      </Card>

      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>
          {connection
            ? t('settings.contactSync.editTitle')
            : t('settings.contactSync.connectTitle')}
        </DialogTitle>
        <DialogContent>
          {step === 'server' ? (
            <Stack spacing={2} sx={{ mt: 1 }}>
              <Alert severity="info" sx={{ py: 0.5 }}>
                {t('settings.contactSync.dialogInfo')}
              </Alert>
              {dialogError && <Alert severity="error" sx={{ py: 0 }}>{dialogError}</Alert>}
              <TextField
                label={t('settings.contactSync.serverUrl')}
                value={baseUrl}
                onChange={e => setBaseUrl(e.target.value)}
                fullWidth
                required
                size="small"
                placeholder="https://cloud.example.com/remote.php/dav"
                helperText={t('settings.contactSync.serverUrlHelp')}
              />
              <Box sx={{ display: 'grid', gap: 1.5, gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' } }}>
                <TextField
                  label={t('settings.contactSync.username')}
                  value={username}
                  onChange={e => setUsername(e.target.value)}
                  fullWidth
                  size="small"
                />
                <TextField
                  label={t('settings.contactSync.password')}
                  type="password"
                  value={clearPassword ? '' : password}
                  onChange={e => setPassword(e.target.value)}
                  fullWidth
                  size="small"
                  disabled={clearPassword}
                  autoComplete="new-password"
                  helperText={
                    connection?.has_password
                      ? t('settings.contactSync.passwordKeep')
                      : t('settings.contactSync.passwordHelp')
                  }
                />
              </Box>
              {connection?.has_password && (
                <FormControlLabel
                  control={
                    <Checkbox
                      size="small"
                      checked={clearPassword}
                      onChange={e => setClearPassword(e.target.checked)}
                    />
                  }
                  label={
                    <Typography variant="body2">
                      {t('settings.contactSync.clearPassword')}
                    </Typography>
                  }
                />
              )}
            </Stack>
          ) : (
            <Stack spacing={2} sx={{ mt: 1 }}>
              {dialogError && <Alert severity="error" sx={{ py: 0 }}>{dialogError}</Alert>}
              <Typography variant="body2" color="text.secondary">
                {t('settings.contactSync.chooseAddressBook')}
              </Typography>
              <RadioGroup value={selectedPath} onChange={e => setSelectedPath(e.target.value)}>
                {books.map(book => (
                  <FormControlLabel
                    key={book.path}
                    value={book.path}
                    control={<Radio size="small" />}
                    label={
                      <Box>
                        <Typography variant="body2">{book.name || book.path}</Typography>
                        {book.description && (
                          <Typography variant="caption" color="text.secondary">
                            {book.description}
                          </Typography>
                        )}
                      </Box>
                    }
                  />
                ))}
              </RadioGroup>
              <FormControl size="small" fullWidth>
                <InputLabel id="carddav-direction-label">
                  {t('settings.contactSync.directionLabel')}
                </InputLabel>
                <Select
                  labelId="carddav-direction-label"
                  label={t('settings.contactSync.directionLabel')}
                  value={direction}
                  onChange={e => setDirection(e.target.value as CardDAVDirection)}
                >
                  <MenuItem value="two_way">{t('settings.contactSync.direction.twoWay')}</MenuItem>
                  <MenuItem value="pull">{t('settings.contactSync.direction.pull')}</MenuItem>
                  <MenuItem value="push">{t('settings.contactSync.direction.push')}</MenuItem>
                </Select>
              </FormControl>
              {direction === 'two_way' && (
                <Alert severity="warning" sx={{ py: 0.5 }}>
                  {t('settings.contactSync.loopWarning')}
                </Alert>
              )}
              <FormControlLabel
                control={
                  <Switch checked={syncEnabled} onChange={e => setSyncEnabled(e.target.checked)} />
                }
                label={t('settings.contactSync.autoSync')}
              />
            </Stack>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDialogOpen(false)}>{t('common.cancel')}</Button>
          {step === 'addressBook' && (
            <Button onClick={() => setStep('server')}>{t('common.back')}</Button>
          )}
          {step === 'server' ? (
            <Button
              variant="contained"
              onClick={handleDiscover}
              disabled={discovering || !baseUrl.trim()}
              startIcon={discovering ? <CircularProgress size={16} color="inherit" /> : undefined}
            >
              {t('settings.contactSync.next')}
            </Button>
          ) : (
            <Button
              variant="contained"
              onClick={handleSave}
              disabled={saving || !selectedPath}
              startIcon={saving ? <CircularProgress size={16} color="inherit" /> : undefined}
            >
              {t('common.save')}
            </Button>
          )}
        </DialogActions>
      </Dialog>

      <Dialog open={disconnectDialogOpen} onClose={() => setDisconnectDialogOpen(false)}>
        <DialogTitle>{t('settings.contactSync.disconnectTitle')}</DialogTitle>
        <DialogContent>
          <DialogContentText>{t('settings.contactSync.disconnectConfirm')}</DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDisconnectDialogOpen(false)}>{t('common.cancel')}</Button>
          <Button color="error" variant="contained" onClick={handleDisconnect} disabled={disconnecting}>
            {t('settings.contactSync.disconnect')}
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}
