import { useState, useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Box,
  Typography,
  Paper,
  TextField,
  Button,
  IconButton,
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
import { useActivities } from './hooks/useActivities';
import { createActivity, updateActivity, deleteActivity, Activity } from './api/activities';
import { getContacts } from './api/contacts';
import AddActivityDialog from './components/AddActivityDialog';
import EditTimelineItemDialog from './components/EditTimelineItemDialog';
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
  const [editingActivity, setEditingActivity] = useState<Activity | null>(null);
  const [editValues, setEditValues] = useState<{
    activityTitle?: string;
    activityDescription?: string;
    activityLocation?: string;
    activityDate?: string;
    activityContacts?: Contact[];
  }>({});
  const [allContacts, setAllContacts] = useState<Contact[]>([]);

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

  const handleEditClick = async (activity: Activity) => {
    setEditingActivity(activity);
    
    // Fetch all contacts if not already loaded
    if (allContacts.length === 0) {
      try {
        const data = await getContacts({ page: 1, limit: 1000 }, token);
        setAllContacts(data.contacts || []);
      } catch (err) {
        console.error('Failed to fetch contacts:', err);
      }
    }
    
    setEditValues({
      activityTitle: activity.title || '',
      activityDescription: activity.description || '',
      activityLocation: activity.location || '',
      activityDate: activity.date ? new Date(activity.date).toISOString().split('T')[0] : '',
      activityContacts: activity.contacts || [],
    });
  };

  const handleSaveEdit = async () => {
    if (!editingActivity || !editValues.activityTitle?.trim()) return;

    try {
      await updateActivity(editingActivity.ID, {
        title: editValues.activityTitle,
        description: editValues.activityDescription || '',
        location: editValues.activityLocation || '',
        date: editValues.activityDate ? new Date(editValues.activityDate).toISOString() : new Date().toISOString(),
        contact_ids: editValues.activityContacts?.map(c => c.ID) || [],
      }, token);
      setEditingActivity(null);
      setEditValues({});
      refetch();
    } catch (err) {
      console.error('Failed to update activity:', err);
    }
  };

  const handleCancelEdit = () => {
    setEditingActivity(null);
    setEditValues({});
  };

  const handleDeleteActivity = async () => {
    if (!editingActivity) return;

    try {
      await deleteActivity(editingActivity.ID, token);
      setEditingActivity(null);
      setEditValues({});
      refetch();
    } catch (err) {
      console.error('Failed to delete activity:', err);
    }
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
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
        <Typography variant="h5">{t('activities.title')}</Typography>
        <Button variant="outlined" startIcon={<EventIcon />} onClick={handleAddActivity}>
          {t('activities.addActivity')}
        </Button>
      </Box>

      <Paper sx={{ p: 1.5, mb: 2 }}>
        <TextField
          fullWidth
          size="small"
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

      {editingActivity && (
        <EditTimelineItemDialog
          open={!!editingActivity}
          onClose={handleCancelEdit}
          onSave={handleSaveEdit}
          onDelete={handleDeleteActivity}
          type="activity"
          values={editValues}
          onChange={setEditValues}
          allContacts={allContacts}
        />
      )}
    </Box>
  );
};

export default ActivitiesPage;
