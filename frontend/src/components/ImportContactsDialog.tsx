import { useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Dialog,
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
  Chip,
  Alert,
  LinearProgress,
  IconButton,
  Tooltip,
} from '@mui/material';
import CloudUploadIcon from '@mui/icons-material/CloudUpload';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import WarningIcon from '@mui/icons-material/Warning';
import ErrorIcon from '@mui/icons-material/Error';
import CloseIcon from '@mui/icons-material/Close';
import { useSnackbar } from '../context/SnackbarContext';
import { getErrorMessage } from '../utils/errorHandler';
import {
  uploadCSVForImport,
  uploadVCFForImport,
  getImportPreview,
  confirmImport,
  confirmVCFImport,
  ColumnMapping,
  ImportUploadResponse,
  ImportPreviewResponse,
  RowImportAction,
  ImportResult,
  IMPORTABLE_CONTACT_FIELDS,
  CONTACT_FIELD_LABELS,
} from '../api/import';

interface ImportContactsDialogProps {
  open: boolean;
  onClose: () => void;
  onImportComplete: () => void;
}

type ImportStep = 'upload' | 'mapping' | 'preview' | 'result';
type ImportType = 'csv' | 'vcf';

const CSV_STEP_KEYS = ['upload', 'mapColumns', 'review', 'done'] as const;
const VCF_STEP_KEYS = ['upload', 'review', 'done'] as const;

export default function ImportContactsDialog({
  open,
  onClose,
  onImportComplete,
}: ImportContactsDialogProps) {
  const { t } = useTranslation();
  const { showSuccess } = useSnackbar();

  // Step state
  const [activeStep, setActiveStep] = useState(0);
  const [step, setStep] = useState<ImportStep>('upload');
  const [importType, setImportType] = useState<ImportType>('csv');

  // Upload state
  const [uploadResponse, setUploadResponse] = useState<ImportUploadResponse | null>(null);
  const [mappings, setMappings] = useState<ColumnMapping[]>([]);

  // Preview state
  const [previewResponse, setPreviewResponse] = useState<ImportPreviewResponse | null>(null);
  const [rowActions, setRowActions] = useState<Map<number, string>>(new Map());

  // Result state
  const [importResult, setImportResult] = useState<ImportResult | null>(null);

  // UI state
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [dragOver, setDragOver] = useState(false);

  // Reset dialog state
  const resetDialog = useCallback(() => {
    setActiveStep(0);
    setStep('upload');
    setImportType('csv');
    setUploadResponse(null);
    setMappings([]);
    setPreviewResponse(null);
    setRowActions(new Map());
    setImportResult(null);
    setLoading(false);
    setError(null);
    setDragOver(false);
  }, []);

  // Handle dialog close
  const handleClose = () => {
    resetDialog();
    onClose();
  };

  // Handle file upload
  const handleFileUpload = async (file: File) => {
    const fileName = file.name.toLowerCase();
    const isCSV = fileName.endsWith('.csv');
    const isVCF = fileName.endsWith('.vcf');

    if (!isCSV && !isVCF) {
      setError(t('contacts.import.errors.invalidFile', 'Please select a valid CSV or VCF file'));
      return;
    }

    const maxSize = isVCF ? 10 * 1024 * 1024 : 5 * 1024 * 1024; // 10MB for VCF, 5MB for CSV
    if (file.size > maxSize) {
      setError(t('contacts.import.errors.fileTooLarge', 'File is too large. Maximum size is {{size}}MB', {
        size: maxSize / (1024 * 1024),
      }));
      return;
    }

    setLoading(true);
    setError(null);

    try {
      if (isVCF) {
        // VCF import - goes directly to preview (no mapping needed)
        setImportType('vcf');
        const response = await uploadVCFForImport(file);
        setPreviewResponse(response);

        // Initialize row actions based on suggested actions
        const initialActions = new Map<number, string>();
        response.rows.forEach((row) => {
          initialActions.set(row.row_index, row.suggested_action);
        });
        setRowActions(initialActions);

        setStep('preview');
        setActiveStep(1); // VCF skips mapping, so preview is step 1
      } else {
        // CSV import - needs column mapping
        setImportType('csv');
        const response = await uploadCSVForImport(file);
        setUploadResponse(response);
        setMappings(response.suggested_mappings);
        setStep('mapping');
        setActiveStep(1);
      }
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  // Handle file input change
  const handleFileInputChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      handleFileUpload(file);
    }
    // Reset input value to allow re-selecting the same file
    event.target.value = '';
  };

  // Handle drag and drop
  const handleDragOver = (event: React.DragEvent) => {
    event.preventDefault();
    setDragOver(true);
  };

  const handleDragLeave = () => {
    setDragOver(false);
  };

  const handleDrop = (event: React.DragEvent) => {
    event.preventDefault();
    setDragOver(false);
    const file = event.dataTransfer.files?.[0];
    if (file) {
      handleFileUpload(file);
    }
  };

  // Handle mapping change
  const handleMappingChange = (index: number, field: string) => {
    const newMappings = [...mappings];
    newMappings[index] = { ...newMappings[index], contact_field: field };
    setMappings(newMappings);
  };

  // Handle preview generation
  const handleGeneratePreview = async () => {
    if (!uploadResponse) return;

    // Check if at least one field is mapped
    const hasMappings = mappings.some((m) => m.contact_field !== '');
    if (!hasMappings) {
      setError(t('contacts.import.errors.noMappings', 'Please map at least one column'));
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const response = await getImportPreview(uploadResponse.session_id, mappings);
      setPreviewResponse(response);

      // Initialize row actions based on suggested actions
      const initialActions = new Map<number, string>();
      response.rows.forEach((row) => {
        initialActions.set(row.row_index, row.suggested_action);
      });
      setRowActions(initialActions);

      setStep('preview');
      setActiveStep(2);
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  // Handle row action change
  const handleRowActionChange = (rowIndex: number, action: string) => {
    const newActions = new Map(rowActions);
    newActions.set(rowIndex, action);
    setRowActions(newActions);
  };

  // Handle import confirmation
  const handleConfirmImport = async () => {
    if (!previewResponse) return;

    setLoading(true);
    setError(null);

    try {
      const actions: RowImportAction[] = [];
      rowActions.forEach((action, rowIndex) => {
        actions.push({ row_index: rowIndex, action: action as 'skip' | 'add' | 'update' });
      });

      // Use appropriate confirm endpoint based on import type
      const result = importType === 'vcf'
        ? await confirmVCFImport(previewResponse.session_id, actions)
        : await confirmImport(previewResponse.session_id, actions);

      setImportResult(result);
      setStep('result');
      setActiveStep(importType === 'vcf' ? 2 : 3); // VCF has fewer steps

      if (result.created > 0 || result.updated > 0) {
        showSuccess(
          t('contacts.import.result.success', 'Import completed: {{created}} created, {{updated}} updated', {
            created: result.created,
            updated: result.updated,
          })
        );
        onImportComplete();
      }
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  // Calculate summary counts
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

  // Render upload step
  const renderUploadStep = () => (
    <Box sx={{ py: 4 }}>
      <Box
        sx={{
          border: '2px dashed',
          borderColor: dragOver ? 'primary.main' : 'grey.400',
          borderRadius: 2,
          p: 6,
          textAlign: 'center',
          cursor: 'pointer',
          bgcolor: dragOver ? 'action.hover' : 'background.paper',
          transition: 'all 0.2s',
        }}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        onClick={() => document.getElementById('import-file-input')?.click()}
      >
        <input
          id="import-file-input"
          type="file"
          accept=".csv,.vcf"
          style={{ display: 'none' }}
          onChange={handleFileInputChange}
        />
        <CloudUploadIcon sx={{ fontSize: 48, color: 'grey.500', mb: 2 }} />
        <Typography variant="h6" gutterBottom>
          {t('contacts.import.upload.dragDrop', 'Drag and drop a CSV or VCF file here, or click to select')}
        </Typography>
        <Typography variant="body2" color="text.secondary">
          {t('contacts.import.upload.supportedFormats', 'Supported formats: CSV (spreadsheet), VCF (vCard)')}
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
          {t('contacts.import.upload.maxSize', 'Maximum file size: 10MB')}
        </Typography>
      </Box>
    </Box>
  );

  // Render mapping step
  const renderMappingStep = () => {
    if (!uploadResponse) return null;

    return (
      <Box>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          {t(
            'contacts.import.mapping.description',
            "Match your CSV columns to contact fields. Columns marked 'Ignore' will not be imported."
          )}
        </Typography>
        <TableContainer component={Paper} sx={{ maxHeight: 400 }}>
          <Table stickyHeader size="small">
            <TableHead>
              <TableRow>
                <TableCell>{t('contacts.import.mapping.csvColumn', 'CSV Column')}</TableCell>
                <TableCell>{t('contacts.import.mapping.sampleData', 'Sample')}</TableCell>
                <TableCell>{t('contacts.import.mapping.mapsTo', 'Maps To')}</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {mappings.map((mapping, index) => (
                <TableRow key={index}>
                  <TableCell>
                    <Typography variant="body2" fontWeight="medium">
                      {mapping.csv_column}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2" color="text.secondary" noWrap sx={{ maxWidth: 150 }}>
                      {uploadResponse.sample_data[0]?.[index] || '-'}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <FormControl size="small" sx={{ minWidth: 180 }}>
                      <Select
                        value={mapping.contact_field}
                        onChange={(e) => handleMappingChange(index, e.target.value)}
                        displayEmpty
                      >
                        <MenuItem value="">
                          <em>{t('contacts.import.mapping.ignore', '-- Ignore --')}</em>
                        </MenuItem>
                        {IMPORTABLE_CONTACT_FIELDS.map((field) => (
                          <MenuItem key={field} value={field}>
                            {CONTACT_FIELD_LABELS[field] || field}
                          </MenuItem>
                        ))}
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

  // Render preview step
  const renderPreviewStep = () => {
    if (!previewResponse) return null;

    const { toCreate, toUpdate, toSkip } = getSummaryCounts();
    const errorCount = previewResponse.rows.filter((r) => r.validation_errors.length > 0).length;

    return (
      <Box>
        {/* Summary */}
        <Box sx={{ mb: 2, display: 'flex', gap: 1, flexWrap: 'wrap' }}>
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
          {errorCount > 0 && (
            <Chip
              icon={<ErrorIcon />}
              label={t('contacts.import.preview.errors', '{{count}} with errors', { count: errorCount })}
              color="error"
              variant="outlined"
            />
          )}
        </Box>

        {/* Preview table */}
        <TableContainer component={Paper} sx={{ maxHeight: 350 }}>
          <Table stickyHeader size="small">
            <TableHead>
              <TableRow>
                <TableCell width={50}>{t('contacts.import.preview.row', 'Row')}</TableCell>
                <TableCell>{t('contacts.add.firstname', 'First Name')}</TableCell>
                <TableCell>{t('contacts.add.lastname', 'Last Name')}</TableCell>
                <TableCell>{t('contacts.add.email', 'Email')}</TableCell>
                <TableCell width={120}>{t('contacts.import.preview.status', 'Status')}</TableCell>
                <TableCell width={160}>{t('contacts.import.preview.action', 'Action')}</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {previewResponse.rows.map((row) => (
                <TableRow key={row.row_index}>
                  <TableCell>{row.row_index + 1}</TableCell>
                  <TableCell>{row.parsed_contact.firstname || '-'}</TableCell>
                  <TableCell>{row.parsed_contact.lastname || '-'}</TableCell>
                  <TableCell>{row.parsed_contact.email || '-'}</TableCell>
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
                        <MenuItem value="skip">
                          {t('contacts.import.preview.actionSkip', 'Skip')}
                        </MenuItem>
                        <MenuItem value="add">
                          {t('contacts.import.preview.actionAdd', 'Add as New')}
                        </MenuItem>
                        {row.duplicate_match && (
                          <MenuItem value="update">
                            {t('contacts.import.preview.actionUpdate', 'Update Existing')}
                          </MenuItem>
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

  // Render result step
  const renderResultStep = () => {
    if (!importResult) return null;

    return (
      <Box sx={{ py: 2 }}>
        <Alert severity="success" sx={{ mb: 2 }}>
          {t('contacts.import.result.title', 'Import Complete')}
        </Alert>

        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
          <Typography>
            <CheckCircleIcon color="success" sx={{ verticalAlign: 'middle', mr: 1 }} />
            {t('contacts.import.result.created', '{{count}} contacts created', {
              count: importResult.created,
            })}
          </Typography>
          <Typography>
            <WarningIcon color="warning" sx={{ verticalAlign: 'middle', mr: 1 }} />
            {t('contacts.import.result.updated', '{{count}} contacts updated', {
              count: importResult.updated,
            })}
          </Typography>
          <Typography color="text.secondary">
            {t('contacts.import.result.skipped', '{{count}} rows skipped', {
              count: importResult.skipped,
            })}
          </Typography>
        </Box>

        {importResult.errors.length > 0 && (
          <Box sx={{ mt: 2 }}>
            <Typography variant="subtitle2" color="error" gutterBottom>
              {t('contacts.import.result.errors', 'Errors')}:
            </Typography>
            {importResult.errors.map((error, index) => (
              <Typography key={index} variant="body2" color="error">
                {error}
              </Typography>
            ))}
          </Box>
        )}
      </Box>
    );
  };

  // Render current step content
  const renderStepContent = () => {
    switch (step) {
      case 'upload':
        return renderUploadStep();
      case 'mapping':
        return renderMappingStep();
      case 'preview':
        return renderPreviewStep();
      case 'result':
        return renderResultStep();
      default:
        return null;
    }
  };

  // Render action buttons
  const renderActions = () => {
    switch (step) {
      case 'upload':
        return (
          <Button onClick={handleClose}>{t('common.cancel', 'Cancel')}</Button>
        );
      case 'mapping':
        return (
          <>
            <Button onClick={() => { setStep('upload'); setActiveStep(0); }}>
              {t('common.back', 'Back')}
            </Button>
            <Button variant="contained" onClick={handleGeneratePreview} disabled={loading}>
              {t('common.continue', 'Continue')}
            </Button>
          </>
        );
      case 'preview':
        return (
          <>
            <Button onClick={() => {
              if (importType === 'vcf') {
                // VCF goes back to upload (no mapping step)
                setStep('upload');
                setActiveStep(0);
                setPreviewResponse(null);
                setRowActions(new Map());
              } else {
                // CSV goes back to mapping
                setStep('mapping');
                setActiveStep(1);
              }
            }}>
              {t('common.back', 'Back')}
            </Button>
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
    <Dialog open={open} onClose={handleClose} maxWidth="md" fullWidth>
      <DialogTitle>
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          {t('contacts.import.title', 'Import Contacts')}
          <IconButton onClick={handleClose} size="small">
            <CloseIcon />
          </IconButton>
        </Box>
      </DialogTitle>

      <DialogContent dividers>
        {/* Stepper - different steps for CSV vs VCF */}
        <Stepper activeStep={activeStep} sx={{ mb: 3 }}>
          {(importType === 'vcf' ? VCF_STEP_KEYS : CSV_STEP_KEYS).map((key) => (
            <Step key={key}>
              <StepLabel>{t(`contacts.import.steps.${key}`)}</StepLabel>
            </Step>
          ))}
        </Stepper>

        {/* Loading indicator */}
        {loading && <LinearProgress sx={{ mb: 2 }} />}

        {/* Error alert */}
        {error && (
          <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
            {error}
          </Alert>
        )}

        {/* Step content */}
        {renderStepContent()}
      </DialogContent>

      <DialogActions>{renderActions()}</DialogActions>
    </Dialog>
  );
}
