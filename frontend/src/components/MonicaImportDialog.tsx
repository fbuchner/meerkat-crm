import { useState, useEffect, useRef, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Box,
  Typography,
  Stepper,
  Step,
  StepLabel,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Select,
  MenuItem,
  FormControl,
  FormControlLabel,
  Checkbox,
  TextField,
  Chip,
  Alert,
  LinearProgress,
  IconButton,
  Tooltip,
  Link,
  Stack,
} from '@mui/material';
import AppDialog from './AppDialog';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import WarningIcon from '@mui/icons-material/Warning';
import ErrorIcon from '@mui/icons-material/Error';
import CloseIcon from '@mui/icons-material/Close';
import { useSnackbar } from '../context/SnackbarContext';
import { getErrorMessage } from '../utils/errorHandler';
import {
  monicaConnect,
  monicaStartFetch,
  monicaGetStatus,
  monicaGetPreview,
  monicaConfirm,
  monicaCancel,
  MonicaConnectResponse,
  MonicaImportStatus,
  MonicaPreviewResponse,
  MonicaImportResult,
} from '../api/monicaImport';
import { RowImportAction } from '../api/import';

interface MonicaImportDialogProps {
  open: boolean;
  onClose: () => void;
  onImportComplete: () => void;
}

type WizardStep = 'connect' | 'fetching' | 'review' | 'importing' | 'result';

const STEP_KEYS = ['connect', 'fetch', 'review', 'import', 'done'] as const;
const STEP_INDEX: Record<WizardStep, number> = {
  connect: 0,
  fetching: 1,
  review: 2,
  importing: 3,
  result: 4,
};

const POLL_INTERVAL_MS = 2000;

// A multi-minute fetch should survive a proxy hiccup or a brief network drop,
// so transient status-poll failures are tolerated up to this many in a row.
// Only a definitive "session is gone" answer aborts immediately.
const MAX_POLL_FAILURES = 3;

// The review step makes no requests of its own, but the server-side session
// (which holds the fetched snapshot) expires on an idle timer. Poll slowly to
// keep it alive while the user works through the rows.
const KEEPALIVE_INTERVAL_MS = 60000;

export default function MonicaImportDialog({
  open,
  onClose,
  onImportComplete,
}: MonicaImportDialogProps) {
  const { t } = useTranslation();
  const { showSuccess } = useSnackbar();

  const [step, setStep] = useState<WizardStep>('connect');

  // Connect state
  const [baseUrl, setBaseUrl] = useState('');
  const [apiToken, setApiToken] = useState('');
  const [includeRelationships, setIncludeRelationships] = useState(true);
  const [includeExtras, setIncludeExtras] = useState(true);
  // Defaults on: Meerkat relationships are directed and already show up on
  // the other contact as incoming, so importing both of Monica's halves
  // lists every relationship twice on both contacts.
  const [collapseReciprocal, setCollapseReciprocal] = useState(true);
  const [connectResponse, setConnectResponse] = useState<MonicaConnectResponse | null>(null);

  // Progress state
  const [status, setStatus] = useState<MonicaImportStatus | null>(null);

  // Review state
  const [preview, setPreview] = useState<MonicaPreviewResponse | null>(null);
  const [rowActions, setRowActions] = useState<Map<number, string>>(new Map());

  // Result state
  const [result, setResult] = useState<MonicaImportResult | null>(null);

  // UI state
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Session bookkeeping for cleanup on close
  const sessionRef = useRef<{ id: string } | null>(null);

  const resetDialog = useCallback(() => {
    setStep('connect');
    setBaseUrl('');
    setApiToken('');
    setIncludeRelationships(true);
    setIncludeExtras(true);
    setConnectResponse(null);
    setStatus(null);
    setPreview(null);
    setRowActions(new Map());
    setResult(null);
    setLoading(false);
    setError(null);
    sessionRef.current = null;
  }, []);

  const handleClose = () => {
    // Drop the session (and the API token held in it) server-side.
    const session = sessionRef.current;
    if (session) {
      monicaCancel(session.id).catch(() => undefined);
    }
    resetDialog();
    onClose();
  };

  const handleConnect = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await monicaConnect(baseUrl.trim(), apiToken.trim());
      setConnectResponse(response);
      sessionRef.current = { id: response.session_id };
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  const handleStartFetch = async () => {
    if (!connectResponse) return;
    setLoading(true);
    setError(null);
    try {
      await monicaStartFetch(connectResponse.session_id, {
        include_relationships: includeRelationships,
        include_extras: includeExtras,
        collapse_reciprocal_relationships: collapseReciprocal,
      });
      setStatus(null);
      setStep('fetching');
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  const loadPreview = useCallback(async (sessionId: string) => {
    try {
      const response = await monicaGetPreview(sessionId);
      setPreview(response);
      const initialActions = new Map<number, string>();
      response.rows.forEach((row) => {
        initialActions.set(row.row_index, row.suggested_action);
      });
      setRowActions(initialActions);
      setStep('review');
    } catch (err) {
      setError(getErrorMessage(err));
      setStep('connect');
    }
  }, []);

  // Poll progress while fetching or waiting for photo downloads.
  useEffect(() => {
    if (!open || !connectResponse) return;
    if (step !== 'fetching' && step !== 'importing') return;

    const sessionId = connectResponse.session_id;
    let stopped = false;
    let consecutiveFailures = 0;

    const poll = async () => {
      try {
        const current = await monicaGetStatus(sessionId);
        if (stopped) return;
        consecutiveFailures = 0;
        setStatus(current);

        if (current.phase === 'failed') {
          setError(current.error || t('settings.monicaImport.errors.fetchFailed', 'Fetching data from Monica failed'));
          setStep('connect');
        } else if (step === 'fetching' && current.phase === 'ready') {
          await loadPreview(sessionId);
        } else if (step === 'importing' && current.phase === 'done') {
          if (current.result) {
            setResult(current.result);
          }
          setStep('result');
        }
      } catch (err) {
        if (stopped) return;
        // 401/404 mean the session is genuinely gone; retrying cannot help.
        const status = (err as { status?: number })?.status;
        const sessionGone = status === 401 || status === 404;
        consecutiveFailures += 1;
        if (sessionGone || consecutiveFailures >= MAX_POLL_FAILURES) {
          setError(getErrorMessage(err));
          setStep('connect');
        }
      }
    };

    poll();
    const interval = setInterval(poll, POLL_INTERVAL_MS);
    return () => {
      stopped = true;
      clearInterval(interval);
    };
  }, [open, step, connectResponse, loadPreview, t]);

  // Keep the server-side session alive while the user reviews rows; without
  // this, a long review ends in "Import session expired" and the whole
  // rate-limited fetch has to be redone.
  useEffect(() => {
    if (!open || !connectResponse || step !== 'review') return;

    const sessionId = connectResponse.session_id;
    const interval = setInterval(() => {
      monicaGetStatus(sessionId).catch(() => {
        // Best effort: a failed keepalive is surfaced by confirm instead.
      });
    }, KEEPALIVE_INTERVAL_MS);
    return () => clearInterval(interval);
  }, [open, step, connectResponse]);

  const handleRowActionChange = (rowIndex: number, action: string) => {
    const newActions = new Map(rowActions);
    newActions.set(rowIndex, action);
    setRowActions(newActions);
  };

  const handleSetAllActions = (action: string) => {
    if (!preview) return;
    const newActions = new Map(rowActions);
    preview.rows.forEach((row) => {
      if (row.validation_errors.length > 0) return;
      if (action === 'update' && !row.duplicate_match) return;
      newActions.set(row.row_index, action);
    });
    setRowActions(newActions);
  };

  const handleConfirmImport = async () => {
    if (!preview) return;
    setLoading(true);
    setError(null);
    try {
      const actions: RowImportAction[] = [];
      rowActions.forEach((action, rowIndex) => {
        actions.push({ row_index: rowIndex, action: action as 'skip' | 'add' | 'update' });
      });
      const importResult = await monicaConfirm(preview.session_id, actions);
      setResult(importResult);

      if (importResult.created > 0 || importResult.updated > 0) {
        showSuccess(
          t('contacts.import.result.success', 'Import completed: {{created}} created, {{updated}} updated', {
            created: importResult.created,
            updated: importResult.updated,
          })
        );
        onImportComplete();
      }

      if (importResult.photos_queued > 0) {
        // Photos download in the background; keep polling until done.
        setStatus(null);
        setStep('importing');
      } else {
        setStep('result');
      }
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  const getSummaryCounts = () => {
    let toCreate = 0;
    let toUpdate = 0;
    let toSkip = 0;
    rowActions.forEach((action) => {
      if (action === 'add') toCreate++;
      else if (action === 'update') toUpdate++;
      else toSkip++;
    });
    return { toCreate, toUpdate, toSkip };
  };

  const formatDuration = (seconds: number) => {
    if (seconds < 90) {
      return t('settings.monicaImport.connect.seconds', '{{count}} seconds', { count: seconds });
    }
    return t('settings.monicaImport.connect.minutes', '{{count}} minutes', {
      count: Math.ceil(seconds / 60),
    });
  };

  const renderConnectStep = () => (
    <Box sx={{ py: 1 }}>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        {t(
          'settings.monicaImport.connect.description',
          'Import your contacts, activities, notes, reminders and relationships from a running Monica instance. Create an API token in Monica under Settings → API.'
        )}{' '}
        <Link href="https://www.monicahq.com/api" target="_blank" rel="noopener noreferrer">
          {t('settings.monicaImport.connect.tokenHelp', 'How to get an API token')}
        </Link>
      </Typography>

      <Stack spacing={2}>
        <TextField
          label={t('settings.monicaImport.connect.baseUrl', 'Monica URL')}
          placeholder="https://monica.example.com"
          value={baseUrl}
          onChange={(e) => setBaseUrl(e.target.value)}
          fullWidth
          size="small"
          disabled={!!connectResponse}
        />
        <TextField
          label={t('settings.monicaImport.connect.apiToken', 'API token')}
          type="password"
          value={apiToken}
          onChange={(e) => setApiToken(e.target.value)}
          fullWidth
          size="small"
          autoComplete="off"
          disabled={!!connectResponse}
        />
      </Stack>

      {connectResponse && (
        <Box sx={{ mt: 3 }}>
          <Alert severity="success" sx={{ mb: 2 }}>
            {t('settings.monicaImport.connect.connected', 'Connected! This Monica account contains:')}
          </Alert>
          <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', mb: 2 }}>
            <Chip label={t('settings.monicaImport.connect.contactsCount', '{{count}} contacts', { count: connectResponse.totals.contacts })} />
            <Chip label={t('settings.monicaImport.connect.activitiesCount', '{{count}} activities', { count: connectResponse.totals.activities })} />
            <Chip label={t('settings.monicaImport.connect.notesCount', '{{count}} notes', { count: connectResponse.totals.notes })} />
            <Chip label={t('settings.monicaImport.connect.remindersCount', '{{count}} reminders', { count: connectResponse.totals.reminders })} />
            {connectResponse.totals.calls + connectResponse.totals.tasks + connectResponse.totals.gifts + connectResponse.totals.debts > 0 && (
              <Chip
                label={t('settings.monicaImport.connect.extrasCount', '{{count}} calls, tasks, gifts & debts', {
                  count:
                    connectResponse.totals.calls +
                    connectResponse.totals.tasks +
                    connectResponse.totals.gifts +
                    connectResponse.totals.debts,
                })}
              />
            )}
          </Box>

          <FormControlLabel
            control={
              <Checkbox
                checked={includeRelationships}
                onChange={(e) => setIncludeRelationships(e.target.checked)}
              />
            }
            label={t('settings.monicaImport.connect.includeRelationships', 'Import relationships (slower: needs one request per contact)')}
          />
          <Box sx={{ pl: 4 }}>
            <FormControlLabel
              disabled={!includeRelationships}
              control={
                <Checkbox
                  checked={collapseReciprocal}
                  onChange={(e) => setCollapseReciprocal(e.target.checked)}
                />
              }
              label={t('settings.monicaImport.connect.collapseReciprocal', 'Import each relationship only once')}
            />
            <Typography variant="body2" color="text.secondary" sx={{ mt: -0.5, mb: 1 }}>
              {t(
                'settings.monicaImport.connect.collapseReciprocalHelp',
                'Monica saves every relationship on both contacts. Meerkat needs only one entry — the other contact shows it under "Incoming Relationships". Turn this off to create a separate entry on each contact, as Monica does.'
              )}
            </Typography>
          </Box>
          <FormControlLabel
            control={
              <Checkbox
                checked={includeExtras}
                onChange={(e) => setIncludeExtras(e.target.checked)}
              />
            }
            label={t('settings.monicaImport.connect.includeExtras', 'Import calls, tasks, gifts and debts as notes')}
          />

          <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
            {t('settings.monicaImport.connect.estimate', 'Estimated fetch time: about {{duration}} (Monica limits API requests)', {
              duration: formatDuration(
                includeRelationships
                  ? connectResponse.estimated_fetch_seconds
                  : Math.max(1, connectResponse.estimated_fetch_seconds - connectResponse.totals.contacts)
              ),
            })}
          </Typography>
        </Box>
      )}
    </Box>
  );

  const renderProgressStep = (mode: 'fetching' | 'importing') => {
    const phase = status?.phase;
    const showDeterminate = !!status && status.phase_total > 0;
    const progress = showDeterminate ? Math.min(100, (status!.phase_done / status!.phase_total) * 100) : 0;

    return (
      <Box sx={{ py: 4, textAlign: 'center' }}>
        <Typography variant="h6" gutterBottom>
          {mode === 'fetching'
            ? t('settings.monicaImport.fetchingTitle', 'Fetching data from Monica…')
            : t('settings.monicaImport.importingTitle', 'Downloading contact photos…')}
        </Typography>
        {phase && (
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            {t(`settings.monicaImport.phase.${phase}`, phase)}
            {showDeterminate && ` (${status!.phase_done}/${status!.phase_total})`}
          </Typography>
        )}
        <LinearProgress
          variant={showDeterminate ? 'determinate' : 'indeterminate'}
          value={progress}
          sx={{ maxWidth: 400, mx: 'auto' }}
        />
        <Typography variant="body2" color="text.secondary" sx={{ mt: 2 }}>
          {t('settings.monicaImport.keepOpen', 'You can leave this dialog open — the import continues on the server.')}
        </Typography>
      </Box>
    );
  };

  const renderRelatedChips = (row: MonicaPreviewResponse['rows'][number]) => {
    const parts: string[] = [];
    if (row.related.activities > 0) {
      parts.push(t('settings.monicaImport.review.activities', '{{count}} activities', { count: row.related.activities }));
    }
    const noteCount = row.related.notes + row.related.extra_notes;
    if (noteCount > 0) {
      parts.push(t('settings.monicaImport.review.notes', '{{count}} notes', { count: noteCount }));
    }
    if (row.related.reminders > 0) {
      parts.push(t('settings.monicaImport.review.reminders', '{{count}} reminders', { count: row.related.reminders }));
    }
    if (row.related.relationships > 0) {
      parts.push(t('settings.monicaImport.review.relationships', '{{count}} relationships', { count: row.related.relationships }));
    }
    if (parts.length === 0) return <Typography variant="body2" color="text.secondary">-</Typography>;
    return (
      <Typography variant="body2" color="text.secondary">
        {parts.join(', ')}
      </Typography>
    );
  };

  const renderReviewStep = () => {
    if (!preview) return null;
    const { toCreate, toUpdate, toSkip } = getSummaryCounts();

    return (
      <Box>
        <Box sx={{ mb: 2, display: 'flex', gap: 1, flexWrap: 'wrap', alignItems: 'center' }}>
          <Chip
            icon={<CheckCircleIcon />}
            label={t('contacts.import.preview.toCreate', '{{count}} to create', { count: toCreate })}
            color="success"
            variant="outlined"
          />
          <Chip
            icon={<WarningIcon />}
            label={t('contacts.import.preview.toUpdate', '{{count}} to update', { count: toUpdate })}
            color="warning"
            variant="outlined"
          />
          <Chip
            label={t('contacts.import.preview.toSkip', '{{count}} to skip', { count: toSkip })}
            variant="outlined"
          />
          {preview.error_count > 0 && (
            <Chip
              icon={<ErrorIcon />}
              label={t('contacts.import.preview.errors', '{{count}} with errors', { count: preview.error_count })}
              color="error"
              variant="outlined"
            />
          )}
          <Box sx={{ flexGrow: 1 }} />
          <FormControl size="small" sx={{ minWidth: 150 }}>
            <Select
              value=""
              displayEmpty
              onChange={(e) => e.target.value && handleSetAllActions(e.target.value)}
            >
              <MenuItem value="" disabled>
                <em>{t('settings.monicaImport.review.setAll', 'Set all to…')}</em>
              </MenuItem>
              <MenuItem value="add">{t('contacts.import.preview.actionAdd', 'Add as New')}</MenuItem>
              <MenuItem value="update">{t('contacts.import.preview.actionUpdate', 'Update Existing')}</MenuItem>
              <MenuItem value="skip">{t('contacts.import.preview.actionSkip', 'Skip')}</MenuItem>
            </Select>
          </FormControl>
        </Box>

        <TableContainer component={Paper} sx={{ maxHeight: 350 }}>
          <Table stickyHeader size="small">
            <TableHead>
              <TableRow>
                <TableCell>{t('contacts.add.firstname', 'First Name')}</TableCell>
                <TableCell>{t('contacts.add.lastname', 'Last Name')}</TableCell>
                <TableCell>{t('contacts.add.email', 'Email')}</TableCell>
                <TableCell>{t('settings.monicaImport.review.related', 'Related data')}</TableCell>
                <TableCell width={110}>{t('contacts.import.preview.status', 'Status')}</TableCell>
                <TableCell width={150}>{t('contacts.import.preview.action', 'Action')}</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {preview.rows.map((row) => (
                <TableRow key={row.row_index}>
                  <TableCell>{row.parsed_contact.firstname || '-'}</TableCell>
                  <TableCell>{row.parsed_contact.lastname || '-'}</TableCell>
                  <TableCell>{row.parsed_contact.email || '-'}</TableCell>
                  <TableCell>{renderRelatedChips(row)}</TableCell>
                  <TableCell>
                    {row.validation_errors.length > 0 ? (
                      <Tooltip title={row.validation_errors.join(', ')}>
                        <Chip icon={<ErrorIcon />} label={t('contacts.import.preview.error')} size="small" color="error" />
                      </Tooltip>
                    ) : row.duplicate_match ? (
                      <Tooltip
                        title={t('contacts.import.preview.duplicateOf', 'Matches: {{name}} ({{reason}})', {
                          name: `${row.duplicate_match.existing_firstname} ${row.duplicate_match.existing_lastname}`,
                          reason: row.duplicate_match.match_reason,
                        })}
                      >
                        <Chip icon={<WarningIcon />} label={t('contacts.import.preview.duplicateStatus')} size="small" color="warning" />
                      </Tooltip>
                    ) : (
                      <Chip icon={<CheckCircleIcon />} label={t('contacts.import.preview.valid')} size="small" color="success" />
                    )}
                  </TableCell>
                  <TableCell>
                    <FormControl size="small" fullWidth>
                      <Select
                        value={rowActions.get(row.row_index) || 'skip'}
                        onChange={(e) => handleRowActionChange(row.row_index, e.target.value)}
                        disabled={row.validation_errors.length > 0}
                      >
                        <MenuItem value="skip">{t('contacts.import.preview.actionSkip', 'Skip')}</MenuItem>
                        <MenuItem value="add">{t('contacts.import.preview.actionAdd', 'Add as New')}</MenuItem>
                        {row.duplicate_match && (
                          <MenuItem value="update">{t('contacts.import.preview.actionUpdate', 'Update Existing')}</MenuItem>
                        )}
                      </Select>
                    </FormControl>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      </Box>
    );
  };

  const renderResultStep = () => {
    if (!result) return null;

    const relatedLines: Array<[string, number]> = [
      [t('settings.monicaImport.result.activities', '{{count}} activities imported', { count: result.activities_created }), result.activities_created],
      [t('settings.monicaImport.result.notes', '{{count}} notes imported', { count: result.notes_created }), result.notes_created],
      [t('settings.monicaImport.result.reminders', '{{count}} reminders imported', { count: result.reminders_created }), result.reminders_created],
      [t('settings.monicaImport.result.relationships', '{{count}} relationships imported', { count: result.relationships_created }), result.relationships_created],
      // Only shown when collapsing actually dropped something, so it doubles
      // as confirmation that the option took effect.
      [t('settings.monicaImport.result.relationshipsCollapsed', '{{count}} reciprocal duplicates skipped (shown as incoming relationships)', { count: result.relationships_collapsed }), result.relationships_collapsed],
      [t('settings.monicaImport.result.photos', '{{count}} photos imported', { count: result.photos_saved }), result.photos_saved],
    ];

    return (
      <Box sx={{ py: 2 }}>
        <Alert severity="success" sx={{ mb: 2 }}>
          {t('contacts.import.result.title', 'Import Complete')}
        </Alert>

        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
          <Typography>
            <CheckCircleIcon color="success" sx={{ verticalAlign: 'middle', mr: 1 }} />
            {t('contacts.import.result.created', '{{count}} contacts created', { count: result.created })}
          </Typography>
          <Typography>
            <WarningIcon color="warning" sx={{ verticalAlign: 'middle', mr: 1 }} />
            {t('contacts.import.result.updated', '{{count}} contacts updated', { count: result.updated })}
          </Typography>
          <Typography color="text.secondary">
            {t('contacts.import.result.skipped', '{{count}} rows skipped', { count: result.skipped })}
          </Typography>
          {relatedLines
            .filter(([, count]) => count > 0)
            .map(([label], index) => (
              <Typography key={index} color="text.secondary">
                {label}
              </Typography>
            ))}
          {result.photos_failed > 0 && (
            <Typography color="text.secondary">
              {t('settings.monicaImport.result.photosFailed', '{{count}} photos could not be downloaded', { count: result.photos_failed })}
            </Typography>
          )}
        </Box>

        {result.errors.length > 0 && (
          <Box sx={{ mt: 2 }}>
            <Typography variant="subtitle2" color="error" gutterBottom>
              {t('contacts.import.result.errors', 'Errors')}:
            </Typography>
            {result.errors.map((message, index) => (
              <Typography key={index} variant="body2" color="error">
                {message}
              </Typography>
            ))}
          </Box>
        )}
      </Box>
    );
  };

  const renderStepContent = () => {
    switch (step) {
      case 'connect':
        return renderConnectStep();
      case 'fetching':
        return renderProgressStep('fetching');
      case 'review':
        return renderReviewStep();
      case 'importing':
        return renderProgressStep('importing');
      case 'result':
        return renderResultStep();
      default:
        return null;
    }
  };

  const renderActions = () => {
    switch (step) {
      case 'connect':
        return (
          <>
            <Button onClick={handleClose}>{t('common.cancel', 'Cancel')}</Button>
            {connectResponse ? (
              <Button variant="contained" onClick={handleStartFetch} disabled={loading}>
                {t('settings.monicaImport.connect.startFetch', 'Fetch data')}
              </Button>
            ) : (
              <Button
                variant="contained"
                onClick={handleConnect}
                disabled={loading || !baseUrl.trim() || !apiToken.trim()}
              >
                {t('settings.monicaImport.connect.connect', 'Connect')}
              </Button>
            )}
          </>
        );
      case 'fetching':
      case 'importing':
        return <Button onClick={handleClose}>{t('common.cancel', 'Cancel')}</Button>;
      case 'review':
        return (
          <>
            <Button onClick={handleClose}>{t('common.cancel', 'Cancel')}</Button>
            <Button variant="contained" onClick={handleConfirmImport} disabled={loading}>
              {t('contacts.import.button', 'Import')}
            </Button>
          </>
        );
      case 'result':
        return (
          <Button variant="contained" onClick={handleClose}>
            {t('contacts.import.result.done', 'Done')}
          </Button>
        );
      default:
        return null;
    }
  };

  return (
    <AppDialog open={open} onClose={handleClose} maxWidth="md" fullWidth>
      <DialogTitle>
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          {t('settings.monicaImport.title', 'Import from Monica')}
          <IconButton onClick={handleClose} size="small">
            <CloseIcon />
          </IconButton>
        </Box>
      </DialogTitle>

      <DialogContent dividers>
        <Stepper activeStep={STEP_INDEX[step]} sx={{ mb: 3 }}>
          {STEP_KEYS.map((key) => (
            <Step key={key}>
              <StepLabel>{t(`settings.monicaImport.steps.${key}`)}</StepLabel>
            </Step>
          ))}
        </Stepper>

        {loading && <LinearProgress sx={{ mb: 2 }} />}

        {error && (
          <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
            {error}
          </Alert>
        )}

        {renderStepContent()}
      </DialogContent>

      <DialogActions>{renderActions()}</DialogActions>
    </AppDialog>
  );
}
