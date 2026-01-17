import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Box,
  Typography,
  Card,
  CardContent,
  IconButton,
  Chip,
  Stack,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  DialogContentText,
  Tooltip
} from '@mui/material';
import NotificationsIcon from '@mui/icons-material/Notifications';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import EmailIcon from '@mui/icons-material/Email';
import RepeatIcon from '@mui/icons-material/Repeat';
import { Reminder } from '../api/reminders';
import { useDateFormat } from '../DateFormatProvider';

interface ReminderListProps {
  reminders: Reminder[];
  onComplete: (reminderId: number) => Promise<void>;
  onEdit: (reminder: Reminder) => void;
  onDelete: (reminderId: number) => Promise<void>;
}

export default function ReminderList({
  reminders,
  onComplete,
  onEdit,
  onDelete
}: ReminderListProps) {
  const { t } = useTranslation();
  const { formatDate } = useDateFormat();
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [reminderToDelete, setReminderToDelete] = useState<number | null>(null);
  const [loading, setLoading] = useState<number | null>(null);

  const handleCompleteClick = async (reminderId: number) => {
    try {
      setLoading(reminderId);
      await onComplete(reminderId);
    } finally {
      setLoading(null);
    }
  };

  const handleDeleteClick = (reminderId: number) => {
    setReminderToDelete(reminderId);
    setDeleteDialogOpen(true);
  };

  const handleDeleteConfirm = async () => {
    if (reminderToDelete) {
      try {
        setLoading(reminderToDelete);
        await onDelete(reminderToDelete);
        setDeleteDialogOpen(false);
        setReminderToDelete(null);
      } finally {
        setLoading(null);
      }
    }
  };

  const getRecurrenceLabel = (recurrence: Reminder['recurrence']) => {
    return t(`reminders.recurrence.${recurrence}`);
  };

  const isOverdue = (dateString: string) => {
    const reminderDate = new Date(dateString);
    const today = new Date();
    today.setHours(0, 0, 0, 0);
    return reminderDate < today;
  };

  if (reminders.length === 0) {
    return (
      <Box sx={{ p: 3, textAlign: 'center' }}>
        <NotificationsIcon sx={{ fontSize: 48, color: 'text.secondary', mb: 1 }} />
        <Typography variant="body2" color="text.secondary">
          {t('reminders.noReminders')}
        </Typography>
      </Box>
    );
  }

  return (
    <>
      <Stack spacing={2}>
        {reminders.map((reminder) => (
          <Card
            key={reminder.ID}
            sx={{
              border: isOverdue(reminder.remind_at) ? '2px solid' : '1px solid',
              borderColor: isOverdue(reminder.remind_at) ? 'warning.main' : 'divider',
              '&:hover .action-buttons': {
                opacity: 1,
              },
            }}
          >
            <CardContent>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <Box sx={{ flexGrow: 1 }}>
                  <Typography variant="body1" gutterBottom>
                    {reminder.message}
                  </Typography>
                  
                  <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', mt: 1 }}>
                    <Chip
                      label={formatDate(reminder.remind_at)}
                      size="small"
                      color={isOverdue(reminder.remind_at) ? 'warning' : 'default'}
                    />
                    
                    {reminder.recurrence !== 'once' && (
                      <Chip
                        icon={<RepeatIcon />}
                        label={getRecurrenceLabel(reminder.recurrence)}
                        size="small"
                        variant="outlined"
                      />
                    )}
                    
                    {reminder.by_mail && (
                      <Chip
                        icon={<EmailIcon />}
                        label={t('reminders.email')}
                        size="small"
                        variant="outlined"
                      />
                    )}
                    
                    {reminder.reoccur_from_completion && reminder.recurrence !== 'once' && (
                      <Tooltip title={t('reminders.reoccurFromCompletionTooltip')}>
                        <Chip
                          label={t('reminders.flexible')}
                          size="small"
                          variant="outlined"
                          color="info"
                        />
                      </Tooltip>
                    )}
                  </Box>
                </Box>

                <Box sx={{ display: 'flex', gap: 0.5, ml: 1 }}>
                  <Tooltip title={t('reminders.complete')}>
                    <IconButton
                      size="small"
                      onClick={() => handleCompleteClick(reminder.ID)}
                      disabled={loading === reminder.ID}
                      color="success"
                      sx={{
                        transition: 'transform 0.15s ease-in-out, box-shadow 0.15s ease-in-out',
                        '&:hover': {
                          transform: 'scale(1.15)',
                          boxShadow: '0 0 8px rgba(76, 175, 80, 0.5)',
                        },
                      }}
                    >
                      <CheckCircleIcon />
                    </IconButton>
                  </Tooltip>
                  
                  <Box
                    className="action-buttons"
                    sx={{
                      display: 'flex',
                      gap: 0.5,
                      opacity: 0,
                      transition: 'opacity 0.2s ease-in-out',
                    }}
                  >
                    <Tooltip title={t('common.edit')}>
                      <IconButton
                        size="small"
                        onClick={() => onEdit(reminder)}
                        disabled={loading === reminder.ID}
                      >
                        <EditIcon />
                      </IconButton>
                    </Tooltip>
                    
                    <Tooltip title={t('common.delete')}>
                      <IconButton
                        size="small"
                        onClick={() => handleDeleteClick(reminder.ID)}
                        disabled={loading === reminder.ID}
                        color="error"
                      >
                        <DeleteIcon />
                      </IconButton>
                    </Tooltip>
                  </Box>
                </Box>
              </Box>
            </CardContent>
          </Card>
        ))}
      </Stack>

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteDialogOpen} onClose={() => setDeleteDialogOpen(false)}>
        <DialogTitle>{t('reminders.deleteConfirm')}</DialogTitle>
        <DialogContent>
          <DialogContentText>
            {t('reminders.deleteMessage')}
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteDialogOpen(false)} disabled={loading !== null}>
            {t('common.cancel')}
          </Button>
          <Button
            onClick={handleDeleteConfirm}
            color="error"
            variant="contained"
            disabled={loading !== null}
          >
            {t('common.delete')}
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}
