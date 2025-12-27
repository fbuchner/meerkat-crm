import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  getContact,
  updateContact,
  getContactProfilePicture,
  deleteContact,
  uploadProfilePicture
} from './api/contacts';
import { 
  getContactNotes, 
  Note 
} from './api/notes';
import {
  getContactActivities,
  Activity
} from './api/activities';
import {
  Box,
  Card,
  CardContent,
  IconButton,
  Divider,
  Button,
  Tabs,
  Tab,
  Typography
} from '@mui/material';
import { ContactDetailHeaderSkeleton, TimelineSkeleton } from './components/LoadingSkeletons';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import NoteIcon from '@mui/icons-material/Note';
import EventIcon from '@mui/icons-material/Event';
import NotificationsActiveIcon from '@mui/icons-material/NotificationsActive';
import AddNoteDialog from './components/AddNoteDialog';
import AddActivityDialog from './components/AddActivityDialog';
import ReminderDialog from './components/ReminderDialog';
import ReminderList from './components/ReminderList';
import EditTimelineItemDialog from './components/EditTimelineItemDialog';
import ContactHeader from './components/ContactHeader';
import ContactInformation from './components/ContactInformation';
import ContactTimeline from './components/ContactTimeline';
import ProfilePictureUploadDialog from './components/ProfilePictureUploadDialog';
import { useContactDialogs } from './hooks/useContactDialogs';
import { useTimelineEditing } from './hooks/useTimelineEditing';
import { useReminderManagement } from './hooks/useReminderManagement';
import { useRelationships } from './hooks/useRelationships';
import AddRelationshipDialog from './components/AddRelationshipDialog';
import { useSnackbar } from './context/SnackbarContext';
import { ApiError } from './api/client';
import { handleFetchError } from './utils/errorHandler';

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
  const { showError } = useSnackbar();
  const [contact, setContact] = useState<Contact | null>(null);
  const [profilePic, setProfilePic] = useState<string>('');
  const [loading, setLoading] = useState(true);
  const [editingField, setEditingField] = useState<string | null>(null);
  const [editValue, setEditValue] = useState<string>('');
  const [validationError, setValidationError] = useState<string>('');
  const [notes, setNotes] = useState<Note[]>([]);
  const [activities, setActivities] = useState<Activity[]>([]);
  
  // Profile editing state
  const [editingProfile, setEditingProfile] = useState(false);
  const [profileValues, setProfileValues] = useState({
    firstname: '',
    lastname: '',
    nickname: '',
    gender: ''
  });

  // Circle editing state
  const [editingCircles, setEditingCircles] = useState(false);
  const [newCircleName, setNewCircleName] = useState('');

  // Tab state
  const [activeTab, setActiveTab] = useState(0);

  // Profile picture upload state
  const [profilePictureDialogOpen, setProfilePictureDialogOpen] = useState(false);

  // Unified refresh function for notes and activities
  const refreshNotesAndActivities = async () => {
    if (!id) return;

    try {
      const notesData = await getContactNotes(id, token);
      setNotes(notesData.notes || []);

      const activitiesData = await getContactActivities(id, token);
      setActivities(activitiesData.activities || []);
    } catch (err) {
      handleFetchError(err, 'refreshing notes and activities');
    }
  };

  // Custom hooks
  const {
    noteDialogOpen,
    activityDialogOpen,
    setNoteDialogOpen,
    setActivityDialogOpen,
    handleSaveNote,
    handleSaveActivity
  } = useContactDialogs(id, token, refreshNotesAndActivities, { showError });

  const {
    editingTimelineItem,
    editTimelineValues,
    allContacts,
    handleStartEditTimelineItem,
    handleCancelEditTimelineItem,
    handleUpdateNote,
    handleUpdateActivity,
    handleDeleteNote,
    handleDeleteActivity,
    setEditTimelineValues
  } = useTimelineEditing(token, contact?.ID, refreshNotesAndActivities, { showError });

  const {
    reminders,
    reminderDialogOpen,
    editingReminder,
    refreshReminders,
    handleSaveReminder,
    handleCompleteReminder,
    handleEditReminder,
    handleDeleteReminder,
    handleAddReminder,
    setReminderDialogOpen,
    setEditingReminder
  } = useReminderManagement(id, token, { showError });

  const {
    relationships,
    relationshipDialogOpen,
    editingRelationship,
    refreshRelationships,
    handleSaveRelationship,
    handleEditRelationship,
    handleDeleteRelationship,
    handleAddRelationship,
    setRelationshipDialogOpen,
    setEditingRelationship,
  } = useRelationships(id, token, { showError });

  // Fetch contact details, notes, and activities
  useEffect(() => {
    if (!id) return;

    const fetchData = async () => {
      try {
        const contactData = await getContact(id, token);
        setContact(contactData);

        const notesData = await getContactNotes(id, token);
        setNotes(notesData.notes || []);

        const activitiesData = await getContactActivities(id, token);
        setActivities(activitiesData.activities || []);

        await refreshReminders();
        await refreshRelationships();

        setLoading(false);
      } catch (err) {
        console.error('Error fetching data:', err);
        setLoading(false);
      }
    };

    fetchData();

    getContactProfilePicture(id, token)
      .then(blob => {
        if (blob) {
          setProfilePic(URL.createObjectURL(blob));
        }
      })
      .catch(err => console.error('Error fetching profile picture:', err));
  }, [id, token, refreshReminders, refreshRelationships]);

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
    if (!value || value.trim() === '') return true;
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
      if (err instanceof ApiError) {
        const errorMessage = err.getDisplayMessage();
        setValidationError(errorMessage);
        showError(errorMessage);
      } else {
        showError(t('contactDetail.updateError'));
      }
    }
  };

  const handleAddCircle = async () => {
    if (!contact || !newCircleName.trim()) return;

    const trimmedCircleName = newCircleName.trim();
    const existingCircles = contact.circles || [];
    
    // Check if the circle already exists (case-insensitive)
    if (existingCircles.some(circle => circle.toLowerCase() === trimmedCircleName.toLowerCase())) {
      return; // Don't add duplicate circles
    }

    const updatedCircles = [...existingCircles, trimmedCircleName];

    try {
      const updatedContact = await updateContact(id!, {
        ...contact,
        circles: updatedCircles
      }, token);
      setContact(updatedContact);
      setNewCircleName('');
    } catch (err) {
      console.error('Error adding circle:', err);
      if (err instanceof ApiError) {
        showError(err.getDisplayMessage());
      } else {
        showError(t('contactDetail.updateError'));
      }
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
      if (err instanceof ApiError) {
        showError(err.getDisplayMessage());
      } else {
        showError(t('contactDetail.updateError'));
      }
    }
  };

  const handleStartEditProfile = () => {
    if (!contact) return;
    setProfileValues({
      firstname: contact.firstname || '',
      lastname: contact.lastname || '',
      nickname: contact.nickname || '',
      gender: contact.gender ? contact.gender.toLowerCase() : ''
    });
    setEditingProfile(true);
  };

  const handleCancelEditProfile = () => {
    setEditingProfile(false);
    setProfileValues({ firstname: '', lastname: '', nickname: '', gender: '' });
  };

  const handleSaveProfile = async () => {
    if (!contact || !profileValues.firstname.trim()) {
      alert('First name is required');
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
      if (err instanceof ApiError) {
        showError(err.getDisplayMessage());
      } else {
        showError(t('contactDetail.updateError'));
      }
    }
  };

  const handleDeleteContact = async () => {
    if (!contact || !id) return;

    const confirmMessage = t('contactDetail.confirmDeleteContact', { 
      name: `${contact.firstname} ${contact.lastname}` 
    });
    
    if (!window.confirm(confirmMessage)) {
      return;
    }

    try {
      await deleteContact(id, token);
      navigate('/contacts');
    } catch (err) {
      console.error('Error deleting contact:', err);
      alert(t('contactDetail.deleteContactError'));
    }
  };

  const handleUploadProfilePicture = async (croppedImageBlob: Blob) => {
    if (!id) return;

    await uploadProfilePicture(id, croppedImageBlob, token);
    
    // Refresh the profile picture
    const blob = await getContactProfilePicture(id, token);
    if (blob) {
      // Revoke old URL to prevent memory leaks
      if (profilePic) {
        URL.revokeObjectURL(profilePic);
      }
      setProfilePic(URL.createObjectURL(blob));
    }
  };

  if (loading) {
    return (
      <Box sx={{ maxWidth: 1200, mx: 'auto', mt: 1, px: 2, pb: 2 }}>
        <ContactDetailHeaderSkeleton />
        <Box sx={{ mt: 3 }}>
          <TimelineSkeleton count={5} />
        </Box>
      </Box>
    );
  }

  if (!contact) {
    return (
      <Box sx={{ maxWidth: 800, mx: 'auto', mt: 2, p: 2 }}>
        <Typography variant="h6">{t('contactDetail.notFound')}</Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ maxWidth: 1200, mx: 'auto', mt: 1, px: 2, pb: 2 }}>
      {/* Header with back button */}
      <IconButton onClick={() => navigate('/contacts')} size="small" sx={{ mb: 0.5, ml: -1 }}>
        <ArrowBackIcon fontSize="small" />
      </IconButton>

      {/* Contact Header Card */}
      <ContactHeader
        contact={contact}
        profilePic={profilePic}
        editingProfile={editingProfile}
        profileValues={profileValues}
        editingCircles={editingCircles}
        newCircleName={newCircleName}
        onStartEditProfile={handleStartEditProfile}
        onCancelEditProfile={handleCancelEditProfile}
        onSaveProfile={handleSaveProfile}
        onDeleteContact={handleDeleteContact}
        onProfileValueChange={setProfileValues}
        onToggleEditCircles={() => setEditingCircles(!editingCircles)}
        onAddCircle={handleAddCircle}
        onDeleteCircle={handleDeleteCircle}
        onNewCircleNameChange={setNewCircleName}
        onUploadProfilePicture={() => setProfilePictureDialogOpen(true)}
      />

      {/* General Information and Timeline - Two Column Layout */}
      <Box sx={{ 
        display: 'flex', 
        flexDirection: { xs: 'column', md: 'row' }, 
        gap: 2 
      }}>
        {/* General Information */}
        <ContactInformation
          contact={contact}
          editingField={editingField}
          editValue={editValue}
          validationError={validationError}
          onEditStart={handleEditStart}
          onEditCancel={handleEditCancel}
          onEditSave={handleEditSave}
          onEditValueChange={(value) => {
            setEditValue(value);
            setValidationError('');
          }}
          relationships={relationships}
          onAddRelationship={handleAddRelationship}
          onEditRelationship={handleEditRelationship}
          onDeleteRelationship={handleDeleteRelationship}
        />

        {/* Timeline and Reminders Tabs */}
        <Card sx={{ flex: 1 }}>
          <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
            <Tabs value={activeTab} onChange={(_, newValue) => setActiveTab(newValue)} aria-label="timeline and reminders tabs">
              <Tab label={t('contactDetail.timeline')} />
              <Tab label={t('reminders.title')} />
            </Tabs>
          </Box>

          {/* Tab Panel 0: Timeline - Notes and Activities */}
          {activeTab === 0 && (
            <CardContent sx={{ py: 2 }}>
              <Box sx={{ display: 'flex', justifyContent: 'flex-end', alignItems: 'center', mb: 1.5, gap: 0.5 }}>
                <Button 
                  startIcon={<NoteIcon />} 
                  onClick={() => setNoteDialogOpen(true)}
                  variant="outlined"
                  size="small"
                >
                  {t('contactDetail.addNote')}
                </Button>
                <Button 
                  startIcon={<EventIcon />} 
                  onClick={() => setActivityDialogOpen(true)}
                  variant="outlined"
                  size="small"
                >
                  {t('contactDetail.addActivity')}
                </Button>
              </Box>
              <Divider sx={{ mb: 2 }} />
              
              <ContactTimeline
                timelineItems={timelineItems}
                onEditItem={handleStartEditTimelineItem}
              />
            </CardContent>
          )}

          {/* Tab Panel 1: Reminders */}
          {activeTab === 1 && (
            <CardContent sx={{ py: 2 }}>
              <Box sx={{ display: 'flex', justifyContent: 'flex-end', alignItems: 'center', mb: 1.5 }}>
                <Button 
                  startIcon={<NotificationsActiveIcon />} 
                  onClick={handleAddReminder}
                  variant="outlined"
                  size="small"
                >
                  {t('reminders.add')}
                </Button>
              </Box>
              <Divider sx={{ mb: 1.5 }} />
              <ReminderList
                reminders={reminders}
                onComplete={handleCompleteReminder}
                onEdit={handleEditReminder}
                onDelete={handleDeleteReminder}
              />
            </CardContent>
          )}
        </Card>
      </Box>

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

      <ReminderDialog
        open={reminderDialogOpen}
        onClose={() => {
          setReminderDialogOpen(false);
          setEditingReminder(null);
        }}
        onSave={handleSaveReminder}
        reminder={editingReminder}
        contactId={contact?.ID || 0}
      />

      {editingTimelineItem && (
        <EditTimelineItemDialog
          open={!!editingTimelineItem}
          onClose={handleCancelEditTimelineItem}
          onSave={() => {
            if (editingTimelineItem.type === 'note') {
              handleUpdateNote(editingTimelineItem.id);
            } else {
              handleUpdateActivity(editingTimelineItem.id);
            }
          }}
          onDelete={() => {
            if (editingTimelineItem.type === 'note') {
              handleDeleteNote(editingTimelineItem.id);
            } else {
              handleDeleteActivity(editingTimelineItem.id);
            }
          }}
          type={editingTimelineItem.type}
          values={editTimelineValues}
          onChange={setEditTimelineValues}
          allContacts={allContacts}
        />
      )}

      <ProfilePictureUploadDialog
        open={profilePictureDialogOpen}
        onClose={() => setProfilePictureDialogOpen(false)}
        onUpload={handleUploadProfilePicture}
      />

      <AddRelationshipDialog
        open={relationshipDialogOpen}
        onClose={() => {
          setRelationshipDialogOpen(false);
          setEditingRelationship(null);
        }}
        onSave={handleSaveRelationship}
        relationship={editingRelationship}
        token={token}
        currentContactId={contact?.ID || 0}
      />
    </Box>
  );
}
