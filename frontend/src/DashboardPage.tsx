import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import {
  Box,
  Typography,
  Card,
  CardContent,
  Avatar,
  Stack,
  Chip,
  Alert
} from '@mui/material';
import CakeIcon from '@mui/icons-material/Cake';
import ShuffleIcon from '@mui/icons-material/Shuffle';
import { Contact, getRandomContacts, getUpcomingBirthdays } from './api/contacts';
import { ContactListSkeleton } from './components/LoadingSkeletons';

interface DashboardPageProps {
  token: string;
}

function DashboardPage({ token }: DashboardPageProps) {
  const { t } = useTranslation();
  const [birthdayContacts, setBirthdayContacts] = useState<Contact[]>([]);
  const [randomContacts, setRandomContacts] = useState<Contact[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadDashboardData();
  }, [token]);

  const loadDashboardData = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const [birthdays, random] = await Promise.all([
        getUpcomingBirthdays(token),
        getRandomContacts(token)
      ]);
      
      setBirthdayContacts(birthdays);
      setRandomContacts(random);
    } catch (err) {
      console.error('Error loading dashboard data:', err);
      setError(t('dashboard.error') || 'Failed to load dashboard data');
    } finally {
      setLoading(false);
    }
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
    <Box sx={{ maxWidth: 1400, mx: 'auto', p: 3 }}>
      <Typography variant="h4" gutterBottom>
        {t('dashboard.title')}
      </Typography>

      <Box sx={{ 
        display: 'grid', 
        gridTemplateColumns: { xs: '1fr', md: 'repeat(3, 1fr)' },
        gap: 3 
      }}>
        {/* Column 1: Upcoming Birthdays */}
        <Box>
          <Box sx={{ mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
            <CakeIcon color="primary" />
            <Typography variant="h6">
              {t('dashboard.upcomingBirthdays')}
            </Typography>
          </Box>

          {birthdayContacts.length === 0 ? (
            <Card>
              <CardContent>
                <Typography variant="body2" color="text.secondary">
                  {t('dashboard.noBirthdays')}
                </Typography>
              </CardContent>
            </Card>
          ) : (
            <Stack spacing={2}>
              {birthdayContacts.map((contact) => (
                <Card
                  key={contact.ID}
                  component={Link}
                  to={`/contacts/${contact.ID}`}
                  sx={{
                    textDecoration: 'none',
                    '&:hover': {
                      boxShadow: 3,
                      transform: 'translateY(-2px)',
                      transition: 'all 0.2s'
                    }
                  }}
                >
                  <CardContent>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                      <Avatar sx={{ bgcolor: 'primary.main' }}>
                        {contact.firstname.charAt(0)}
                      </Avatar>
                      <Box sx={{ flexGrow: 1 }}>
                        <Typography variant="subtitle1">
                          {getContactName(contact)}
                        </Typography>
                        <Typography variant="body2" color="text.secondary">
                          {formatBirthday(contact.birthday)}
                        </Typography>
                      </Box>
                      {contact.circles && contact.circles.length > 0 && (
                        <Box>
                          <Chip
                            label={contact.circles[0]}
                            size="small"
                            variant="outlined"
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
          <Box sx={{ mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
            <ShuffleIcon color="primary" />
            <Typography variant="h6">
              {t('dashboard.randomContacts')}
            </Typography>
          </Box>

          {randomContacts.length === 0 ? (
            <Card>
              <CardContent>
                <Typography variant="body2" color="text.secondary">
                  {t('dashboard.noContacts')}
                </Typography>
              </CardContent>
            </Card>
          ) : (
            <Stack spacing={2}>
              {randomContacts.map((contact) => (
                <Card
                  key={contact.ID}
                  component={Link}
                  to={`/contacts/${contact.ID}`}
                  sx={{
                    textDecoration: 'none',
                    '&:hover': {
                      boxShadow: 3,
                      transform: 'translateY(-2px)',
                      transition: 'all 0.2s'
                    }
                  }}
                >
                  <CardContent>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                      <Avatar sx={{ bgcolor: 'secondary.main' }}>
                        {contact.firstname.charAt(0)}
                      </Avatar>
                      <Box sx={{ flexGrow: 1 }}>
                        <Typography variant="subtitle1">
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
                                sx={{ mr: 0.5 }}
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

        {/* Column 3: Empty for now */}
        <Box>
          <Box sx={{ mb: 2 }}>
            <Typography variant="h6" color="text.secondary">
              {t('dashboard.comingSoon')}
            </Typography>
          </Box>
          <Card>
            <CardContent>
              <Typography variant="body2" color="text.secondary">
                {t('dashboard.moreFeaturesComingSoon')}
              </Typography>
            </CardContent>
          </Card>
        </Box>
      </Box>
    </Box>
  );
}

export default DashboardPage;
