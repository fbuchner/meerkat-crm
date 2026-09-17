import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Divider,
  Button,
  Stack,
  Alert,
  CircularProgress,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
} from '@mui/material';
import DownloadIcon from '@mui/icons-material/Download';
import CloudDownloadIcon from '@mui/icons-material/CloudDownload';
import { exportDataAsCsv, exportContactsAsVcf, VCardVersion } from './api/export';
import CustomFieldsSettings from './components/CustomFieldsSettings';
import ContactFieldSettings from './components/ContactFieldSettings';
import CalendarSyncSettings from './components/CalendarSyncSettings';
import ContactSyncSettings from './components/ContactSyncSettings';
import MonicaImportDialog from './components/MonicaImportDialog';

export default function DataSettingsPage() {
  const { t } = useTranslation();
  const [monicaDialogOpen, setMonicaDialogOpen] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [exportError, setExportError] = useState('');
  const [exportSuccess, setExportSuccess] = useState('');
  const [exportingVcf, setExportingVcf] = useState(false);
  const [exportVcfError, setExportVcfError] = useState('');
  const [exportVcfSuccess, setExportVcfSuccess] = useState('');
  const [vcfVersion, setVcfVersion] = useState<VCardVersion>('3.0');

  const handleExportData = async () => {
    setExportError('');
    setExportSuccess('');
    setExporting(true);

    try {
      await exportDataAsCsv();
      setExportSuccess(t('settings.export.success'));
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : t('settings.export.error');
      setExportError(errorMessage);
    } finally {
      setExporting(false);
    }
  };

  const handleExportVcf = async () => {
    setExportVcfError('');
    setExportVcfSuccess('');
    setExportingVcf(true);

    try {
      await exportContactsAsVcf(vcfVersion);
      setExportVcfSuccess(t('settings.exportVcf.success'));
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : t('settings.exportVcf.error');
      setExportVcfError(errorMessage);
    } finally {
      setExportingVcf(false);
    }
  };

  return (
    <Box sx={{ maxWidth: 1200, mx: 'auto', mt: 2, p: 2 }}>
      <Typography variant="h5" gutterBottom sx={{ mb: 1.5 }}>
        {t('settings.data.title')}
      </Typography>

      <ContactFieldSettings />

      <CustomFieldsSettings />

      <CalendarSyncSettings />

      <ContactSyncSettings />

      <Card sx={{ mb: 2 }}>
        <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
          <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
            <CloudDownloadIcon sx={{ mr: 1, color: 'text.secondary', fontSize: 20 }} />
            <Typography variant="subtitle1" sx={{ fontWeight: 500 }}>
              {t('settings.monicaImport.title')}
            </Typography>
          </Box>
          <Divider sx={{ mb: 1.5 }} />

          <Stack spacing={1.5}>
            <Typography variant="body2" color="text.secondary">
              {t('settings.monicaImport.description')}
            </Typography>
            <Button
              variant="contained"
              size="small"
              startIcon={<CloudDownloadIcon />}
              onClick={() => setMonicaDialogOpen(true)}
            >
              {t('settings.monicaImport.button')}
            </Button>
          </Stack>
        </CardContent>
      </Card>

      <MonicaImportDialog
        open={monicaDialogOpen}
        onClose={() => setMonicaDialogOpen(false)}
        onImportComplete={() => undefined}
      />

      <Card sx={{ mb: 2 }}>
        <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
          <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
            <DownloadIcon sx={{ mr: 1, color: 'text.secondary', fontSize: 20 }} />
            <Typography variant="subtitle1" sx={{ fontWeight: 500 }}>
              {t('settings.export.title')}
            </Typography>
          </Box>
          <Divider sx={{ mb: 1.5 }} />

          <Stack spacing={1.5}>
            <Typography variant="body2" color="text.secondary">
              {t('settings.export.description')}
            </Typography>
            {exportError && <Alert severity="error" sx={{ py: 0 }}>{exportError}</Alert>}
            {exportSuccess && <Alert severity="success" sx={{ py: 0 }}>{exportSuccess}</Alert>}
            <Button
              variant="contained"
              size="small"
              startIcon={exporting ? <CircularProgress size={16} color="inherit" /> : <DownloadIcon />}
              onClick={handleExportData}
              disabled={exporting}
            >
              {exporting ? t('settings.export.exporting') : t('settings.export.downloadButton')}
            </Button>
          </Stack>
        </CardContent>
      </Card>

      <Card sx={{ mb: 2 }}>
        <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
          <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
            <DownloadIcon sx={{ mr: 1, color: 'text.secondary', fontSize: 20 }} />
            <Typography variant="subtitle1" sx={{ fontWeight: 500 }}>
              {t('settings.exportVcf.title')}
            </Typography>
          </Box>
          <Divider sx={{ mb: 1.5 }} />

          <Stack spacing={1.5}>
            <Typography variant="body2" color="text.secondary">
              {t('settings.exportVcf.description')}
            </Typography>
            {exportVcfError && <Alert severity="error" sx={{ py: 0 }}>{exportVcfError}</Alert>}
            {exportVcfSuccess && <Alert severity="success" sx={{ py: 0 }}>{exportVcfSuccess}</Alert>}
            <Stack direction="row" spacing={1.5} alignItems="center">
              <FormControl size="small" sx={{ minWidth: 220 }}>
                <InputLabel id="vcf-version-label">{t('settings.exportVcf.versionLabel')}</InputLabel>
                <Select
                  labelId="vcf-version-label"
                  label={t('settings.exportVcf.versionLabel')}
                  value={vcfVersion}
                  onChange={(e) => setVcfVersion(e.target.value as VCardVersion)}
                >
                  <MenuItem value="3.0">{t('contactDetail.exportVcard3')}</MenuItem>
                  <MenuItem value="4.0">{t('contactDetail.exportVcard4')}</MenuItem>
                </Select>
              </FormControl>
              <Button
                variant="contained"
                size="small"
                startIcon={exportingVcf ? <CircularProgress size={16} color="inherit" /> : <DownloadIcon />}
                onClick={handleExportVcf}
                disabled={exportingVcf}
              >
                {exportingVcf ? t('settings.exportVcf.exporting') : t('settings.exportVcf.downloadButton')}
              </Button>
            </Stack>
          </Stack>
        </CardContent>
      </Card>
    </Box>
  );
}
