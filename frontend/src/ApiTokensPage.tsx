import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import {
  Box,
  Typography,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  IconButton,
  Chip,
  CircularProgress,
  Alert,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  Stack,
  Tooltip,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import BlockIcon from '@mui/icons-material/Block';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import { getApiTokens, createApiToken, revokeApiToken, ApiToken, ApiTokenCreateResponse } from './api/apiTokens';
import { isAdmin } from './auth';
import { useSnackbar } from './context/SnackbarContext';

export default function ApiTokensPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { showSuccess, showError } = useSnackbar();

  const [tokens, setTokens] = useState<ApiToken[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  // Create dialog
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [newTokenName, setNewTokenName] = useState('');
  const [createLoading, setCreateLoading] = useState(false);

  // Token display dialog (shown once after creation)
  const [createdToken, setCreatedToken] = useState<ApiTokenCreateResponse | null>(null);
  const [copied, setCopied] = useState(false);

  // Revoke dialog
  const [revokeDialogOpen, setRevokeDialogOpen] = useState(false);
  const [revokingToken, setRevokingToken] = useState<ApiToken | null>(null);
  const [revokeLoading, setRevokeLoading] = useState(false);

  const admin = isAdmin();

  useEffect(() => {
    if (!admin) {
      navigate('/');
    }
  }, [admin, navigate]);

  const fetchTokens = async () => {
    setLoading(true);
    setError('');
    try {
      const response = await getApiTokens();
      setTokens(response.tokens);
    } catch (err) {
      setError(err instanceof Error ? err.message : t('apiTokens.loadError'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!admin) return;
    fetchTokens();
  }, [admin]);

  const handleCreate = async () => {
    if (!newTokenName.trim()) return;
    setCreateLoading(true);
    try {
      const result = await createApiToken(newTokenName.trim());
      setCreateDialogOpen(false);
      setNewTokenName('');
      setCreatedToken(result);
      setCopied(false);
      await fetchTokens();
    } catch (err) {
      showError(err instanceof Error ? err.message : t('apiTokens.createError'));
    } finally {
      setCreateLoading(false);
    }
  };

  const handleCopy = () => {
    if (createdToken) {
      navigator.clipboard.writeText(createdToken.token);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const handleRevoke = async () => {
    if (!revokingToken) return;
    setRevokeLoading(true);
    try {
      await revokeApiToken(revokingToken.id);
      setRevokeDialogOpen(false);
      setRevokingToken(null);
      showSuccess(t('apiTokens.revokeSuccess'));
      await fetchTokens();
    } catch (err) {
      showError(err instanceof Error ? err.message : t('apiTokens.revokeError'));
    } finally {
      setRevokeLoading(false);
    }
  };

  const formatDate = (dateStr: string | null) => {
    if (!dateStr) return t('apiTokens.neverUsed');
    return new Date(dateStr).toLocaleString();
  };

  return (
    <Box sx={{ maxWidth: 1200, mx: 'auto', mt: 2, p: 2 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1.5 }}>
        <Typography variant="h5">{t('apiTokens.title')}</Typography>
        <Button variant="outlined" startIcon={<AddIcon />} onClick={() => setCreateDialogOpen(true)}>
          {t('apiTokens.createButton')}
        </Button>
      </Box>

      {loading && <CircularProgress />}
      {error && <Alert severity="error">{error}</Alert>}

      {!loading && !error && (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>{t('apiTokens.columns.name')}</TableCell>
                <TableCell>{t('apiTokens.columns.created')}</TableCell>
                <TableCell>{t('apiTokens.columns.lastUsed')}</TableCell>
                <TableCell>{t('apiTokens.columns.status')}</TableCell>
                <TableCell>{t('apiTokens.columns.actions')}</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {tokens.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} align="center">
                    <Typography color="text.secondary">{t('apiTokens.noTokens')}</Typography>
                  </TableCell>
                </TableRow>
              ) : (
                tokens.map((token) => (
                  <TableRow key={token.id}>
                    <TableCell>{token.name}</TableCell>
                    <TableCell>{new Date(token.created_at).toLocaleString()}</TableCell>
                    <TableCell>{formatDate(token.last_used_at)}</TableCell>
                    <TableCell>
                      {token.revoked_at ? (
                        <Chip label={t('apiTokens.revoked')} color="error" size="small" />
                      ) : (
                        <Chip label={t('apiTokens.active')} color="success" size="small" />
                      )}
                    </TableCell>
                    <TableCell>
                      {!token.revoked_at && (
                        <Tooltip title={t('apiTokens.revokeDialog.title')}>
                          <IconButton
                            size="small"
                            color="error"
                            onClick={() => { setRevokingToken(token); setRevokeDialogOpen(true); }}
                          >
                            <BlockIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      )}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      {/* Create dialog */}
      <Dialog open={createDialogOpen} onClose={() => setCreateDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>{t('apiTokens.createDialog.title')}</DialogTitle>
        <DialogContent>
          <TextField
            autoFocus
            label={t('apiTokens.createDialog.nameLabel')}
            value={newTokenName}
            onChange={(e) => setNewTokenName(e.target.value)}
            onKeyDown={(e) => { if (e.key === 'Enter') handleCreate(); }}
            fullWidth
            margin="normal"
            inputProps={{ maxLength: 100 }}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateDialogOpen(false)} disabled={createLoading}>
            {t('common.cancel')}
          </Button>
          <Button variant="contained" onClick={handleCreate} disabled={createLoading || !newTokenName.trim()}>
            {t('apiTokens.createDialog.createButton')}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Token created dialog */}
      <Dialog open={!!createdToken} onClose={() => setCreatedToken(null)} maxWidth="sm" fullWidth>
        <DialogTitle>{t('apiTokens.createdDialog.title')}</DialogTitle>
        <DialogContent>
          <Alert severity="warning" sx={{ mb: 2 }}>
            {t('apiTokens.createdDialog.warning')}
          </Alert>
          <Stack direction="row" alignItems="center" spacing={1}>
            <TextField
              value={createdToken?.token || ''}
              InputProps={{ readOnly: true }}
              fullWidth
              size="small"
              inputProps={{ style: { fontFamily: 'monospace', fontSize: '0.85rem' } }}
            />
            <Tooltip title={copied ? t('apiTokens.createdDialog.copied') : t('apiTokens.createdDialog.copy')}>
              <IconButton onClick={handleCopy} color={copied ? 'success' : 'default'}>
                <ContentCopyIcon />
              </IconButton>
            </Tooltip>
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button variant="contained" onClick={() => setCreatedToken(null)}>
            {t('apiTokens.createdDialog.done')}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Revoke confirmation dialog */}
      <Dialog open={revokeDialogOpen} onClose={() => setRevokeDialogOpen(false)}>
        <DialogTitle>{t('apiTokens.revokeDialog.title')}</DialogTitle>
        <DialogContent>
          <Typography>
            {t('apiTokens.revokeDialog.message', { name: revokingToken?.name || '' })}
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setRevokeDialogOpen(false)} disabled={revokeLoading}>
            {t('common.cancel')}
          </Button>
          <Button variant="contained" color="error" onClick={handleRevoke} disabled={revokeLoading}>
            {t('apiTokens.revokeDialog.confirm')}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
