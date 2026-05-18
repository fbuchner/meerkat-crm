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
  Collapse,
  Autocomplete,
  Tooltip,
} from '@mui/material';
import WebhookIcon from '@mui/icons-material/Webhook';
import AddIcon from '@mui/icons-material/Add';
import DeleteIcon from '@mui/icons-material/Delete';
import EditIcon from '@mui/icons-material/Edit';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import {
  getWebhooks,
  createWebhook,
  updateWebhook,
  deleteWebhook,
  testWebhook,
  getWebhookDeliveries,
  Webhook,
  WebhookCreateResponse,
  WebhookDelivery,
} from '../api/webhooks';

const SUPPORTED_EVENTS = [
  'contact.created',
  'contact.updated',
  'contact.deleted',
  'note.created',
  'note.updated',
  'note.deleted',
  'activity.created',
  'activity.updated',
  'activity.deleted',
  'reminder.triggered',
  'birthday.occurred',
];

interface WebhookFormState {
  name: string;
  url: string;
  events: string[];
  is_active: boolean;
}

const emptyForm = (): WebhookFormState => ({
  name: '',
  url: '',
  events: [],
  is_active: true,
});

export default function WebhooksSettings() {
  const { t } = useTranslation();
  const [webhooks, setWebhooks] = useState<Webhook[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [form, setForm] = useState<WebhookFormState>(emptyForm());
  const [saving, setSaving] = useState(false);

  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [webhookToDelete, setWebhookToDelete] = useState<Webhook | null>(null);
  const [deleting, setDeleting] = useState(false);

  const [expandedDeliveries, setExpandedDeliveries] = useState<Record<number, boolean>>({});
  const [deliveries, setDeliveries] = useState<Record<number, WebhookDelivery[]>>({});
  const [loadingDeliveries, setLoadingDeliveries] = useState<Record<number, boolean>>({});

  const [testing, setTesting] = useState<Record<number, boolean>>({});

  const [createdWebhook, setCreatedWebhook] = useState<WebhookCreateResponse | null>(null);
  const [secretCopied, setSecretCopied] = useState(false);

  const loadWebhooks = useCallback(async () => {
    try {
      setLoading(true);
      const data = await getWebhooks();
      setWebhooks(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : t('settings.webhooks.loadError'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    loadWebhooks();
  }, [loadWebhooks]);

  const openCreate = () => {
    setEditingId(null);
    setForm(emptyForm());
    setDialogOpen(true);
  };

  const openEdit = (wh: Webhook) => {
    setEditingId(wh.id);
    setForm({ name: wh.name, url: wh.url, events: wh.events, is_active: wh.is_active });
    setDialogOpen(true);
  };

  const handleSave = async () => {
    setSaving(true);
    setError('');
    try {
      if (editingId !== null) {
        const updated = await updateWebhook(editingId, form);
        setWebhooks(prev => prev.map(w => (w.id === editingId ? updated : w)));
        setDialogOpen(false);
      } else {
        const created = await createWebhook(form);
        setWebhooks(prev => [created, ...prev]);
        setDialogOpen(false);
        setCreatedWebhook(created);
        setSecretCopied(false);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : t('settings.webhooks.saveError'));
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteClick = (wh: Webhook) => {
    setWebhookToDelete(wh);
    setDeleteDialogOpen(true);
  };

  const handleDeleteConfirm = async () => {
    if (!webhookToDelete) return;
    setDeleting(true);
    try {
      await deleteWebhook(webhookToDelete.id);
      setWebhooks(prev => prev.filter(w => w.id !== webhookToDelete.id));
      setDeleteDialogOpen(false);
      setWebhookToDelete(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : t('settings.webhooks.deleteError'));
      setDeleteDialogOpen(false);
    } finally {
      setDeleting(false);
    }
  };

  const handleTest = async (wh: Webhook) => {
    setTesting(prev => ({ ...prev, [wh.id]: true }));
    setSuccess('');
    setError('');
    try {
      const result = await testWebhook(wh.id);
      const d = result.delivery;
      if (d.status_code && d.status_code >= 200 && d.status_code < 300) {
        setSuccess(t('settings.webhooks.testSuccess', { statusCode: d.status_code }));
      } else {
        setError(t('settings.webhooks.testFailed', { error: d.error || `status ${d.status_code ?? 'unknown'}` }));
      }
      // Prepend the new delivery into the cached list so it shows immediately
      setDeliveries(prev => ({ ...prev, [wh.id]: [d, ...(prev[wh.id] || [])] }));
      // Auto-expand deliveries so the user can see the result
      setExpandedDeliveries(prev => ({ ...prev, [wh.id]: true }));
    } catch (err) {
      setError(err instanceof Error ? err.message : t('settings.webhooks.testFailed', { error: 'unknown' }));
    } finally {
      setTesting(prev => ({ ...prev, [wh.id]: false }));
    }
  };

  const toggleDeliveries = async (wh: Webhook) => {
    const isOpen = expandedDeliveries[wh.id];
    setExpandedDeliveries(prev => ({ ...prev, [wh.id]: !isOpen }));
    if (!isOpen && !deliveries[wh.id]) {
      setLoadingDeliveries(prev => ({ ...prev, [wh.id]: true }));
      try {
        const data = await getWebhookDeliveries(wh.id);
        setDeliveries(prev => ({ ...prev, [wh.id]: data }));
      } catch {
        // silently fail
      } finally {
        setLoadingDeliveries(prev => ({ ...prev, [wh.id]: false }));
      }
    }
  };

  const getEventLabel = (event: string) => {
    return t(`settings.webhooks.eventLabels.${event}`, { defaultValue: event });
  };

  return (
    <>
      <Card sx={{ mb: 2 }}>
        <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
            <Box sx={{ display: 'flex', alignItems: 'center' }}>
              <WebhookIcon sx={{ mr: 1, color: 'text.secondary', fontSize: 20 }} />
              <Typography variant="subtitle1" sx={{ fontWeight: 500 }}>
                {t('settings.webhooks.title')}
              </Typography>
            </Box>
            <Button
              variant="outlined"
              size="small"
              startIcon={<AddIcon />}
              onClick={openCreate}
            >
              {t('settings.webhooks.add')}
            </Button>
          </Box>
          <Divider sx={{ mb: 1.5 }} />

          <Stack spacing={1.5}>
            <Typography variant="body2" color="text.secondary">
              {t('settings.webhooks.description')}
            </Typography>

            {error && <Alert severity="error" sx={{ py: 0 }} onClose={() => setError('')}>{error}</Alert>}
            {success && <Alert severity="success" sx={{ py: 0 }} onClose={() => setSuccess('')}>{success}</Alert>}

            {loading ? (
              <Typography variant="body2" color="text.secondary">
                {t('settings.webhooks.loading')}
              </Typography>
            ) : webhooks.length === 0 ? (
              <Typography variant="body2" color="text.secondary" sx={{ fontStyle: 'italic' }}>
                {t('settings.webhooks.noWebhooks')}
              </Typography>
            ) : (
              <List dense sx={{ py: 0 }}>
                {webhooks.map(wh => (
                  <Box key={wh.id}>
                    <ListItem
                      sx={{ px: 0 }}
                      secondaryAction={
                        <Box sx={{ display: 'flex', gap: 0.5 }}>
                          <IconButton
                            size="small"
                            onClick={() => handleTest(wh)}
                            disabled={testing[wh.id]}
                            title={t('settings.webhooks.test')}
                          >
                            <PlayArrowIcon fontSize="small" />
                          </IconButton>
                          <IconButton
                            size="small"
                            onClick={() => openEdit(wh)}
                            title={t('settings.webhooks.edit')}
                          >
                            <EditIcon fontSize="small" />
                          </IconButton>
                          <IconButton
                            size="small"
                            onClick={() => handleDeleteClick(wh)}
                            color="error"
                            title={t('settings.webhooks.delete')}
                          >
                            <DeleteIcon fontSize="small" />
                          </IconButton>
                        </Box>
                      }
                    >
                      <ListItemText
                        primaryTypographyProps={{ component: 'div' }}
                        secondaryTypographyProps={{ component: 'div' }}
                        primary={
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                            <Typography variant="body2" sx={{ fontWeight: 500 }}>{wh.name}</Typography>
                            <Chip
                              size="small"
                              label={wh.is_active ? t('settings.webhooks.active') : t('settings.webhooks.inactive')}
                              color={wh.is_active ? 'success' : 'default'}
                              sx={{ height: 18, fontSize: '0.7rem' }}
                            />
                          </Box>
                        }
                        secondary={
                          <Box>
                            <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }}>
                              {wh.url.length > 60 ? wh.url.slice(0, 57) + '...' : wh.url}
                            </Typography>
                            <Typography variant="caption" color="text.secondary">
                              {wh.events.length} {t('settings.webhooks.eventsCount')}
                            </Typography>
                          </Box>
                        }
                      />
                    </ListItem>

                    <Box sx={{ pl: 2, pb: 1 }}>
                      <Button
                        size="small"
                        onClick={() => toggleDeliveries(wh)}
                        endIcon={expandedDeliveries[wh.id] ? <ExpandLessIcon fontSize="small" /> : <ExpandMoreIcon fontSize="small" />}
                        sx={{ fontSize: '0.75rem', color: 'text.secondary', textTransform: 'none', p: 0 }}
                      >
                        {t('settings.webhooks.deliveries')}
                      </Button>
                      <Collapse in={expandedDeliveries[wh.id]}>
                        <Box sx={{ mt: 0.5 }}>
                          {loadingDeliveries[wh.id] ? (
                            <Typography variant="caption" color="text.secondary">{t('settings.webhooks.loadingDeliveries')}</Typography>
                          ) : (deliveries[wh.id] || []).length === 0 ? (
                            <Typography variant="caption" color="text.secondary">{t('settings.webhooks.noDeliveries')}</Typography>
                          ) : (
                            (deliveries[wh.id] || []).slice(0, 5).map(d => (
                              <Box key={d.id} sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 0.5 }}>
                                <Chip
                                  size="small"
                                  label={d.status_code ?? 'err'}
                                  color={d.status_code && d.status_code < 300 ? 'success' : 'error'}
                                  sx={{ height: 18, fontSize: '0.7rem', minWidth: 36 }}
                                />
                                <Typography variant="caption" color="text.secondary">{d.event_type}</Typography>
                                <Typography variant="caption" color="text.secondary">
                                  {new Date(d.created_at).toLocaleString()}
                                </Typography>
                                {d.error && (
                                  <Typography variant="caption" color="error.main" sx={{ ml: 0.5 }}>
                                    {d.error}
                                  </Typography>
                                )}
                              </Box>
                            ))
                          )}
                        </Box>
                      </Collapse>
                    </Box>
                  </Box>
                ))}
              </List>
            )}
          </Stack>
        </CardContent>
      </Card>

      {/* Add/Edit dialog */}
      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>
          {editingId !== null ? t('settings.webhooks.editTitle') : t('settings.webhooks.addTitle')}
        </DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1 }}>
            <TextField
              label={t('settings.webhooks.name')}
              value={form.name}
              onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
              size="small"
              required
              fullWidth
            />
            <TextField
              label={t('settings.webhooks.url')}
              value={form.url}
              onChange={e => setForm(f => ({ ...f, url: e.target.value }))}
              size="small"
              required
              fullWidth
              placeholder="https://example.com/webhook"
            />
            <Autocomplete
              multiple
              options={SUPPORTED_EVENTS}
              value={form.events}
              onChange={(_, value) => setForm(f => ({ ...f, events: value }))}
              getOptionLabel={getEventLabel}
              renderInput={params => (
                <TextField {...params} label={t('settings.webhooks.events')} size="small" required />
              )}
              renderTags={(value, getTagProps) =>
                value.map((option, index) => (
                  <Chip
                    {...getTagProps({ index })}
                    key={option}
                    label={getEventLabel(option)}
                    size="small"
                  />
                ))
              }
            />
            <FormControlLabel
              control={
                <Switch
                  checked={form.is_active}
                  onChange={e => setForm(f => ({ ...f, is_active: e.target.checked }))}
                  size="small"
                />
              }
              label={t('settings.webhooks.active')}
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDialogOpen(false)}>
            {t('settings.webhooks.cancel')}
          </Button>
          <Button
            onClick={handleSave}
            variant="contained"
            disabled={saving || !form.name.trim() || !form.url.trim() || form.events.length === 0}
          >
            {saving ? t('settings.webhooks.saving') : t('settings.webhooks.save')}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Secret revealed dialog — shown once after creation */}
      <Dialog open={!!createdWebhook} onClose={() => setCreatedWebhook(null)} maxWidth="sm" fullWidth>
        <DialogTitle>{t('settings.webhooks.secretDialog.title')}</DialogTitle>
        <DialogContent>
          <Alert severity="warning" sx={{ mb: 2 }}>
            {t('apiTokens.createdDialog.warning')}
          </Alert>
          <Stack direction="row" alignItems="center" spacing={1}>
            <TextField
              value={createdWebhook?.secret || ''}
              InputProps={{ readOnly: true }}
              fullWidth
              size="small"
              inputProps={{ style: { fontFamily: 'monospace', fontSize: '0.85rem' } }}
            />
            <Tooltip title={secretCopied ? t('apiTokens.createdDialog.copied') : t('apiTokens.createdDialog.copy')}>
              <IconButton
                onClick={() => {
                  if (createdWebhook) {
                    navigator.clipboard.writeText(createdWebhook.secret);
                    setSecretCopied(true);
                    setTimeout(() => setSecretCopied(false), 2000);
                  }
                }}
                color={secretCopied ? 'success' : 'default'}
              >
                <ContentCopyIcon />
              </IconButton>
            </Tooltip>
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button variant="contained" onClick={() => setCreatedWebhook(null)}>
            {t('apiTokens.createdDialog.done')}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Delete confirmation dialog */}
      <Dialog open={deleteDialogOpen} onClose={() => setDeleteDialogOpen(false)}>
        <DialogTitle>{t('settings.webhooks.deleteDialog.title')}</DialogTitle>
        <DialogContent>
          <DialogContentText>
            {t('settings.webhooks.deleteDialog.message', { name: webhookToDelete?.name ?? '' })}
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteDialogOpen(false)}>
            {t('settings.webhooks.deleteDialog.cancel')}
          </Button>
          <Button onClick={handleDeleteConfirm} color="error" disabled={deleting} autoFocus>
            {t('settings.webhooks.deleteDialog.confirm')}
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}
