import React, { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  getContact,
  updateContact,
  getContactProfilePicture,
  getContacts
} from './api/contacts';
import { 
  getContactNotes, 
  createNote, 
  updateNote, 
  deleteNote,
  Note 
} from './api/notes';
import {
  getContactActivities,
  createActivity,
  updateActivity,
  deleteActivity,
  Activity
} from './api/activities';
import {
  Box,
  Card,
  CardContent,
  Avatar,
  Typography,
  Chip,
  CircularProgress,
  IconButton,
  Divider,
  Stack,
  Paper,
  TextField,
  SpeedDial,
  SpeedDialAction,
  SpeedDialIcon,
  Autocomplete,
  Link
} from '@mui/material';
import {
  Timeline,
  TimelineItem,
  TimelineSeparator,
  TimelineConnector,
  TimelineContent,
  TimelineDot,
  TimelineOppositeContent
} from '@mui/lab';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import EmailIcon from '@mui/icons-material/Email';
import PhoneIcon from '@mui/icons-material/Phone';
import CakeIcon from '@mui/icons-material/Cake';
import HomeIcon from '@mui/icons-material/Home';
import WorkIcon from '@mui/icons-material/Work';
import RestaurantIcon from '@mui/icons-material/Restaurant';
import PeopleIcon from '@mui/icons-material/People';
import NoteIcon from '@mui/icons-material/Note';
import EventIcon from '@mui/icons-material/Event';
import EditIcon from '@mui/icons-material/Edit';
import SaveIcon from '@mui/icons-material/Save';
import CloseIcon from '@mui/icons-material/Close';
import DeleteIcon from '@mui/icons-material/Delete';
import AddIcon from '@mui/icons-material/Add';
import AddNoteDialog from './components/AddNoteDialog';
import AddActivityDialog from './components/AddActivityDialog';

interface Contact {
  ID: number;
  firstname: string;
  lastname: string;
  nickname?: string;
  gender?: string;
  email?: string;
  phone?: string;
  birthday?: string;
  address?: string;
  how_we_met?: string;
  food_preference?: string;
  work_information?: string;
  contact_information?: string;
  circles?: string[];
  notes?: Note[];
  activities?: Activity[];
}

export default function ContactDetailPage({ token }: { token: string }) {
  const { t } = useTranslation();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [contact, setContact] = useState<Contact | null>(null);
  const [profilePic, setProfilePic] = useState<string>('');
  const [loading, setLoading] = useState(true);
  const [editingField, setEditingField] = useState<string | null>(null);
  const [editValue, setEditValue] = useState<string>('');
  const [validationError, setValidationError] = useState<string>('');
  const [noteDialogOpen, setNoteDialogOpen] = useState(false);
  const [activityDialogOpen, setActivityDialogOpen] = useState(false);
  
  // Timeline item editing state
  const [editingTimelineItem, setEditingTimelineItem] = useState<{ type: 'note' | 'activity'; id: number } | null>(null);
  const [editTimelineValues, setEditTimelineValues] = useState<{
    noteContent?: string;
    noteDate?: string;
    activityTitle?: string;
    activityDescription?: string;
    activityLocation?: string;
    activityDate?: string;
    activityContacts?: { ID: number; firstname: string; lastname: string; nickname?: string }[];
  }>({});
  const [allContacts, setAllContacts] = useState<{ ID: number; firstname: string; lastname: string; nickname?: string }[]>([]);
  const [notes, setNotes] = useState<Note[]>([]);
  const [activities, setActivities] = useState<Activity[]>([]);
  
  // Circle editing state
  const [editingCircles, setEditingCircles] = useState(false);
  const [newCircleName, setNewCircleName] = useState('');
  
  // Profile editing state
  const [editingProfile, setEditingProfile] = useState(false);
  const [profileValues, setProfileValues] = useState({
    firstname: '',
    lastname: '',
    nickname: '',
    gender: ''
  });

  // Fetch contact details, notes, and activities
  useEffect(() => {
    if (!id) return;

    const fetchData = async () => {
      try {
        // Fetch contact details
        const contactData = await getContact(id, token);
        setContact(contactData);

        // Fetch detailed notes
        const notesData = await getContactNotes(id, token);
        setNotes(notesData.notes || []);

        // Fetch detailed activities
        const activitiesData = await getContactActivities(id, token);
        setActivities(activitiesData.activities || []);

        setLoading(false);
      } catch (err) {
        console.error('Error fetching data:', err);
        setLoading(false);
      }
    };

    fetchData();

    // Fetch profile picture
    getContactProfilePicture(id, token)
      .then(blob => {
        if (blob) {
          setProfilePic(URL.createObjectURL(blob));
        }
      })
      .catch(err => console.error('Error fetching profile picture:', err));
  }, [id, token]);

  // Unified refresh function for notes and activities
  const refreshNotesAndActivities = async () => {
    if (!id) return;

    try {
      // Fetch detailed notes
      const notesData = await getContactNotes(id, token);
      setNotes(notesData.notes || []);

      // Fetch detailed activities
      const activitiesData = await getContactActivities(id, token);
      setActivities(activitiesData.activities || []);
    } catch (err) {
      console.error('Error refreshing notes and activities:', err);
    }
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', mt: 4 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (!contact) {
    return (
      <Box sx={{ maxWidth: 800, mx: 'auto', mt: 4 }}>
        <Typography variant="h6">{t('contactDetail.notFound')}</Typography>
      </Box>
    );
  }

  // Combine and sort notes and activities for timeline
  const timelineItems: Array<{ type: 'note' | 'activity'; data: Note | Activity; date: string }> = [
    ...notes.map(note => ({
      type: 'note' as const,
      data: note,
      date: note.date || note.CreatedAt
    })),
    ...activities.map(activity => ({
      type: 'activity' as const,
      data: activity,
      date: activity.date || activity.CreatedAt
    }))
  ].sort((a, b) => new Date(b.date).getTime() - new Date(a.date).getTime());

  const validateBirthday = (value: string): boolean => {
    if (!value || value.trim() === '') return true; // Empty is valid
    
    // Format: DD.MM. or DD.MM.YYYY
    // DD: 01-31, MM: 01-12, YYYY: 4 digits
    const regex = /^(0[1-9]|[12][0-9]|3[01])\.(0[1-9]|1[0-2])(\.|(\.\d{4}))$/;
    return regex.test(value.trim());
  };

  const handleEditStart = (field: string, currentValue: string) => {
    setEditingField(field);
    setEditValue(currentValue || '');
    setValidationError('');
  };

  const handleEditCancel = () => {
    setEditingField(null);
    setEditValue('');
    setValidationError('');
  };

  const handleEditSave = async (field: string) => {
    if (!contact) return;

    // Validate birthday if editing birthday field
    if (field === 'birthday' && !validateBirthday(editValue)) {
      setValidationError(t('contactDetail.birthdayError'));
      return;
    }

    try {
      const updatedContact = await updateContact(id!, {
        ...contact,
        [field]: editValue
      }, token);
      setContact(updatedContact);
      setEditingField(null);
      setEditValue('');
      setValidationError('');
    } catch (err) {
      console.error('Error updating contact:', err);
    }
  };

  const handleSaveNote = async (content: string, date: string) => {
    if (!id) return;

    try {
      await createNote(id, {
        content,
        date: new Date(date).toISOString()
      }, token);
      // Refresh notes and activities
      await refreshNotesAndActivities();
    } catch (err) {
      console.error('Failed to save note:', err);
      throw new Error('Failed to save note');
    }
  };

  const handleSaveActivity = async (activity: {
    title: string;
    description: string;
    location: string;
    date: string;
    contact_ids: number[];
  }) => {
    try {
      await createActivity({
        title: activity.title,
        description: activity.description,
        location: activity.location,
        date: new Date(activity.date).toISOString(),
        contact_ids: activity.contact_ids
      }, token);
      // Refresh notes and activities
      await refreshNotesAndActivities();
    } catch (err) {
      console.error('Failed to save activity:', err);
      throw new Error('Failed to save activity');
    }
  };

  const handleStartEditTimelineItem = async (type: 'note' | 'activity', item: Note | Activity) => {
    setEditingTimelineItem({ type, id: item.ID });
    
    if (type === 'note') {
      const note = item as Note;
      setEditTimelineValues({
        noteContent: note.content,
        noteDate: note.date ? new Date(note.date).toISOString().split('T')[0] : ''
      });
    } else {
      const activity = item as Activity;
      
      // Fetch all contacts for the autocomplete if not already loaded
      if (allContacts.length === 0) {
        try {
          const data = await getContacts({ page: 1, limit: 1000 }, token);
          setAllContacts(data.contacts || []);
        } catch (err) {
          console.error('Failed to fetch contacts:', err);
        }
      }
      
      setEditTimelineValues({
        activityTitle: activity.title,
        activityDescription: activity.description || '',
        activityLocation: activity.location || '',
        activityDate: activity.date ? new Date(activity.date).toISOString().split('T')[0] : '',
        activityContacts: activity.contacts || []
      });
    }
  };

  const handleCancelEditTimelineItem = () => {
    setEditingTimelineItem(null);
    setEditTimelineValues({});
  };

  const handleUpdateNote = async (noteId: number) => {
    if (!editTimelineValues.noteContent?.trim()) return;

    try {
      await updateNote(noteId, {
        content: editTimelineValues.noteContent,
        date: editTimelineValues.noteDate ? new Date(editTimelineValues.noteDate).toISOString() : new Date().toISOString(),
        contact_id: contact?.ID
      }, token);
      // Refresh notes and activities
      await refreshNotesAndActivities();
      handleCancelEditTimelineItem();
    } catch (err) {
      console.error('Error updating note:', err);
    }
  };

  const handleUpdateActivity = async (activityId: number) => {
    if (!editTimelineValues.activityTitle?.trim()) return;

    try {
      await updateActivity(activityId, {
        title: editTimelineValues.activityTitle,
        description: editTimelineValues.activityDescription || '',
        location: editTimelineValues.activityLocation || '',
        date: editTimelineValues.activityDate ? new Date(editTimelineValues.activityDate).toISOString() : new Date().toISOString(),
        contact_ids: editTimelineValues.activityContacts?.map(c => c.ID) || []
      }, token);
      // Refresh notes and activities
      await refreshNotesAndActivities();
      handleCancelEditTimelineItem();
    } catch (err) {
      console.error('Error updating activity:', err);
    }
  };

  const handleDeleteNote = async (noteId: number) => {
    if (!window.confirm(t('contactDetail.confirmDelete'))) {
      return;
    }

    try {
      await deleteNote(noteId, token);
      // Refresh notes and activities
      await refreshNotesAndActivities();
      handleCancelEditTimelineItem();
    } catch (err) {
      console.error('Error deleting note:', err);
    }
  };

  const handleDeleteActivity = async (activityId: number) => {
    if (!window.confirm(t('contactDetail.confirmDelete'))) {
      return;
    }

    try {
      await deleteActivity(activityId, token);
      // Refresh notes and activities
      await refreshNotesAndActivities();
      handleCancelEditTimelineItem();
    } catch (err) {
      console.error('Error deleting activity:', err);
    }
  };

  const handleAddCircle = async () => {
    if (!contact || !newCircleName.trim()) return;

    const updatedCircles = [...(contact.circles || []), newCircleName.trim()];

    try {
      const updatedContact = await updateContact(id!, {
        ...contact,
        circles: updatedCircles
      }, token);
      setContact(updatedContact);
      setNewCircleName('');
    } catch (err) {
      console.error('Error adding circle:', err);
    }
  };

  const handleDeleteCircle = async (circleToDelete: string) => {
    if (!contact) return;

    const updatedCircles = (contact.circles || []).filter(circle => circle !== circleToDelete);

    try {
      const updatedContact = await updateContact(id!, {
        ...contact,
        circles: updatedCircles
      }, token);
      setContact(updatedContact);
    } catch (err) {
      console.error('Error deleting circle:', err);
    }
  };

  const handleStartEditProfile = () => {
    if (!contact) return;
    setProfileValues({
      firstname: contact.firstname || '',
      lastname: contact.lastname || '',
      nickname: contact.nickname || '',
      gender: contact.gender || ''
    });
    setEditingProfile(true);
  };

  const handleCancelEditProfile = () => {
    setEditingProfile(false);
    setProfileValues({ firstname: '', lastname: '', nickname: '', gender: '' });
  };

  const handleSaveProfile = async () => {
    if (!contact || !profileValues.firstname.trim() || !profileValues.lastname.trim()) {
      alert('First name and last name are required');
      return;
    }

    try {
      const updatedContact = await updateContact(id!, {
        ...contact,
        firstname: profileValues.firstname.trim(),
        lastname: profileValues.lastname.trim(),
        nickname: profileValues.nickname.trim(),
        gender: profileValues.gender
      }, token);
      setContact(updatedContact);
      setEditingProfile(false);
    } catch (err) {
      console.error('Error updating profile:', err);
    }
  };

  // Reusable EditableField component
  const EditableField = ({ 
    icon, 
    label, 
    field, 
    value, 
    multiline = false,
    placeholder = ''
  }: { 
    icon: React.ReactNode; 
    label: string; 
    field: string; 
    value: string; 
    multiline?: boolean;
    placeholder?: string;
  }) => {
    const isEditing = editingField === field;
    const displayValue = value || '-';
    const showError = isEditing && validationError && editingField === field;

    return (
      <Box 
        sx={{ 
          position: 'relative',
          '&:hover .edit-icon': {
            opacity: 1
          }
        }}
      >
        <Box sx={{ display: 'flex', alignItems: multiline ? 'flex-start' : 'center' }}>
          {icon}
          <Box sx={{ flex: 1 }}>
            <Typography variant="caption" color="text.secondary">
              {label}
            </Typography>
            {isEditing ? (
              <Box>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 0.5 }}>
                  <TextField
                    value={editValue}
                    onChange={(e) => {
                      setEditValue(e.target.value);
                      setValidationError('');
                    }}
                    size="small"
                    fullWidth
                    multiline={multiline}
                    rows={multiline ? 3 : 1}
                    autoFocus
                    error={!!showError}
                    placeholder={placeholder}
                  />
                  <IconButton size="small" color="primary" onClick={() => handleEditSave(field)}>
                    <SaveIcon fontSize="small" />
                  </IconButton>
                  <IconButton size="small" onClick={handleEditCancel}>
                    <CloseIcon fontSize="small" />
                  </IconButton>
                </Box>
                {showError && (
                  <Typography variant="caption" color="error" sx={{ mt: 0.5, display: 'block' }}>
                    {validationError}
                  </Typography>
                )}
              </Box>
            ) : (
              <Typography variant="body1" sx={{ wordBreak: 'break-word' }}>
                {displayValue}
              </Typography>
            )}
          </Box>
          {!isEditing && (
            <IconButton
              className="edit-icon"
              size="small"
              onClick={() => handleEditStart(field, value)}
              sx={{ 
                opacity: 0, 
                transition: 'opacity 0.2s',
                ml: 1
              }}
            >
              <EditIcon fontSize="small" />
            </IconButton>
          )}
        </Box>
      </Box>
    );
  };

  return (
    <Box sx={{ maxWidth: 1000, mx: 'auto', mt: 4, p: 2 }}>
      {/* Header with back button */}
      <Box sx={{ mb: 3 }}>
        <IconButton onClick={() => navigate('/contacts')} sx={{ mr: 2 }}>
          <ArrowBackIcon />
        </IconButton>
      </Box>

      {/* Contact Header Card */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Box sx={{ display: 'flex', alignItems: 'flex-start', mb: 2 }}>
            <Avatar
              src={profilePic || undefined}
              sx={{ width: 100, height: 100, mr: 3 }}
            />
            <Box sx={{ flex: 1 }}>
              {editingProfile ? (
                // Edit Mode
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                  <TextField
                    label={t('contactDetail.firstname')}
                    value={profileValues.firstname}
                    onChange={(e) => setProfileValues({ ...profileValues, firstname: e.target.value })}
                    size="small"
                    required
                    autoFocus
                  />
                  <TextField
                    label={t('contactDetail.lastname')}
                    value={profileValues.lastname}
                    onChange={(e) => setProfileValues({ ...profileValues, lastname: e.target.value })}
                    size="small"
                    required
                  />
                  <TextField
                    label={t('contactDetail.nickname')}
                    value={profileValues.nickname}
                    onChange={(e) => setProfileValues({ ...profileValues, nickname: e.target.value })}
                    size="small"
                  />
                  <TextField
                    select
                    label={t('contactDetail.gender')}
                    value={profileValues.gender}
                    onChange={(e) => setProfileValues({ ...profileValues, gender: e.target.value })}
                    size="small"
                    SelectProps={{ native: true }}
                  >
                    <option value=""></option>
                    <option value="Male">{t('contactDetail.male')}</option>
                    <option value="Female">{t('contactDetail.female')}</option>
                    <option value="Other">{t('contactDetail.other')}</option>
                  </TextField>
                  <Box sx={{ display: 'flex', gap: 1 }}>
                    <IconButton size="small" color="primary" onClick={handleSaveProfile}>
                      <SaveIcon />
                    </IconButton>
                    <IconButton size="small" onClick={handleCancelEditProfile}>
                      <CloseIcon />
                    </IconButton>
                  </Box>
                </Box>
              ) : (
                // View Mode
                <>
                  <Box 
                    sx={{ 
                      display: 'flex', 
                      alignItems: 'center',
                      '&:hover .edit-icon': {
                        opacity: 1
                      }
                    }}
                  >
                    <Typography variant="h4" sx={{ fontWeight: 500 }}>
                      {contact.firstname} {contact.nickname && `"${contact.nickname}"`} {contact.lastname}
                    </Typography>
                    <IconButton 
                      className="edit-icon"
                      size="small" 
                      onClick={handleStartEditProfile} 
                      sx={{ 
                        ml: 2,
                        opacity: 0,
                        transition: 'opacity 0.2s'
                      }}
                    >
                      <EditIcon />
                    </IconButton>
                  </Box>
                  {contact.gender && (
                    <Typography variant="body1" color="text.secondary" sx={{ mt: 1 }}>
                      {contact.gender}
                    </Typography>
                  )}
                </>
              )}
              
              {/* Circles Section */}
              <Box 
                sx={{ 
                  mt: 2,
                  '&:hover .edit-icon': {
                    opacity: 1
                  }
                }}
              >
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                  <Typography variant="caption" color="text.secondary">
                    {t('contactDetail.circles')}
                  </Typography>
                  <IconButton 
                    className="edit-icon"
                    size="small" 
                    onClick={() => setEditingCircles(!editingCircles)}
                    sx={{ 
                      ml: 'auto',
                      opacity: 0,
                      transition: 'opacity 0.2s'
                    }}
                  >
                    <EditIcon fontSize="small" />
                  </IconButton>
                </Box>
                
                {editingCircles ? (
                  // Edit Mode
                  <Box>
                    <Stack direction="row" spacing={1} flexWrap="wrap" sx={{ gap: 1, mb: 1 }}>
                      {contact.circles && contact.circles.length > 0 ? (
                        contact.circles.map((circle, index) => (
                          <Chip 
                            key={index} 
                            label={circle} 
                            size="small" 
                            color="primary"
                            onDelete={() => handleDeleteCircle(circle)}
                          />
                        ))
                      ) : (
                        <Typography variant="caption" color="text.secondary">
                          {t('contactDetail.noCircles')}
                        </Typography>
                      )}
                    </Stack>
                    <Box sx={{ display: 'flex', gap: 1, mt: 1 }}>
                      <TextField
                        size="small"
                        placeholder={t('contactDetail.newCircle')}
                        value={newCircleName}
                        onChange={(e) => setNewCircleName(e.target.value)}
                        onKeyPress={(e) => {
                          if (e.key === 'Enter') {
                            handleAddCircle();
                          }
                        }}
                        sx={{ flexGrow: 1 }}
                      />
                      <IconButton 
                        size="small" 
                        color="primary"
                        onClick={handleAddCircle}
                        disabled={!newCircleName.trim()}
                      >
                        <AddIcon />
                      </IconButton>
                    </Box>
                  </Box>
                ) : (
                  // View Mode
                  <Stack direction="row" spacing={1} flexWrap="wrap" sx={{ gap: 1 }}>
                    {contact.circles && contact.circles.length > 0 ? (
                      contact.circles.map((circle, index) => (
                        <Chip key={index} label={circle} size="small" color="primary" />
                      ))
                    ) : (
                      <Typography variant="caption" color="text.secondary">
                        {t('contactDetail.noCircles')}
                      </Typography>
                    )}
                  </Stack>
                )}
              </Box>
            </Box>
          </Box>
        </CardContent>
      </Card>

      {/* General Information and Timeline - Two Column Layout */}
      <Box sx={{ 
        display: 'flex', 
        flexDirection: { xs: 'column', md: 'row' }, 
        gap: 3 
      }}>
        {/* General Information */}
        <Card sx={{ flex: 1 }}>
          <CardContent>
            <Typography variant="h5" sx={{ mb: 2, fontWeight: 500 }}>
              {t('contactDetail.generalInfo')}
            </Typography>
            <Divider sx={{ mb: 2 }} />
          
          <Stack spacing={3}>
            <EditableField
              icon={<EmailIcon sx={{ mr: 1, color: 'text.secondary' }} />}
              label={t('contactDetail.email')}
              field="email"
              value={contact.email || ''}
            />

            <EditableField
              icon={<PhoneIcon sx={{ mr: 1, color: 'text.secondary' }} />}
              label={t('contactDetail.phone')}
              field="phone"
              value={contact.phone || ''}
            />

            <EditableField
              icon={<CakeIcon sx={{ mr: 1, color: 'text.secondary' }} />}
              label={t('contactDetail.birthday')}
              field="birthday"
              value={contact.birthday || ''}
              placeholder="DD.MM. or DD.MM.YYYY"
            />

            <EditableField
              icon={<HomeIcon sx={{ mr: 1, color: 'text.secondary' }} />}
              label={t('contactDetail.address')}
              field="address"
              value={contact.address || ''}
              multiline
            />

            <EditableField
              icon={<WorkIcon sx={{ mr: 1, mt: 0.5, color: 'text.secondary' }} />}
              label={t('contactDetail.workInfo')}
              field="work_information"
              value={contact.work_information || ''}
              multiline
            />

            <EditableField
              icon={<RestaurantIcon sx={{ mr: 1, mt: 0.5, color: 'text.secondary' }} />}
              label={t('contactDetail.foodPreferences')}
              field="food_preference"
              value={contact.food_preference || ''}
              multiline
            />

            <EditableField
              icon={<PeopleIcon sx={{ mr: 1, mt: 0.5, color: 'text.secondary' }} />}
              label={t('contactDetail.howWeMet')}
              field="how_we_met"
              value={contact.how_we_met || ''}
              multiline
            />

            <EditableField
              icon={null}
              label={t('contactDetail.additionalInfo')}
              field="contact_information"
              value={contact.contact_information || ''}
              multiline
            />
          </Stack>
          </CardContent>
        </Card>

        {/* Timeline - Notes and Activities */}
        <Card sx={{ flex: 1 }}>
        <CardContent>
          <Typography variant="h5" sx={{ mb: 2, fontWeight: 500 }}>
            {t('contactDetail.timeline')}
          </Typography>
          <Divider sx={{ mb: 3 }} />
          
          {timelineItems.length === 0 ? (
            <Typography variant="body2" color="text.secondary">
              {t('contactDetail.noActivity')}
            </Typography>
          ) : (
            <Timeline position="right">
              {timelineItems.map((item, index) => {
                const itemDate = new Date(item.date);
                const isValidDate = !isNaN(itemDate.getTime());
                const isEditing = editingTimelineItem?.type === item.type && editingTimelineItem?.id === item.data.ID;
                
                return (
                <TimelineItem key={`${item.type}-${item.data.ID}`}>
                  <TimelineOppositeContent color="text.secondary" sx={{ flex: 0.3 }}>
                    {isEditing ? (
                      <TextField
                        type="date"
                        value={item.type === 'note' ? editTimelineValues.noteDate : editTimelineValues.activityDate}
                        onChange={(e) => setEditTimelineValues({
                          ...editTimelineValues,
                          ...(item.type === 'note' ? { noteDate: e.target.value } : { activityDate: e.target.value })
                        })}
                        size="small"
                        sx={{ width: '100%' }}
                        InputLabelProps={{ shrink: true }}
                      />
                    ) : (
                      <Typography variant="caption">
                        {isValidDate ? itemDate.toLocaleDateString() : (item.date || 'N/A')}
                      </Typography>
                    )}
                  </TimelineOppositeContent>
                  <TimelineSeparator>
                    <TimelineDot color={item.type === 'note' ? 'primary' : 'secondary'}>
                      {item.type === 'note' ? <NoteIcon fontSize="small" /> : <EventIcon fontSize="small" />}
                    </TimelineDot>
                    {index < timelineItems.length - 1 && <TimelineConnector />}
                  </TimelineSeparator>
                  <TimelineContent>
                    <Paper 
                      elevation={2} 
                      sx={{ 
                        p: 2,
                        position: 'relative',
                        '&:hover .edit-icon': {
                          opacity: 1
                        }
                      }}
                    >
                      {isEditing ? (
                        // Edit Mode
                        <Box>
                          {item.type === 'note' ? (
                            // Edit Note
                            <TextField
                              fullWidth
                              multiline
                              rows={3}
                              value={editTimelineValues.noteContent}
                              onChange={(e) => setEditTimelineValues({
                                ...editTimelineValues,
                                noteContent: e.target.value
                              })}
                              size="small"
                              autoFocus
                              placeholder={t('noteDialog.contentPlaceholder')}
                            />
                          ) : (
                            // Edit Activity
                            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
                              <TextField
                                fullWidth
                                value={editTimelineValues.activityTitle}
                                onChange={(e) => setEditTimelineValues({
                                  ...editTimelineValues,
                                  activityTitle: e.target.value
                                })}
                                size="small"
                                label={t('activityDialog.activityTitle')}
                                autoFocus
                              />
                              <TextField
                                fullWidth
                                multiline
                                rows={2}
                                value={editTimelineValues.activityDescription}
                                onChange={(e) => setEditTimelineValues({
                                  ...editTimelineValues,
                                  activityDescription: e.target.value
                                })}
                                size="small"
                                label={t('activityDialog.description')}
                              />
                              <TextField
                                fullWidth
                                value={editTimelineValues.activityLocation}
                                onChange={(e) => setEditTimelineValues({
                                  ...editTimelineValues,
                                  activityLocation: e.target.value
                                })}
                                size="small"
                                label={t('activityDialog.location')}
                              />
                              <Autocomplete
                                multiple
                                options={allContacts}
                                getOptionLabel={(contact) => `${contact.firstname}${contact.nickname ? ` "${contact.nickname}"` : ''} ${contact.lastname}`}
                                value={editTimelineValues.activityContacts || []}
                                onChange={(_, newValue) => setEditTimelineValues({
                                  ...editTimelineValues,
                                  activityContacts: newValue
                                })}
                                renderInput={(params) => (
                                  <TextField
                                    {...params}
                                    label={t('activityDialog.contacts')}
                                    placeholder={t('activityDialog.selectContacts')}
                                    size="small"
                                  />
                                )}
                                renderTags={(value, getTagProps) =>
                                  value.map((contact, index) => (
                                    <Chip
                                      label={`${contact.firstname}${contact.nickname ? ` "${contact.nickname}"` : ''} ${contact.lastname}`}
                                      {...getTagProps({ index })}
                                      size="small"
                                      key={contact.ID}
                                    />
                                  ))
                                }
                              />
                            </Box>
                          )}
                          
                          <Box sx={{ mt: 2, display: 'flex', gap: 1, justifyContent: 'space-between' }}>
                            <IconButton 
                              size="small" 
                              color="error"
                              onClick={() => item.type === 'note' 
                                ? handleDeleteNote(item.data.ID) 
                                : handleDeleteActivity(item.data.ID)
                              }
                              title={t('contactDetail.delete')}
                            >
                              <DeleteIcon fontSize="small" />
                            </IconButton>
                            <Box sx={{ display: 'flex', gap: 1 }}>
                              <IconButton 
                                size="small" 
                                color="primary"
                                onClick={() => item.type === 'note' 
                                  ? handleUpdateNote(item.data.ID) 
                                  : handleUpdateActivity(item.data.ID)
                                }
                                title={t('contactDetail.save')}
                              >
                                <SaveIcon fontSize="small" />
                              </IconButton>
                              <IconButton 
                                size="small"
                                onClick={handleCancelEditTimelineItem}
                                title={t('contactDetail.cancel')}
                              >
                                <CloseIcon fontSize="small" />
                              </IconButton>
                            </Box>
                          </Box>
                        </Box>
                      ) : (
                        // View Mode
                        <>
                          <Typography variant="subtitle2" sx={{ fontWeight: 500 }}>
                            {item.type === 'note' ? t('contactDetail.note') : (item.data as Activity).title}
                          </Typography>
                          <Typography variant="body2" sx={{ mt: 1 }}>
                            {item.type === 'note' 
                              ? (item.data as Note).content 
                              : (item.data as Activity).description || t('contactDetail.noDescription')}
                          </Typography>
                          {item.type === 'activity' && (item.data as Activity).location && (
                            <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: 'block' }}>
                              📍 {(item.data as Activity).location}
                            </Typography>
                          )}
                          {item.type === 'activity' && (item.data as Activity).contacts && (item.data as Activity).contacts!.length > 0 && (
                            <Box sx={{ mt: 1.5, display: 'flex', flexWrap: 'wrap', gap: 0.5, alignItems: 'center' }}>
                              <Typography variant="caption" color="text.secondary" sx={{ mr: 0.5 }}>
                                👥
                              </Typography>
                              {(item.data as Activity).contacts!.map((activityContact, idx) => (
                                <React.Fragment key={activityContact.ID}>
                                  <Link
                                    component="button"
                                    variant="caption"
                                    onClick={(e) => {
                                      e.preventDefault();
                                      navigate(`/contacts/${activityContact.ID}`);
                                    }}
                                    sx={{ 
                                      cursor: 'pointer',
                                      textDecoration: 'none',
                                      '&:hover': { textDecoration: 'underline' }
                                    }}
                                  >
                                    {activityContact.firstname}
                                    {activityContact.nickname ? ` "${activityContact.nickname}"` : ''} 
                                    {activityContact.lastname}
                                  </Link>
                                  {idx < (item.data as Activity).contacts!.length - 1 && (
                                    <Typography variant="caption" color="text.secondary">, </Typography>
                                  )}
                                </React.Fragment>
                              ))}
                            </Box>
                          )}
                          
                          <IconButton
                            className="edit-icon"
                            size="small"
                            onClick={() => handleStartEditTimelineItem(item.type, item.data)}
                            sx={{
                              position: 'absolute',
                              top: 8,
                              right: 8,
                              opacity: 0,
                              transition: 'opacity 0.2s'
                            }}
                          >
                            <EditIcon fontSize="small" />
                          </IconButton>
                        </>
                      )}
                    </Paper>
                  </TimelineContent>
                </TimelineItem>
              )})}
            </Timeline>
          )}
        </CardContent>
        </Card>
      </Box>

      {/* Speed Dial for Adding Notes and Activities */}
      <SpeedDial
        ariaLabel="Add note or activity"
        sx={{ position: 'fixed', bottom: 16, right: 16 }}
        icon={<SpeedDialIcon />}
      >
        <SpeedDialAction
          icon={<NoteIcon />}
          tooltipTitle={t('contactDetail.addNote')}
          onClick={() => setNoteDialogOpen(true)}
        />
        <SpeedDialAction
          icon={<EventIcon />}
          tooltipTitle={t('contactDetail.addActivity')}
          onClick={() => setActivityDialogOpen(true)}
        />
      </SpeedDial>

      {/* Dialogs */}
      <AddNoteDialog
        open={noteDialogOpen}
        onClose={() => setNoteDialogOpen(false)}
        onSave={handleSaveNote}
      />
      
      <AddActivityDialog
        open={activityDialogOpen}
        onClose={() => setActivityDialogOpen(false)}
        onSave={handleSaveActivity}
        token={token}
        preselectedContactId={contact?.ID}
      />
    </Box>
  );
}
