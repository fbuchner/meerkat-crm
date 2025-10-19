import React, { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { API_BASE_URL } from './api';
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
  TextField
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

interface Note {
  ID: number;
  content: string;
  date: string;
  CreatedAt: string;
  UpdatedAt: string;
}

interface Activity {
  ID: number;
  title: string;
  description?: string;
  date: string;
  CreatedAt: string;
  UpdatedAt: string;
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

  useEffect(() => {
    if (!id) return;

    // Fetch contact details
    fetch(`${API_BASE_URL}/contacts/${id}`, {
      headers: { 'Authorization': `Bearer ${token}` }
    })
      .then(res => res.json())
      .then(data => {
        setContact(data);
        setLoading(false);
      })
      .catch(err => {
        console.error('Error fetching contact:', err);
        setLoading(false);
      });

    // Fetch profile picture
    fetch(`${API_BASE_URL}/contacts/${id}/profile_picture`, {
      headers: { 'Authorization': `Bearer ${token}` }
    })
      .then(res => {
        if (res.ok) return res.blob();
        return null;
      })
      .then(blob => {
        if (blob) {
          setProfilePic(URL.createObjectURL(blob));
        }
      })
      .catch(err => console.error('Error fetching profile picture:', err));
  }, [id, token]);

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
    ...(contact.notes || []).map(note => ({
      type: 'note' as const,
      data: note,
      date: note.date || note.CreatedAt
    })),
    ...(contact.activities || []).map(activity => ({
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
      const response = await fetch(`${API_BASE_URL}/contacts/${id}`, {
        method: 'PUT',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          ...contact,
          [field]: editValue
        })
      });

      if (response.ok) {
        const updatedContact = await response.json();
        setContact(updatedContact);
        setEditingField(null);
        setEditValue('');
        setValidationError('');
      }
    } catch (err) {
      console.error('Error updating contact:', err);
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
          <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
            <Avatar
              src={profilePic || undefined}
              sx={{ width: 100, height: 100, mr: 3 }}
            />
            <Box sx={{ flex: 1 }}>
              <Typography variant="h4" sx={{ fontWeight: 500 }}>
                {contact.firstname} {contact.nickname && `"${contact.nickname}"`} {contact.lastname}
              </Typography>
              {contact.gender && (
                <Typography variant="body1" color="text.secondary" sx={{ mt: 1 }}>
                  {contact.gender}
                </Typography>
              )}
              {contact.circles && contact.circles.length > 0 && (
                <Stack direction="row" spacing={1} mt={2}>
                  {contact.circles.map((circle, index) => (
                    <Chip key={index} label={circle} size="small" color="primary" />
                  ))}
                </Stack>
              )}
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
                
                return (
                <TimelineItem key={`${item.type}-${item.data.ID}`}>
                  <TimelineOppositeContent color="text.secondary" sx={{ flex: 0.3 }}>
                    <Typography variant="caption">
                      {isValidDate ? itemDate.toLocaleDateString() : (item.date || 'N/A')}
                    </Typography>
                  </TimelineOppositeContent>
                  <TimelineSeparator>
                    <TimelineDot color={item.type === 'note' ? 'primary' : 'secondary'}>
                      {item.type === 'note' ? <NoteIcon fontSize="small" /> : <EventIcon fontSize="small" />}
                    </TimelineDot>
                    {index < timelineItems.length - 1 && <TimelineConnector />}
                  </TimelineSeparator>
                  <TimelineContent>
                    <Paper elevation={2} sx={{ p: 2 }}>
                      <Typography variant="subtitle2" sx={{ fontWeight: 500 }}>
                        {item.type === 'note' ? t('contactDetail.note') : (item.data as Activity).title}
                      </Typography>
                      <Typography variant="body2" sx={{ mt: 1 }}>
                        {item.type === 'note' 
                          ? (item.data as Note).content 
                          : (item.data as Activity).description || t('contactDetail.noDescription')}
                      </Typography>
                    </Paper>
                  </TimelineContent>
                </TimelineItem>
              )})}
            </Timeline>
          )}
        </CardContent>
        </Card>
      </Box>
    </Box>
  );
}
