import { useState, useEffect, useCallback } from 'react';
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
  Tooltip,
  Popover
} from '@mui/material';
import CakeIcon from '@mui/icons-material/Cake';
import ShuffleIcon from '@mui/icons-material/Shuffle';
import NotificationsIcon from '@mui/icons-material/Notifications';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import EmailIcon from '@mui/icons-material/Email';
import RepeatIcon from '@mui/icons-material/Repeat';
import WarningIcon from '@mui/icons-material/Warning';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import { Contact, Birthday, getRandomContacts, getUpcomingBirthdays, getContact } from './api/contacts';
import { Reminder, getUpcomingReminders, completeReminder } from './api/reminders';
import { ContactListSkeleton } from './components/LoadingSkeletons';
import { handleFetchError, handleError } from './utils/errorHandler';
import { useDateFormat } from './DateFormatProvider';

interface DashboardPageProps {
  token: string;
}

function DashboardPage({ token }: DashboardPageProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { formatBirthday: formatBirthdayDate, formatDate } = useDateFormat();
  const [birthdays, setBirthdays] = useState<Birthday[]>([]);
  const [randomContacts, setRandomContacts] = useState<Contact[]>([]);
  const [upcomingReminders, setUpcomingReminders] = useState<Reminder[]>([]);
  const [contactsMap, setContactsMap] = useState<Record<number, Contact>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [birthdaysInfoAnchor, setBirthdaysInfoAnchor] = useState<HTMLElement | null>(null);
  const [remindersInfoAnchor, setRemindersInfoAnchor] = useState<HTMLElement | null>(null);
  const [stayInTouchInfoAnchor, setStayInTouchInfoAnchor] = useState<HTMLElement | null>(null);

  const loadDashboardData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      const [birthdayData, random, reminders] = await Promise.all([
        getUpcomingBirthdays(token),
        getRandomContacts(token),
        getUpcomingReminders(token)
      ]);

      setBirthdays(birthdayData);
      setRandomContacts(random);
      setUpcomingReminders(reminders);

      // Build contact map from random contacts
      const newContactsMap: Record<number, Contact> = {};
      random.forEach(c => { newContactsMap[c.ID] = c; });

      // Fetch missing contacts for reminders
      const missingContactIds = reminders
        .map(r => r.contact_id)
        .filter(id => !newContactsMap[id]);
      const uniqueMissingIds = Array.from(new Set(missingContactIds));

      if (uniqueMissingIds.length > 0) {
        const fetchedContacts = await Promise.all(
          uniqueMissingIds.map(id => getContact(id, token).catch(() => null))
        );
        fetchedContacts.forEach(c => {
          if (c) newContactsMap[c.ID] = c;
        });
      }

      setContactsMap(newContactsMap);
    } catch (err) {
      const message = handleFetchError(err, 'loading dashboard data');
      setError(message);
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    loadDashboardData();
  }, [loadDashboardData]);

  const handleCompleteReminder = async (reminderId: number) => {
    try {
      await completeReminder(reminderId, token);
      // Reload reminders after completion
      const reminders = await getUpcomingReminders(token);
      setUpcomingReminders(reminders);
    } catch (err) {
      handleError(err, { operation: 'completing reminder' });
    }
  };

  const isOverdue = (remindAt: string) => {
    return new Date(remindAt) < new Date();
  };

  const formatBirthday = (birthday: string | undefined) => {
    if (!birthday) return '';
    
    // Use the date format provider's birthday formatter
    const formattedDate = formatBirthdayDate(birthday);
    
    // Check if year is present to calculate age
    if (!birthday.startsWith('--')) {
      const parts = birthday.split('-');
      if (parts.length === 3 && parts[0].length === 4) {
        const birthYear = parseInt(parts[0], 10);
        if (!isNaN(birthYear)) {
          const currentYear = new Date().getFullYear();
          const age = currentYear - birthYear;
          // Remove the year from the formatted date for dashboard display (just show DD.MM. or MM/DD)
          const dateWithoutYear = formatBirthdayDate(`--${parts[1]}-${parts[2]}`);
          return `${dateWithoutYear} ${t('dashboard.yearsOld', { age })}`;
        }
      }
    }
    
    return formattedDate;
  };

  const getContactName = (contact: Contact) => {
    if (contact.nickname) return `${contact.nickname} ${contact.lastname}`;
    return `${contact.firstname} ${contact.lastname}`;
  };


  if (loading) {
    return (
      <Box sx={{ maxWidth: 1400, mx: 'auto', mt: 2, p: 2 }}>
        <Typography variant="h5" gutterBottom>
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
          <Box>
            <ContactListSkeleton count={5} />
          </Box>
        </Box>
      </Box>
    );
  }

  if (error) {
    return (
      <Box sx={{ maxWidth: 1400, mx: 'auto', mt: 2, p: 2 }}>
        <Alert severity="error">{error}</Alert>
      </Box>
    );
  }

  return (
    <Box sx={{ maxWidth: 1400, mx: 'auto', mt: 2, p: 2 }}>
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
            <IconButton
              size="small"
              onClick={(e) => setBirthdaysInfoAnchor(e.currentTarget)}
            >
              <InfoOutlinedIcon fontSize="small" />
            </IconButton>
          </Box>
          <Popover
            open={Boolean(birthdaysInfoAnchor)}
            anchorEl={birthdaysInfoAnchor}
            onClose={() => setBirthdaysInfoAnchor(null)}
            anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}
          >
            <Box sx={{ p: 2, maxWidth: 320 }}>
              <Typography variant="body2">{t('dashboard.birthdaysInfo')}</Typography>
            </Box>
          </Popover>

          {birthdays.length === 0 ? (
            <Card>
              <CardContent sx={{ py: 2 }}>
                <Typography variant="body2" color="text.secondary">
                  {t('dashboard.noBirthdays')}
                </Typography>
              </CardContent>
            </Card>
          ) : (
            <Stack spacing={1.5}>
              {birthdays.map((birthday, index) => (
                <Card
                  key={`${birthday.type}-${birthday.contact_id}-${index}`}
                  component={Link}
                  to={`/contacts/${birthday.contact_id}`}
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
                      <Avatar
                        src={birthday.type === 'contact' ? birthday.photo_thumbnail : undefined}
                        sx={{ bgcolor: 'primary.main', width: 40, height: 40 }}
                      >
                        {birthday.name.charAt(0)}
                      </Avatar>
                      <Box sx={{ flexGrow: 1 }}>
                        <Typography variant="body2" fontWeight={500}>
                          {birthday.name}
                        </Typography>
                        <Typography variant="caption" color="text.secondary">
                          {formatBirthday(birthday.birthday)}
                        </Typography>
                      </Box>
                      {birthday.type === 'relationship' && birthday.relationship_type && (
                        <Box sx={{ textAlign: 'right' }}>
                          <Chip
                            label={birthday.relationship_type}
                            size="small"
                            variant="outlined"
                            sx={{ height: 20, fontSize: '0.7rem' }}
                          />
                          {birthday.associated_contact_name && (
                            <Typography variant="caption" color="text.secondary" display="block">
                              {t('relationships.ofContact')} {birthday.associated_contact_name}
                            </Typography>
                          )}
                        </Box>
                      )}
                    </Box>
                  </CardContent>
                </Card>
              ))}
            </Stack>
          )}
        </Box>

        {/* Column 2: Upcoming Reminders */}
        <Box>
          <Box sx={{ mb: 1.5, display: 'flex', alignItems: 'center', gap: 1 }}>
            <NotificationsIcon color="primary" fontSize="small" />
            <Typography variant="subtitle1" fontWeight={500}>
              {t('dashboard.upcomingReminders')}
            </Typography>
            <IconButton
              size="small"
              onClick={(e) => setRemindersInfoAnchor(e.currentTarget)}
            >
              <InfoOutlinedIcon fontSize="small" />
            </IconButton>
          </Box>
          <Popover
            open={Boolean(remindersInfoAnchor)}
            anchorEl={remindersInfoAnchor}
            onClose={() => setRemindersInfoAnchor(null)}
            anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}
          >
            <Box sx={{ p: 2, maxWidth: 320 }}>
              <Typography variant="body2">{t('dashboard.remindersInfo')}</Typography>
            </Box>
          </Popover>

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
                const contact = contactsMap[reminder.contact_id];

                return (
                  <Card
                    key={reminder.ID}
                    sx={{
                      border: '1px solid',
                      borderColor: overdue ? 'warning.main' : 'divider',
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
                          {contact && (
                            <Typography variant="caption" color="text.secondary">
                              {getContactName(contact)}
                            </Typography>
                          )}
                          <Box sx={{ mt: 0.75, display: 'flex', gap: 0.5, flexWrap: 'wrap', alignItems: 'center' }}>
                            <Chip
                              icon={overdue ? <WarningIcon fontSize="small" /> : undefined}
                              label={formatDate(reminder.remind_at)}
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
                            sx={{
                              transition: 'transform 0.15s ease-in-out, box-shadow 0.15s ease-in-out',
                              '&:hover': {
                                transform: 'scale(1.15)',
                                boxShadow: '0 0 8px rgba(76, 175, 80, 0.5)',
                              },
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

        {/* Column 3: Random Contacts (Stay in Touch) */}
        <Box>
          <Box sx={{ mb: 1.5, display: 'flex', alignItems: 'center', gap: 1 }}>
            <ShuffleIcon color="primary" fontSize="small" />
            <Typography variant="subtitle1" fontWeight={500}>
              {t('dashboard.randomContacts')}
            </Typography>
            <IconButton
              size="small"
              onClick={(e) => setStayInTouchInfoAnchor(e.currentTarget)}
            >
              <InfoOutlinedIcon fontSize="small" />
            </IconButton>
          </Box>
          <Popover
            open={Boolean(stayInTouchInfoAnchor)}
            anchorEl={stayInTouchInfoAnchor}
            onClose={() => setStayInTouchInfoAnchor(null)}
            anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}
          >
            <Box sx={{ p: 2, maxWidth: 320 }}>
              <Typography variant="body2">{t('dashboard.stayInTouchInfo')}</Typography>
            </Box>
          </Popover>

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
                      <Avatar
                        src={contact.photo_thumbnail || undefined}
                        sx={{ bgcolor: 'secondary.main', width: 40, height: 40 }}
                      >
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
      </Box>
    </Box>
  );
}

export default DashboardPage;
