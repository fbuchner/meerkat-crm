// frontend/src/components/EmploymentHistoryList.tsx
import {
  Box,
  Typography,
  IconButton,
  Stack,
  Paper,
  Chip,
  Tooltip,
} from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import EventBusyIcon from '@mui/icons-material/EventBusy';
import { useTranslation } from 'react-i18next';
import { EmploymentHistoryEntry } from '../api/employmentHistory';

interface EmploymentHistoryListProps {
  entries: EmploymentHistoryEntry[];
  onEdit: (entry: EmploymentHistoryEntry) => void;
  onDelete: (entryId: number) => void;
  onEnd: (entryId: number) => void;
}

export default function EmploymentHistoryList({
  entries,
  onEdit,
  onDelete,
  onEnd,
}: EmploymentHistoryListProps) {
  const { t } = useTranslation();

  const handleDeleteClick = (entryId: number) => {
    if (window.confirm(t('employmentHistory.deleteMessage'))) {
      onDelete(entryId);
    }
  };

  const handleEndClick = (entryId: number) => {
    if (window.confirm(t('employmentHistory.endMessage'))) {
      onEnd(entryId);
    }
  };

  if (entries.length === 0) {
    return (
      <Typography variant="body2" color="text.secondary" sx={{ py: 2, textAlign: 'center' }}>
        {t('employmentHistory.noEntries')}
      </Typography>
    );
  }

  return (
    <Stack spacing={1.5}>
      {entries.map((entry) => {
        const isCurrent = !entry.end_date;
        const dateRange = [entry.start_date || t('employmentHistory.unknownDate'), isCurrent ? t('employmentHistory.present') : entry.end_date]
          .join(' – ');

        return (
          <Paper
            key={entry.ID}
            variant="outlined"
            sx={{
              p: 2,
              '&:hover .action-buttons': { opacity: 1 },
            }}
          >
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
              <Box sx={{ flex: 1 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <Typography variant="subtitle1" sx={{ fontWeight: 500 }}>
                    {entry.organization}
                  </Typography>
                  {isCurrent && (
                    <Chip label={t('employmentHistory.current')} size="small" color="primary" variant="outlined" />
                  )}
                </Box>
                <Typography variant="body2" color="text.secondary">
                  {[entry.job_title, entry.department, entry.role].filter(Boolean).join(' · ')}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  {dateRange}
                </Typography>
                {entry.notes && (
                  <Typography variant="body2" sx={{ mt: 0.5, whiteSpace: 'pre-wrap' }}>
                    {entry.notes}
                  </Typography>
                )}
              </Box>
              <Box className="action-buttons" sx={{ display: 'flex', gap: 0.5, opacity: 0, transition: 'opacity 0.2s ease-in-out' }}>
                {isCurrent && (
                  <Tooltip title={t('employmentHistory.endJob')}>
                    <IconButton size="small" onClick={() => handleEndClick(entry.ID)} aria-label={t('employmentHistory.endJob')}>
                      <EventBusyIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                )}
                <IconButton size="small" onClick={() => onEdit(entry)} aria-label={t('common.edit')}>
                  <EditIcon fontSize="small" />
                </IconButton>
                <IconButton size="small" onClick={() => handleDeleteClick(entry.ID)} aria-label={t('common.delete')} color="error">
                  <DeleteIcon fontSize="small" />
                </IconButton>
              </Box>
            </Box>
          </Paper>
        );
      })}
    </Stack>
  );
}
