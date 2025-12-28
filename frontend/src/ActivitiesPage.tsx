import { useState, useMemo, ChangeEvent, MouseEvent } from 'react';
import { useTranslation } from 'react-i18next';
import { Link as RouterLink } from 'react-router-dom';
import {
  Box,
  Typography,
  Paper,
  TextField,
  Button,
  IconButton,
  Chip,
  Pagination,
  Popover,
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
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import { useActivities } from './hooks/useActivities';
import { useDebouncedValue } from './hooks/useDebounce';
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

  const [searchInput, setSearchInput] = useState('');
  const debouncedSearch = useDebouncedValue(searchInput, 400);
  const [page, setPage] = useState(1);
  const ACTIVITIES_PER_PAGE = 25;

  // Memoize params to prevent unnecessary refetching
  const activityParams = useMemo(
    () => ({
      includeContacts: true,
      page,
      limit: ACTIVITIES_PER_PAGE,
      search: debouncedSearch.trim() || undefined,
    }),
    [page, debouncedSearch]
  );

  const {
    activities,
    total,
    page: serverPage,
    limit,
    loading,
    refetch,
  } = useActivities(activityParams);
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
  const [infoAnchorEl, setInfoAnchorEl] = useState<HTMLElement | null>(null);
  const infoOpen = Boolean(infoAnchorEl);
  const infoPopoverId = infoOpen ? 'activities-info-popover' : undefined;

  const handleSearchChange = (value: string) => {
    setSearchInput(value);
    setPage(1);
  };

  const handlePageChange = (_: ChangeEvent<unknown>, value: number) => {
    setPage(value);
  };

  const currentPage = serverPage || page;
  const pageSize = limit || ACTIVITIES_PER_PAGE;
  const totalPages = Math.max(1, Math.ceil((total || 0) / pageSize));

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

  const handleInfoClick = (event: MouseEvent<HTMLElement>) => {
    setInfoAnchorEl(event.currentTarget);
  };

  const handleInfoClose = () => {
    setInfoAnchorEl(null);
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  };

  const isInitialLoading = loading && activities.length === 0;
  const hasSearchQuery = searchInput.trim().length > 0;

  return (
    <Box sx={{ maxWidth: 1200, mx: 'auto', mt: 2, p: 2 }}>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
        <Box display="flex" alignItems="center" gap={1}>
          <Typography variant="h5">{t('activities.title')}</Typography>
          <IconButton
            size="small"
            aria-describedby={infoPopoverId}
            onClick={handleInfoClick}
          >
            <InfoOutlinedIcon fontSize="small" />
          </IconButton>
        </Box>
        <Button variant="outlined" startIcon={<EventIcon />} onClick={handleAddActivity}>
          {t('activities.addActivity')}
        </Button>
      </Box>

      <Popover
        id={infoPopoverId}
        open={infoOpen}
        anchorEl={infoAnchorEl}
        onClose={handleInfoClose}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}
      >
        <Box sx={{ p: 2, maxWidth: 320 }}>
          <Typography variant="body2">{t('activities.allContactsInfo')}</Typography>
        </Box>
      </Popover>

      <Paper sx={{ p: 1.5, mb: 2 }}>
        <TextField
          fullWidth
          size="small"
          label={t('activities.search')}
          value={searchInput}
          onChange={(e) => handleSearchChange(e.target.value)}
          variant="outlined"
        />
      </Paper>

      {isInitialLoading ? (
        <ListSkeleton count={8} />
      ) : activities.length === 0 ? (
        <Paper sx={{ p: 4, textAlign: 'center' }}>
          <Typography variant="body1" color="text.secondary">
            {hasSearchQuery ? t('activities.noResults') : t('activities.noActivities')}
          </Typography>
        </Paper>
      ) : (
        <Timeline position="right">
          {activities.map((activity, index) => (
            <TimelineItem key={activity.ID}>
              <TimelineOppositeContent color="text.secondary" sx={{ flex: 0.2 }}>
                {formatDate(activity.date)}
              </TimelineOppositeContent>
              <TimelineSeparator>
                <TimelineDot color="primary">
                  <EventIcon />
                </TimelineDot>
                {index < activities.length - 1 && <TimelineConnector />}
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
                      <Box sx={{ flex: 1, pr: 2 }}>
                        {activity.title && (
                          <Typography
                            variant="body2"
                            fontWeight={600}
                            sx={{ wordBreak: 'break-word', color: 'text.primary' }}
                          >
                            {activity.title}
                          </Typography>
                        )}
                        <Typography variant="body2" sx={{ whiteSpace: 'pre-wrap', mt: activity.title ? 0.5 : 0 }}>
                          {activity.description}
                        </Typography>
                      </Box>
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
                            component={RouterLink}
                            to={`/contacts/${contact.ID}`}
                            clickable
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

      {totalPages > 1 && (
        <Box display="flex" justifyContent="center" mt={3}>
          <Pagination
            color="primary"
            count={totalPages}
            page={currentPage}
            onChange={handlePageChange}
            showFirstButton
            showLastButton
          />
        </Box>
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
