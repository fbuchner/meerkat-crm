import { useState, useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Box,
  Typography,
  Paper,
  TextField,
  Button,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Chip,
} from '@mui/material';
import {
  Timeline,
  TimelineItem,
  TimelineSeparator,
  TimelineConnector,
  TimelineContent,
  TimelineDot,
  TimelineOppositeContent,
} from '@mui/lab';
import EventIcon from '@mui/icons-material/Event';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import AddIcon from '@mui/icons-material/Add';
import SaveIcon from '@mui/icons-material/Save';
import CancelIcon from '@mui/icons-material/Cancel';
import { useActivities } from './hooks/useActivities';
import { createActivity, updateActivity, deleteActivity, Activity } from './api/activities';
import AddActivityDialog from './components/AddActivityDialog';
import { ListSkeleton } from './components/LoadingSkeletons';

interface Contact {
  ID: number;
  firstname: string;
  lastname: string;
  nickname?: string;
}

interface ActivitiesPageProps {
  token: string;
}

const ActivitiesPage: React.FC<ActivitiesPageProps> = ({ token }) => {
  const { t } = useTranslation();
  
  // Memoize params to prevent infinite re-renders
  const activityParams = useMemo(() => ({ includeContacts: true }), []);
  
  const { activities: allActivities, loading, refetch } = useActivities(activityParams);
  const [filteredActivities, setFilteredActivities] = useState<Activity[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [editingActivityId, setEditingActivityId] = useState<number | null>(null);
  const [editValues, setEditValues] = useState<{ description: string; date: string }>({
    description: '',
    date: '',
  });
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);
  const [activityToDelete, setActivityToDelete] = useState<number | null>(null);

  // Sort activities by date descending (newest first) and filter
  useEffect(() => {
    const sorted = [...allActivities].sort((a, b) => {
      return new Date(b.date).getTime() - new Date(a.date).getTime();
    });

    if (searchQuery.trim() === '') {
      setFilteredActivities(sorted);
    } else {
      const query = searchQuery.toLowerCase();
      const filtered = sorted.filter((activity) => {
        const descriptionMatch = activity.description?.toLowerCase().includes(query);
        const contactMatch = activity.contacts?.some((contact: Contact) =>
          `${contact.firstname} ${contact.lastname}`.toLowerCase().includes(query)
        );
        return descriptionMatch || contactMatch;
      });
      setFilteredActivities(filtered);
    }
  }, [searchQuery, allActivities]);

  const handleAddActivity = () => {
    setAddDialogOpen(true);
  };

  const handleActivitySave = async (activity: {
    title: string;
    description: string;
    location: string;
    date: string;
    contact_ids: number[];
  }) => {
    try {
      await createActivity({
        ...activity,
        date: new Date(activity.date).toISOString()
      }, token);
      setAddDialogOpen(false);
      refetch();
    } catch (err) {
      console.error('Failed to create activity:', err);
      throw err;
    }
  };

  const handleEditClick = (activity: Activity) => {
    setEditingActivityId(activity.ID);
    setEditValues({
      description: activity.description || '',
      date: activity.date ? new Date(activity.date).toISOString().split('T')[0] : '',
    });
  };

  const handleSaveEdit = async (activityId: number) => {
    try {
      await updateActivity(activityId, {
        description: editValues.description,
        date: new Date(editValues.date).toISOString(),
      }, token);
      setEditingActivityId(null);
      refetch();
    } catch (err) {
      console.error('Failed to update activity:', err);
    }
  };

  const handleCancelEdit = () => {
    setEditingActivityId(null);
    setEditValues({ description: '', date: '' });
  };

  const handleDeleteClick = (activityId: number) => {
    setActivityToDelete(activityId);
    setDeleteConfirmOpen(true);
  };

  const handleConfirmDelete = async () => {
    if (!activityToDelete) return;

    try {
      await deleteActivity(activityToDelete, token);
      setDeleteConfirmOpen(false);
      setActivityToDelete(null);
      refetch();
    } catch (err) {
      console.error('Failed to delete activity:', err);
    }
  };

  const handleCancelDelete = () => {
    setDeleteConfirmOpen(false);
    setActivityToDelete(null);
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  };

  if (loading) {
    return (
      <Box>
        <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
          <Typography variant="h4">{t('activities.title')}</Typography>
        </Box>
        <ListSkeleton count={8} />
      </Box>
    );
  }

  return (
    <Box>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Typography variant="h4">{t('activities.title')}</Typography>
        <Button variant="contained" startIcon={<AddIcon />} onClick={handleAddActivity}>
          {t('activities.addActivity')}
        </Button>
      </Box>

      <Paper sx={{ p: 2, mb: 3 }}>
        <TextField
          fullWidth
          label={t('activities.search')}
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          variant="outlined"
        />
      </Paper>

      {filteredActivities.length === 0 ? (
        <Paper sx={{ p: 4, textAlign: 'center' }}>
          <Typography variant="body1" color="text.secondary">
            {searchQuery ? t('activities.noResults') : t('activities.noActivities')}
          </Typography>
        </Paper>
      ) : (
        <Timeline position="right">
          {filteredActivities.map((activity, index) => (
            <TimelineItem key={activity.ID}>
              <TimelineOppositeContent color="text.secondary" sx={{ flex: 0.2 }}>
                {formatDate(activity.date)}
              </TimelineOppositeContent>
              <TimelineSeparator>
                <TimelineDot color="primary">
                  <EventIcon />
                </TimelineDot>
                {index < filteredActivities.length - 1 && <TimelineConnector />}
              </TimelineSeparator>
              <TimelineContent sx={{ flex: 0.8 }}>
                <Paper
                  elevation={2}
                  sx={{
                    p: 2,
                    '&:hover .edit-actions': {
                      opacity: 1,
                    },
                  }}
                >
                  {editingActivityId === activity.ID ? (
                    <Box>
                      <TextField
                        fullWidth
                        multiline
                        rows={3}
                        value={editValues.description}
                        onChange={(e) =>
                          setEditValues({ ...editValues, description: e.target.value })
                        }
                        sx={{ mb: 2 }}
                      />
                      <TextField
                        fullWidth
                        type="date"
                        label={t('activities.date')}
                        value={editValues.date}
                        onChange={(e) => setEditValues({ ...editValues, date: e.target.value })}
                        InputLabelProps={{ shrink: true }}
                        sx={{ mb: 2 }}
                      />
                      <Box display="flex" gap={1}>
                        <Button
                          size="small"
                          variant="contained"
                          startIcon={<SaveIcon />}
                          onClick={() => handleSaveEdit(activity.ID)}
                        >
                          {t('common.save')}
                        </Button>
                        <Button
                          size="small"
                          variant="outlined"
                          startIcon={<CancelIcon />}
                          onClick={handleCancelEdit}
                        >
                          {t('common.cancel')}
                        </Button>
                      </Box>
                    </Box>
                  ) : (
                    <Box>
                      <Box display="flex" justifyContent="space-between" alignItems="flex-start">
                        <Typography variant="body1" sx={{ whiteSpace: 'pre-wrap', flex: 1 }}>
                          {activity.description}
                        </Typography>
                        <Box
                          className="edit-actions"
                          sx={{ opacity: 0, transition: 'opacity 0.2s', display: 'flex', gap: 1 }}
                        >
                          <IconButton size="small" onClick={() => handleEditClick(activity)}>
                            <EditIcon fontSize="small" />
                          </IconButton>
                          <IconButton
                            size="small"
                            color="error"
                            onClick={() => handleDeleteClick(activity.ID)}
                          >
                            <DeleteIcon fontSize="small" />
                          </IconButton>
                        </Box>
                      </Box>
                      {activity.contacts && activity.contacts.length > 0 && (
                        <Box mt={1} display="flex" flexWrap="wrap" gap={0.5}>
                          {activity.contacts.map((contact: Contact) => (
                            <Chip
                              key={contact.ID}
                              label={`${contact.firstname} ${contact.lastname}`}
                              size="small"
                              variant="outlined"
                            />
                          ))}
                        </Box>
                      )}
                    </Box>
                  )}
                </Paper>
              </TimelineContent>
            </TimelineItem>
          ))}
        </Timeline>
      )}

      <AddActivityDialog
        open={addDialogOpen}
        onClose={() => setAddDialogOpen(false)}
        onSave={handleActivitySave}
        token={token}
      />

      <Dialog open={deleteConfirmOpen} onClose={handleCancelDelete}>
        <DialogTitle>{t('activities.deleteConfirm')}</DialogTitle>
        <DialogContent>
          <Typography>{t('activities.deleteMessage')}</Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCancelDelete}>{t('common.cancel')}</Button>
          <Button onClick={handleConfirmDelete} color="error" variant="contained">
            {t('common.delete')}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default ActivitiesPage;
