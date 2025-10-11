import React, { useEffect, useState } from 'react';
import { fetchContacts, API_BASE_URL } from './api';
import {
  Box,
  Card,
  Avatar,
  Typography,
  Chip,
  TextField,
  CircularProgress,
  InputAdornment,
  Stack,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Pagination
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';



function debounce<T extends (...args: any[]) => void>(fn: T, delay: number): T {
  let timer: NodeJS.Timeout;
  return ((...args: any[]) => {
    clearTimeout(timer);
    timer = setTimeout(() => fn(...args), delay);
  }) as T;
}

export default function ContactsPage({ token }: { token: string }) {
  const [contacts, setContacts] = useState<any[]>([]);
  const [profilePics, setProfilePics] = useState<{ [key: string]: string }>({});
  const [loading, setLoading] = useState(false);
  const [search, setSearch] = useState('');
  const [selectedCircle, setSelectedCircle] = useState('');
  const [circles, setCircles] = useState<string[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(10);
  const [totalContacts, setTotalContacts] = useState(0);

  // Fetch circles for filter
  useEffect(() => {
    fetch(`${API_BASE_URL}/contacts/circles`, {
      headers: { 'Authorization': `Bearer ${token}` }
    })
      .then(res => res.json())
      .then(data => setCircles(Array.isArray(data) ? data : []));
  }, []);

  // Debounced search and pagination
  const debouncedSearch = debounce((value: string, pageNum: number = 1) => {
    setLoading(true);
    let url = `${API_BASE_URL}/contacts?page=${pageNum}&size=${pageSize}`;
    if (value) url += `&search=${encodeURIComponent(value)}`;
    if (selectedCircle) url += `&circle=${encodeURIComponent(selectedCircle)}`;
    fetch(url, {
      headers: { 'Authorization': `Bearer ${token}` }
    })
      .then(res => res.json())
      .then(async data => {
        const contactsArr = data.contacts || [];
        setContacts(contactsArr);
        setTotalContacts(data.total || contactsArr.length);
        // Fetch profile pictures for each contact
        const picPromises = contactsArr.map(async (contact: any) => {
          try {
            const res = await fetch(`${API_BASE_URL}/contacts/${contact.ID}/profile_picture`, {
              headers: { 'Authorization': `Bearer ${token}` }
            });
            if (res.ok) {
              const blob = await res.blob();
              return { id: contact.ID, url: URL.createObjectURL(blob) };
            }
          } catch {}
          return { id: contact.ID, url: '' };
        });
        const picResults = await Promise.all(picPromises);
        const picMap: { [key: string]: string } = {};
        picResults.forEach(({ id, url }) => { picMap[id] = url; });
        setProfilePics(picMap);
        setLoading(false);
      });
  }, 400);

  useEffect(() => {
    debouncedSearch(search, page);
  }, [search, page, selectedCircle]);

  // Filter contacts by selected circle
  // With backend pagination, contacts is already filtered
  const filteredContacts = contacts;

  const isFiltered = selectedCircle !== '';
  return (
    <Box sx={{ maxWidth: 800, mx: 'auto', mt: 4 }}>
      <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} mb={2}>
        <TextField
          label="Search Contacts"
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
          <InputLabel id="circle-select-label">Circle</InputLabel>
          <Select
            labelId="circle-select-label"
            value={selectedCircle}
            label="Circle"
            onChange={e => setSelectedCircle(e.target.value)}
          >
            <MenuItem value="">All</MenuItem>
            {circles.map(circle => (
              <MenuItem key={circle} value={circle}>{circle}</MenuItem>
            ))}
          </Select>
        </FormControl>
      </Stack>
      {isFiltered && (
        <Box sx={{ mb: 2, p: 1, bgcolor: '#f5f5f5', borderRadius: 1, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Typography variant="body2">
            Showing {filteredContacts.length} out of {totalContacts} contacts in "{selectedCircle}"&nbsp;
          </Typography>
          <Chip label="Reset filter" color="primary" size="small" onClick={() => { setSelectedCircle(''); setPage(1); }} clickable />
        </Box>
      )}
      {loading ? (
        <CircularProgress />
      ) : (
        <>
          <Stack spacing={2}>
            {filteredContacts.map(contact => (
              <Card key={contact.ID} sx={{ display: 'flex', alignItems: 'center', p: 1 }}>
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
          <Box sx={{ display: 'flex', justifyContent: 'center', mt: 2 }}>
            <Pagination
              count={Math.ceil(totalContacts / pageSize)}
              page={page}
              onChange={(_, value) => setPage(value)}
              color="primary"
              size="large"
            />
          </Box>
        </>
      )}
    </Box>
  );
}
