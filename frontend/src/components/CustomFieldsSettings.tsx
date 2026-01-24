import { useState, useEffect } from 'react';
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
  ListItemSecondaryAction,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogContentText,
  DialogActions,
} from '@mui/material';
import TuneIcon from '@mui/icons-material/Tune';
import AddIcon from '@mui/icons-material/Add';
import DeleteIcon from '@mui/icons-material/Delete';
import ArrowUpwardIcon from '@mui/icons-material/ArrowUpward';
import ArrowDownwardIcon from '@mui/icons-material/ArrowDownward';
import { getCustomFieldNames, updateCustomFieldNames } from '../api/users';

export default function CustomFieldsSettings() {
  const { t } = useTranslation();
  const [fieldNames, setFieldNames] = useState<string[]>([]);
  const [newFieldName, setNewFieldName] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [fieldToDelete, setFieldToDelete] = useState<number | null>(null);
  const [hasChanges, setHasChanges] = useState(false);
  const [originalFieldNames, setOriginalFieldNames] = useState<string[]>([]);

  useEffect(() => {
    loadCustomFieldNames();
  }, []);

  const loadCustomFieldNames = async () => {
    try {
      setLoading(true);
      const names = await getCustomFieldNames();
      setFieldNames(names);
      setOriginalFieldNames(names);
    } catch (err) {
      setError(err instanceof Error ? err.message : t('settings.customFields.loadError'));
    } finally {
      setLoading(false);
    }
  };

  const handleAddField = () => {
    const trimmedName = newFieldName.trim();
    if (!trimmedName) return;
    
    // Check for duplicates (case-insensitive)
    if (fieldNames.some(name => name.toLowerCase() === trimmedName.toLowerCase())) {
      setError(t('settings.customFields.duplicateError'));
      return;
    }

    setFieldNames([...fieldNames, trimmedName]);
    setNewFieldName('');
    setHasChanges(true);
    setError('');
  };

  const handleDeleteClick = (index: number) => {
    setFieldToDelete(index);
    setDeleteDialogOpen(true);
  };

  const handleDeleteConfirm = () => {
    if (fieldToDelete !== null) {
      const newNames = fieldNames.filter((_, i) => i !== fieldToDelete);
      setFieldNames(newNames);
      setHasChanges(true);
    }
    setDeleteDialogOpen(false);
    setFieldToDelete(null);
  };

  const handleMoveUp = (index: number) => {
    if (index === 0) return;
    const newNames = [...fieldNames];
    [newNames[index - 1], newNames[index]] = [newNames[index], newNames[index - 1]];
    setFieldNames(newNames);
    setHasChanges(true);
  };

  const handleMoveDown = (index: number) => {
    if (index === fieldNames.length - 1) return;
    const newNames = [...fieldNames];
    [newNames[index], newNames[index + 1]] = [newNames[index + 1], newNames[index]];
    setFieldNames(newNames);
    setHasChanges(true);
  };

  const handleSave = async () => {
    try {
      setSaving(true);
      setError('');
      setSuccess('');
      const savedNames = await updateCustomFieldNames(fieldNames);
      setFieldNames(savedNames);
      setOriginalFieldNames(savedNames);
      setHasChanges(false);
      setSuccess(t('settings.customFields.saveSuccess'));
    } catch (err) {
      setError(err instanceof Error ? err.message : t('settings.customFields.saveError'));
    } finally {
      setSaving(false);
    }
  };

  const handleCancel = () => {
    setFieldNames(originalFieldNames);
    setHasChanges(false);
    setError('');
    setSuccess('');
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleAddField();
    }
  };

  return (
    <>
      <Card sx={{ mb: 2 }}>
        <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
          <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
            <TuneIcon sx={{ mr: 1, color: 'text.secondary', fontSize: 20 }} />
            <Typography variant="subtitle1" sx={{ fontWeight: 500 }}>
              {t('settings.customFields.title')}
            </Typography>
          </Box>
          <Divider sx={{ mb: 1.5 }} />

          <Stack spacing={1.5}>
            <Typography variant="body2" color="text.secondary">
              {t('settings.customFields.description')}
            </Typography>

            {error && <Alert severity="error" sx={{ py: 0 }}>{error}</Alert>}
            {success && <Alert severity="success" sx={{ py: 0 }}>{success}</Alert>}

            {loading ? (
              <Typography variant="body2" color="text.secondary">
                {t('settings.customFields.loading')}
              </Typography>
            ) : (
              <>
                {fieldNames.length > 0 ? (
                  <List dense sx={{ py: 0 }}>
                    {fieldNames.map((name, index) => (
                      <ListItem key={index} sx={{ px: 0 }}>
                        <ListItemText primary={name} />
                        <ListItemSecondaryAction>
                          <IconButton
                            size="small"
                            onClick={() => handleMoveUp(index)}
                            disabled={index === 0}
                            aria-label={t('settings.customFields.moveUp')}
                          >
                            <ArrowUpwardIcon fontSize="small" />
                          </IconButton>
                          <IconButton
                            size="small"
                            onClick={() => handleMoveDown(index)}
                            disabled={index === fieldNames.length - 1}
                            aria-label={t('settings.customFields.moveDown')}
                          >
                            <ArrowDownwardIcon fontSize="small" />
                          </IconButton>
                          <IconButton
                            size="small"
                            onClick={() => handleDeleteClick(index)}
                            aria-label={t('settings.customFields.delete')}
                            color="error"
                          >
                            <DeleteIcon fontSize="small" />
                          </IconButton>
                        </ListItemSecondaryAction>
                      </ListItem>
                    ))}
                  </List>
                ) : (
                  <Typography variant="body2" color="text.secondary" sx={{ fontStyle: 'italic' }}>
                    {t('settings.customFields.noFields')}
                  </Typography>
                )}

                <Box sx={{ display: 'flex', gap: 1 }}>
                  <TextField
                    size="small"
                    placeholder={t('settings.customFields.newFieldPlaceholder')}
                    value={newFieldName}
                    onChange={(e) => setNewFieldName(e.target.value)}
                    onKeyPress={handleKeyPress}
                    sx={{ flexGrow: 1 }}
                    inputProps={{ maxLength: 100 }}
                  />
                  <Button
                    variant="outlined"
                    size="small"
                    startIcon={<AddIcon />}
                    onClick={handleAddField}
                    disabled={!newFieldName.trim()}
                  >
                    {t('settings.customFields.add')}
                  </Button>
                </Box>

                {hasChanges && (
                  <Box sx={{ display: 'flex', gap: 1, justifyContent: 'flex-end' }}>
                    <Button
                      variant="outlined"
                      size="small"
                      onClick={handleCancel}
                      disabled={saving}
                    >
                      {t('settings.customFields.cancel')}
                    </Button>
                    <Button
                      variant="contained"
                      size="small"
                      onClick={handleSave}
                      disabled={saving}
                    >
                      {saving ? t('settings.customFields.saving') : t('settings.customFields.save')}
                    </Button>
                  </Box>
                )}
              </>
            )}
          </Stack>
        </CardContent>
      </Card>

      <Dialog open={deleteDialogOpen} onClose={() => setDeleteDialogOpen(false)}>
        <DialogTitle>{t('settings.customFields.deleteDialog.title')}</DialogTitle>
        <DialogContent>
          <DialogContentText>
            {t('settings.customFields.deleteDialog.message', { 
              fieldName: fieldToDelete !== null ? fieldNames[fieldToDelete] : '' 
            })}
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteDialogOpen(false)}>
            {t('settings.customFields.deleteDialog.cancel')}
          </Button>
          <Button onClick={handleDeleteConfirm} color="error" autoFocus>
            {t('settings.customFields.deleteDialog.confirm')}
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}
