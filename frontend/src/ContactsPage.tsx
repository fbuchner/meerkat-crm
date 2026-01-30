import { useEffect, useState, useMemo, useRef, useCallback } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useContacts } from './hooks/useContacts';
import { getCircles } from './api/contacts';
import { getCustomFieldNames } from './api/users';
import AddContactDialog from './components/AddContactDialog';
import ImportContactsDialog from './components/ImportContactsDialog';
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
import FileUploadIcon from '@mui/icons-material/FileUpload';
import { ContactListSkeleton } from './components/LoadingSkeletons';

export default function ContactsPage({ token }: { token: string }) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const searchQuery = searchParams.get('search') || '';
  const page = parseInt(searchParams.get('page') || '1', 10);
  const [selectedCircle, setSelectedCircle] = useState('');
  const [circles, setCircles] = useState<string[]>([]);
  const [sortOption, setSortOption] = useState(() => {
    return localStorage.getItem('contacts-sort-option') || 'id-desc';
  });
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [importDialogOpen, setImportDialogOpen] = useState(false);
  const [customFieldNames, setCustomFieldNames] = useState<string[]>([]);
  const pageSize = 10;

  // Parse sort option into field and order
  const [sortField, sortOrder] = sortOption.split('-');

  // Persist sort option to localStorage
  useEffect(() => {
    localStorage.setItem('contacts-sort-option', sortOption);
  }, [sortOption]);

  // Helper to update page in URL
  const setPage = useCallback((newPage: number) => {
    setSearchParams(prev => {
      const params = new URLSearchParams(prev);
      if (newPage === 1) {
        params.delete('page');
      } else {
        params.set('page', String(newPage));
      }
      return params;
    });
  }, [setSearchParams]);

  // Memoize params to prevent infinite re-renders
  const contactParams = useMemo(() => ({
    page,
    limit: pageSize,
    search: searchQuery,
    circle: selectedCircle,
    sort: sortField,
    order: sortOrder,
  }), [page, searchQuery, selectedCircle, sortField, sortOrder]);

  // Use custom hook for fetching contacts
  const { contacts, total: totalContacts, loading, refetch } = useContacts(contactParams);

  // Fetch circles for filter and custom field names
  useEffect(() => {
    const fetchData = async () => {
      try {
        const [circlesData, fieldNames] = await Promise.all([
          getCircles(token),
          getCustomFieldNames(token)
        ]);
        setCircles(Array.isArray(circlesData) ? circlesData : []);
        setCustomFieldNames(fieldNames);
      } catch (err) {
        console.error('Error fetching circles or custom field names:', err);
      }
    };
    fetchData();
  }, [token]);

  // Reset to page 1 when search or filter changes (but not on initial mount)
  const prevFiltersRef = useRef({ searchQuery, selectedCircle });
  useEffect(() => {
    const prev = prevFiltersRef.current;
    if (prev.searchQuery !== searchQuery || prev.selectedCircle !== selectedCircle) {
      setPage(1);
      prevFiltersRef.current = { searchQuery, selectedCircle };
    }
  }, [searchQuery, selectedCircle, setPage]);

  // Filter contacts by selected circle
  // With backend pagination, contacts are already filtered
  const filteredContacts = contacts;

  const handleContactAdded = (contactId: number) => {
    navigate(`/contacts/${contactId}`);
  };

  const handleImportComplete = async () => {
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
        <FormControl sx={{ minWidth: 150 }} size="small">
          <InputLabel id="sort-select-label">{t('contacts.sortBy')}</InputLabel>
          <Select
            labelId="sort-select-label"
            value={sortOption}
            label={t('contacts.sortBy')}
            onChange={e => setSortOption(e.target.value)}
          >
            <MenuItem value="id-desc">{t('contacts.sort.recentlyAdded')}</MenuItem>
            <MenuItem value="id-asc">{t('contacts.sort.oldestFirst')}</MenuItem>
            <MenuItem value="firstname-asc">{t('contacts.sort.nameAZ')}</MenuItem>
            <MenuItem value="firstname-desc">{t('contacts.sort.nameZA')}</MenuItem>
            <MenuItem value="random-asc">{t('contacts.sort.random')}</MenuItem>
          </Select>
        </FormControl>
        <Button
          variant="outlined"
          startIcon={<FileUploadIcon />}
          onClick={() => setImportDialogOpen(true)}
          sx={{ whiteSpace: 'nowrap' }}
        >
          {t('contacts.import.button', 'Import')}
        </Button>
        <Button
          variant="outlined"
          startIcon={<PersonAddIcon />}
          onClick={() => setAddDialogOpen(true)}
          sx={{ whiteSpace: 'nowrap' }}
        >
          {t('contacts.add.button')}
        </Button>
      </Stack>
      {totalContacts > 0 && (
        <Box sx={{ mb: 2, p: 1.5, bgcolor: 'action.hover', borderRadius: 1, display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap' }}>
          <Typography variant="body2" sx={{ flexGrow: 1 }}>
            {searchQuery && selectedCircle 
              ? t('contacts.filteredBySearchAndCircle', { search: searchQuery, circle: selectedCircle, count: totalContacts })
              : searchQuery 
                ? t('contacts.filteredBySearch', { search: searchQuery, count: totalContacts })
                : selectedCircle
                  ? t('contacts.filteredMessage', { count: filteredContacts.length, total: totalContacts, circle: selectedCircle })
                  : t('contacts.totalContacts', { count: totalContacts })
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
                <Avatar src={contact.photo_thumbnail || undefined} sx={{ width: 48, height: 48, mr: 1.5, bgcolor: 'primary.main' }}>
                  {contact.firstname.charAt(0)}
                </Avatar>
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
        customFieldNames={customFieldNames}
      />
      <ImportContactsDialog
        open={importDialogOpen}
        onClose={() => setImportDialogOpen(false)}
        onImportComplete={handleImportComplete}
        token={token}
      />
    </Box>
  );
}
