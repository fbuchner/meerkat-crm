import { useEffect, useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
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
  TextField,
  InputAdornment,
  Stack,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Pagination,
  Button
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import AddIcon from '@mui/icons-material/Add';
import { ContactListSkeleton } from './components/LoadingSkeletons';

export default function ContactsPage({ token }: { token: string }) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [profilePics, setProfilePics] = useState<{ [key: string]: string }>({});
  const [search, setSearch] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');
  const [selectedCircle, setSelectedCircle] = useState('');
  const [circles, setCircles] = useState<string[]>([]);
  const [page, setPage] = useState(1);
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const pageSize = 10;

  // Memoize params to prevent infinite re-renders
  const contactParams = useMemo(() => ({
    page,
    limit: pageSize,
    search: debouncedSearch,
    circle: selectedCircle
  }), [page, debouncedSearch, selectedCircle]);

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

  // Debounce search input
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1); // Reset to page 1 when search changes
    }, 400);
    return () => clearTimeout(timer);
  }, [search]);

  // Fetch profile pictures when contacts change
  useEffect(() => {
    const fetchProfilePics = async () => {
      const picPromises = contacts.map(async (contact) => {
        try {
          const blob = await getContactProfilePicture(contact.ID, token);
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
      setProfilePics(picMap);
    };
    
    if (contacts.length > 0) {
      fetchProfilePics();
    }
  }, [contacts, token]);

  // Filter contacts by selected circle
  // With backend pagination, contacts are already filtered
  const filteredContacts = contacts;

  const isFiltered = selectedCircle !== '';

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
    <Box sx={{ maxWidth: 800, mx: 'auto', mt: 4 }}>
      <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} mb={2} alignItems="center">
        <TextField
          label={t('contacts.search')}
          variant="outlined"
          value={search}
          onChange={e => setSearch(e.target.value)}
          InputProps={{
            endAdornment: (
              <InputAdornment position="end">
                <SearchIcon />
              </InputAdornment>
            )
          }}
        />
        <FormControl sx={{ minWidth: 120 }}>
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
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => setAddDialogOpen(true)}
          sx={{ whiteSpace: 'nowrap' }}
        >
          {t('contacts.add.button')}
        </Button>
      </Stack>
      {isFiltered && (
        <Box sx={{ mb: 2, p: 1, bgcolor: '#f5f5f5', borderRadius: 1, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Typography variant="body2">
            {t('contacts.filteredMessage', { count: filteredContacts.length, total: totalContacts, circle: selectedCircle })}
          </Typography>
          <Chip label={t('contacts.resetFilter')} color="primary" size="small" onClick={() => { setSelectedCircle(''); setPage(1); }} clickable />
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
                  p: 1,
                  cursor: 'pointer',
                  '&:hover': {
                    bgcolor: 'action.hover'
                  }
                }}
                onClick={() => navigate(`/contacts/${contact.ID}`)}
              >
                <Avatar src={profilePics[contact.ID] || undefined} sx={{ width: 56, height: 56, mr: 2 }} />
                <Box sx={{ flex: 1 }}>
                  <Typography variant="subtitle1" sx={{ fontWeight: 500 }}>
                    {contact.firstname} {contact.nickname && `"${contact.nickname}"`} {contact.lastname}
                  </Typography>
                  <Stack direction="row" spacing={1} mt={0.5}>
                    {contact.circles && contact.circles.map((circle: string) => (
                      <Chip
                        key={`${contact.ID}-${circle}`}
                        label={circle}
                        size="small"
                        variant="outlined"
                        clickable
                        onClick={() => { setSelectedCircle(circle); setPage(1); }}
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
