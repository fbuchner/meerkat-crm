import { useEffect, useState, useMemo } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useContacts } from './hooks/useContacts';
import { getCircles, getContactProfilePicture } from './api/contacts';
import AddContactDialog from './components/AddContactDialog';
import {
  Box,
  Card,
  Avatar,
  Typography,
  Chip,
  Stack,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Pagination,
  Button
} from '@mui/material';
import PersonAddIcon from '@mui/icons-material/PersonAdd';
import { ContactListSkeleton } from './components/LoadingSkeletons';

export default function ContactsPage({ token }: { token: string }) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [profilePics, setProfilePics] = useState<{ [key: string]: string }>({});
  const searchQuery = searchParams.get('search') || '';
  const [selectedCircle, setSelectedCircle] = useState('');
  const [circles, setCircles] = useState<string[]>([]);
  const [page, setPage] = useState(1);
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const pageSize = 10;

  // Memoize params to prevent infinite re-renders
  const contactParams = useMemo(() => ({
    page,
    limit: pageSize,
    search: searchQuery,
    circle: selectedCircle
  }), [page, searchQuery, selectedCircle]);

  // Use custom hook for fetching contacts
  const { contacts, total: totalContacts, loading, refetch } = useContacts(contactParams);

  // Fetch circles for filter
  useEffect(() => {
    const fetchCircles = async () => {
      try {
        const data = await getCircles(token);
        setCircles(Array.isArray(data) ? data : []);
      } catch (err) {
        console.error('Error fetching circles:', err);
      }
    };
    fetchCircles();
  }, [token]);

  // Reset to page 1 when search changes
  useEffect(() => {
    setPage(1);
  }, [searchQuery]);

  // Fetch profile pictures (thumbnails) when contacts change
  useEffect(() => {
    const fetchProfilePics = async () => {
      const picPromises = contacts.map(async (contact) => {
        try {
          const blob = await getContactProfilePicture(contact.ID, token, true);
          if (blob) {
            return { id: contact.ID, url: URL.createObjectURL(blob) };
          }
        } catch {
          // Silently fail for profile picture loading
        }
        return { id: contact.ID, url: '' };
      });
      const picResults = await Promise.all(picPromises);
      const picMap: { [key: string]: string } = {};
      picResults.forEach(({ id, url }) => { picMap[id] = url; });
      setProfilePics((prevPics) => {
        // Revoke old blob URLs to prevent memory leaks
        Object.values(prevPics).forEach((url) => {
          if (url) URL.revokeObjectURL(url);
        });
        return picMap;
      });
    };
    
    if (contacts.length > 0) {
      fetchProfilePics();
    }

    // Cleanup blob URLs on unmount
    return () => {
      setProfilePics((prevPics) => {
        Object.values(prevPics).forEach((url) => {
          if (url) URL.revokeObjectURL(url);
        });
        return {};
      });
    };
  }, [contacts, token]);

  // Filter contacts by selected circle
  // With backend pagination, contacts are already filtered
  const filteredContacts = contacts;

  const isFiltered = selectedCircle !== '' || searchQuery !== '';

  const handleContactAdded = async () => {
    await refetch();
    // Also refresh circles
    try {
      const data = await getCircles(token);
      setCircles(Array.isArray(data) ? data : []);
    } catch (err) {
      console.error('Error fetching circles:', err);
    }
  };
  
  return (
    <Box sx={{ maxWidth: 1200, mx: 'auto', mt: 2, p: 2 }}>
      <Typography variant="h5" gutterBottom sx={{ mb: 2 }}>
        {t('contacts.title')}
      </Typography>
      <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5} mb={2} alignItems="center">
        <FormControl sx={{ minWidth: 180 }} size="small">
          <InputLabel id="circle-select-label">{t('contacts.filterByCircle')}</InputLabel>
          <Select
            labelId="circle-select-label"
            value={selectedCircle}
            label={t('contacts.filterByCircle')}
            onChange={e => setSelectedCircle(e.target.value)}
          >
            <MenuItem value="">{t('contacts.allCircles')}</MenuItem>
            {circles.map(circle => (
              <MenuItem key={circle} value={circle}>{circle}</MenuItem>
            ))}
          </Select>
        </FormControl>
        <Button
          variant="outlined"
          startIcon={<PersonAddIcon />}
          onClick={() => setAddDialogOpen(true)}
          sx={{ whiteSpace: 'nowrap' }}
        >
          {t('contacts.add.button')}
        </Button>
      </Stack>
      {isFiltered && (
        <Box sx={{ mb: 2, p: 1.5, bgcolor: 'action.hover', borderRadius: 1, display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap' }}>
          <Typography variant="body2" sx={{ flexGrow: 1 }}>
            {searchQuery && selectedCircle 
              ? t('contacts.filteredBySearchAndCircle', { search: searchQuery, circle: selectedCircle, count: totalContacts })
              : searchQuery 
                ? t('contacts.filteredBySearch', { search: searchQuery, count: totalContacts })
                : t('contacts.filteredMessage', { count: filteredContacts.length, total: totalContacts, circle: selectedCircle })
            }
          </Typography>
          {searchQuery && (
            <Chip 
              label={`"${searchQuery}"`} 
              size="small" 
              onDelete={() => navigate('/contacts')} 
            />
          )}
          {selectedCircle && (
            <Chip 
              label={selectedCircle} 
              size="small" 
              onDelete={() => { setSelectedCircle(''); setPage(1); }} 
            />
          )}
        </Box>
      )}
      {loading ? (
        <ContactListSkeleton count={10} />
      ) : (
        <>
          <Stack spacing={2}>
            {filteredContacts.map(contact => (
              <Card 
                key={contact.ID} 
                sx={{ 
                  display: 'flex', 
                  alignItems: 'center', 
                  p: 1.5,
                  cursor: 'pointer',
                  '&:hover': {
                    bgcolor: 'action.hover'
                  }
                }}
                onClick={() => navigate(`/contacts/${contact.ID}`)}
              >
                <Avatar src={profilePics[contact.ID] || undefined} sx={{ width: 48, height: 48, mr: 1.5 }} />
                <Box sx={{ flex: 1 }}>
                  <Typography variant="body1" sx={{ fontWeight: 500 }}>
                    {contact.firstname} {contact.nickname && `"${contact.nickname}"`} {contact.lastname}
                  </Typography>
                  <Stack direction="row" spacing={0.5} mt={0.5} flexWrap="wrap" gap={0.5}>
                    {contact.circles && contact.circles.filter((circle, idx, arr) => arr.indexOf(circle) === idx).map((circle: string) => (
                      <Chip
                        key={`${contact.ID}-${circle}`}
                        label={circle}
                        size="small"
                        variant="outlined"
                        clickable
                        onClick={(e) => { e.stopPropagation(); setSelectedCircle(circle); setPage(1); }}
                        sx={{ height: 20, fontSize: '0.75rem' }}
                      />
                    ))}
                  </Stack>
                </Box>
              </Card>
            ))}
          </Stack>
          {totalContacts > 0 && (
            <Box sx={{ display: 'flex', justifyContent: 'center', mt: 2 }}>
              <Pagination
                count={Math.max(1, Math.ceil(totalContacts / pageSize))}
                page={page}
                onChange={(_, value) => setPage(value)}
                color="primary"
                size="large"
              />
            </Box>
          )}
        </>
      )}
      <AddContactDialog
        open={addDialogOpen}
        onClose={() => setAddDialogOpen(false)}
        onContactAdded={handleContactAdded}
        token={token}
        availableCircles={circles}
      />
    </Box>
  );
}
