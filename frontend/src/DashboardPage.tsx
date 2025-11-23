import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, useNavigate } from 'react-router-dom';
import {
  Box,
  Typography,
  Card,
  CardContent,
  Avatar,
  Stack,
  Chip,
  Alert,
  IconButton,
  Tooltip
} from '@mui/material';
import CakeIcon from '@mui/icons-material/Cake';
import ShuffleIcon from '@mui/icons-material/Shuffle';
import NotificationsIcon from '@mui/icons-material/Notifications';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import EmailIcon from '@mui/icons-material/Email';
import RepeatIcon from '@mui/icons-material/Repeat';
import WarningIcon from '@mui/icons-material/Warning';
import { Contact, getRandomContacts, getUpcomingBirthdays } from './api/contacts';
import { Reminder, getUpcomingReminders, completeReminder } from './api/reminders';
import { ContactListSkeleton } from './components/LoadingSkeletons';

interface DashboardPageProps {
  token: string;
}

function DashboardPage({ token }: DashboardPageProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [birthdayContacts, setBirthdayContacts] = useState<Contact[]>([]);
  const [randomContacts, setRandomContacts] = useState<Contact[]>([]);
  const [upcomingReminders, setUpcomingReminders] = useState<Reminder[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadDashboardData();
  }, [token]);

  const loadDashboardData = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const [birthdays, random, reminders] = await Promise.all([
        getUpcomingBirthdays(token),
        getRandomContacts(token),
        getUpcomingReminders(token)
      ]);
      
      setBirthdayContacts(birthdays);
      setRandomContacts(random);
      setUpcomingReminders(reminders);
    } catch (err) {
      console.error('Error loading dashboard data:', err);
      setError(t('dashboard.error') || 'Failed to load dashboard data');
    } finally {
      setLoading(false);
    }
  };

  const handleCompleteReminder = async (reminderId: number) => {
    try {
      await completeReminder(reminderId, token);
      // Reload reminders after completion
      const reminders = await getUpcomingReminders(token);
      setUpcomingReminders(reminders);
    } catch (err) {
      console.error('Error completing reminder:', err);
    }
  };

  const isOverdue = (remindAt: string) => {
    return new Date(remindAt) < new Date();
  };

  const formatBirthday = (birthday: string | undefined) => {
    if (!birthday) return '';
    // Birthday format is DD.MM.YYYY or DD.MM.
    const parts = birthday.split('.');
    if (parts.length >= 2) {
      return `${parts[0]}.${parts[1]}.`;
    }
    return birthday;
  };

  const getContactName = (contact: Contact) => {
    if (contact.nickname) return contact.nickname;
    return `${contact.firstname} ${contact.lastname}`;
  };

  if (loading) {
    return (
      <Box sx={{ maxWidth: 1400, mx: 'auto', p: 3 }}>
        <Typography variant="h4" gutterBottom>
          {t('dashboard.title')}
        </Typography>
        <Box sx={{ 
          display: 'grid', 
          gridTemplateColumns: { xs: '1fr', md: 'repeat(3, 1fr)' },
          gap: 3 
        }}>
          <Box>
            <ContactListSkeleton count={5} />
          </Box>
          <Box>
            <ContactListSkeleton count={5} />
          </Box>
          <Box sx={{ p: 2 }}>
            <Typography variant="body2" color="text.secondary">
              {t('dashboard.comingSoon')}
            </Typography>
          </Box>
        </Box>
      </Box>
    );
  }

  if (error) {
    return (
      <Box sx={{ maxWidth: 1400, mx: 'auto', p: 3 }}>
        <Alert severity="error">{error}</Alert>
      </Box>
    );
  }

  return (
    <Box sx={{ maxWidth: 1600, mx: 'auto', p: 2 }}>
      <Typography variant="h5" gutterBottom sx={{ mb: 2 }}>
        {t('dashboard.title')}
      </Typography>

      <Box sx={{ 
        display: 'grid', 
        gridTemplateColumns: { xs: '1fr', md: 'repeat(3, 1fr)' },
        gap: 2 
      }}>
        {/* Column 1: Upcoming Birthdays */}
        <Box>
          <Box sx={{ mb: 1.5, display: 'flex', alignItems: 'center', gap: 1 }}>
            <CakeIcon color="primary" fontSize="small" />
            <Typography variant="subtitle1" fontWeight={500}>
              {t('dashboard.upcomingBirthdays')}
            </Typography>
          </Box>

          {birthdayContacts.length === 0 ? (
            <Card>
              <CardContent sx={{ py: 2 }}>
                <Typography variant="body2" color="text.secondary">
                  {t('dashboard.noBirthdays')}
                </Typography>
              </CardContent>
            </Card>
          ) : (
            <Stack spacing={1.5}>
              {birthdayContacts.map((contact) => (
                <Card
                  key={contact.ID}
                  component={Link}
                  to={`/contacts/${contact.ID}`}
                  sx={{
                    textDecoration: 'none',
                    '&:hover': {
                      boxShadow: 2,
                      transform: 'translateY(-1px)',
                      transition: 'all 0.2s'
                    }
                  }}
                >
                  <CardContent sx={{ py: 1.5 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                      <Avatar sx={{ bgcolor: 'primary.main', width: 40, height: 40 }}>
                        {contact.firstname.charAt(0)}
                      </Avatar>
                      <Box sx={{ flexGrow: 1 }}>
                        <Typography variant="body2" fontWeight={500}>
                          {getContactName(contact)}
                        </Typography>
                        <Typography variant="caption" color="text.secondary">
                          {formatBirthday(contact.birthday)}
                        </Typography>
                      </Box>
                      {contact.circles && contact.circles.length > 0 && (
                        <Box>
                          <Chip
                            label={contact.circles[0]}
                            size="small"
                            variant="outlined"
                            sx={{ height: 20, fontSize: '0.7rem' }}
                          />
                        </Box>
                      )}
                    </Box>
                  </CardContent>
                </Card>
              ))}
            </Stack>
          )}
        </Box>

        {/* Column 2: Random Contacts */}
        <Box>
          <Box sx={{ mb: 1.5, display: 'flex', alignItems: 'center', gap: 1 }}>
            <ShuffleIcon color="primary" fontSize="small" />
            <Typography variant="subtitle1" fontWeight={500}>
              {t('dashboard.randomContacts')}
            </Typography>
          </Box>

          {randomContacts.length === 0 ? (
            <Card>
              <CardContent sx={{ py: 2 }}>
                <Typography variant="body2" color="text.secondary">
                  {t('dashboard.noContacts')}
                </Typography>
              </CardContent>
            </Card>
          ) : (
            <Stack spacing={1.5}>
              {randomContacts.map((contact) => (
                <Card
                  key={contact.ID}
                  component={Link}
                  to={`/contacts/${contact.ID}`}
                  sx={{
                    textDecoration: 'none',
                    '&:hover': {
                      boxShadow: 2,
                      transform: 'translateY(-1px)',
                      transition: 'all 0.2s'
                    }
                  }}
                >
                  <CardContent sx={{ py: 1.5 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                      <Avatar sx={{ bgcolor: 'secondary.main', width: 40, height: 40 }}>
                        {contact.firstname.charAt(0)}
                      </Avatar>
                      <Box sx={{ flexGrow: 1 }}>
                        <Typography variant="body2" fontWeight={500}>
                          {getContactName(contact)}
                        </Typography>
                        {contact.circles && contact.circles.length > 0 && (
                          <Box sx={{ mt: 0.5 }}>
                            {contact.circles.slice(0, 2).map((circle, idx) => (
                              <Chip
                                key={idx}
                                label={circle}
                                size="small"
                                variant="outlined"
                                sx={{ mr: 0.5, height: 20, fontSize: '0.7rem' }}
                              />
                            ))}
                          </Box>
                        )}
                      </Box>
                    </Box>
                  </CardContent>
                </Card>
              ))}
            </Stack>
          )}
        </Box>

        {/* Column 3: Upcoming Reminders */}
        <Box>
          <Box sx={{ mb: 1.5, display: 'flex', alignItems: 'center', gap: 1 }}>
            <NotificationsIcon color="primary" fontSize="small" />
            <Typography variant="subtitle1" fontWeight={500}>
              {t('dashboard.upcomingReminders')}
            </Typography>
          </Box>

          {upcomingReminders.length === 0 ? (
            <Card>
              <CardContent sx={{ py: 2 }}>
                <Typography variant="body2" color="text.secondary">
                  {t('dashboard.noReminders')}
                </Typography>
              </CardContent>
            </Card>
          ) : (
            <Stack spacing={1.5}>
              {upcomingReminders.map((reminder) => {
                const overdue = isOverdue(reminder.remind_at);
                const reminderDate = new Date(reminder.remind_at);
                
                return (
                  <Card
                    key={reminder.ID}
                    sx={{
                      border: overdue ? '1px solid' : 'none',
                      borderColor: overdue ? 'warning.main' : 'transparent',
                      cursor: 'pointer',
                      '&:hover': {
                        boxShadow: 2,
                        transform: 'translateY(-1px)',
                        transition: 'all 0.2s'
                      }
                    }}
                    onClick={() => navigate(`/contacts/${reminder.contact_id}`)}
                  >
                    <CardContent sx={{ py: 1.5 }}>
                      <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 1 }}>
                        <Box sx={{ flexGrow: 1 }}>
                          <Typography variant="body2" sx={{ fontWeight: 500 }}>
                            {reminder.message}
                          </Typography>
                          <Box sx={{ mt: 0.75, display: 'flex', gap: 0.5, flexWrap: 'wrap', alignItems: 'center' }}>
                            <Chip
                              icon={overdue ? <WarningIcon fontSize="small" /> : undefined}
                              label={reminderDate.toLocaleDateString()}
                              size="small"
                              color={overdue ? 'warning' : 'default'}
                              sx={{ height: 20, fontSize: '0.7rem' }}
                            />
                            {reminder.recurrence !== 'once' && (
                              <Chip
                                icon={<RepeatIcon fontSize="small" />}
                                label={t(`reminders.recurrence.${reminder.recurrence}`)}
                                size="small"
                                variant="outlined"
                                sx={{ height: 20, fontSize: '0.7rem' }}
                              />
                            )}
                            {reminder.by_mail && (
                              <Chip
                                icon={<EmailIcon fontSize="small" />}
                                label={t('reminders.email')}
                                size="small"
                                variant="outlined"
                                sx={{ height: 20, fontSize: '0.7rem' }}
                              />
                            )}
                          </Box>
                        </Box>
                        <Tooltip title={t('reminders.complete')}>
                          <IconButton
                            size="small"
                            color="success"
                            onClick={(e) => {
                              e.stopPropagation();
                              handleCompleteReminder(reminder.ID);
                            }}
                          >
                            <CheckCircleIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      </Box>
                    </CardContent>
                  </Card>
                );
              })}
            </Stack>
          )}
        </Box>
      </Box>
    </Box>
  );
}

export default DashboardPage;
