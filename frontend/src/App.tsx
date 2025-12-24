import { useState, useEffect, Suspense } from 'react';
import ContactsPage from './ContactsPage';
import ContactDetailPage from './ContactDetailPage';
import ActivitiesPage from './ActivitiesPage';
import NotesPage from './NotesPage';
import DashboardPage from './DashboardPage';
import SettingsPage from './SettingsPage';
import LoginPage from './LoginPage';
import RegisterPage from './RegisterPage';
import { getToken, logoutUser } from './auth';
import { BrowserRouter as Router, Routes, Route, Link, useLocation, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  AppBar,
  Toolbar,
  IconButton,
  Typography,
  Drawer,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Box,
  Button,
  CircularProgress,
  useTheme,
  useMediaQuery,
  TextField,
  Autocomplete,
  InputAdornment
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import ClearIcon from '@mui/icons-material/Clear';
import { getContacts, Contact } from './api/contacts';
import MenuIcon from '@mui/icons-material/Menu';
import DashboardIcon from '@mui/icons-material/Dashboard';
import ContactsIcon from '@mui/icons-material/Contacts';
import EventNoteIcon from '@mui/icons-material/EventNote';
import NoteIcon from '@mui/icons-material/Note';
import SettingsIcon from '@mui/icons-material/Settings';
import LogoutIcon from '@mui/icons-material/Logout';
import './App.css';

const drawerWidth = 180;

// Inner component that can use useLocation (must be inside Router)
function AppContent({ token, setToken }: { token: string | null; setToken: (token: string | null) => void }) {
  const { t } = useTranslation();
  const theme = useTheme();
  const location = useLocation();
  const navigate = useNavigate();
  const isMobile = useMediaQuery(theme.breakpoints.down('md'));
  const [mobileDrawerOpen, setMobileDrawerOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState<Contact[]>([]);
  const [searchLoading, setSearchLoading] = useState(false);
  
  const handleDrawerToggle = () => {
    setMobileDrawerOpen(!mobileDrawerOpen);
  };

  // Debounced search for contacts
  useEffect(() => {
    if (!token || searchQuery.length < 2) {
      setSearchResults([]);
      return;
    }

    const timer = setTimeout(async () => {
      setSearchLoading(true);
      try {
        const result = await getContacts({ search: searchQuery, limit: 10 }, token);
        setSearchResults(result.contacts || []);
      } catch (err) {
        console.error('Search error:', err);
        setSearchResults([]);
      } finally {
        setSearchLoading(false);
      }
    }, 300);

    return () => clearTimeout(timer);
  }, [searchQuery, token]);

  const handleLogout = () => {
    logoutUser();
    window.location.href = '/login';
  };

  const navItems = [
    { text: t('nav.dashboard'), icon: <DashboardIcon />, path: '/' },
    { text: t('nav.contacts'), icon: <ContactsIcon />, path: '/contacts' },
    { text: t('nav.activities'), icon: <EventNoteIcon />, path: '/activities' },
    { text: t('nav.notes'), icon: <NoteIcon />, path: '/notes' },
    { text: t('nav.settings'), icon: <SettingsIcon />, path: '/settings' }
  ];

  // Check if current path matches the nav item (handle exact match for "/" and prefix match for others)
  const isActiveRoute = (path: string) => {
    if (path === '/') {
      return location.pathname === '/';
    }
    return location.pathname.startsWith(path);
  };

  const drawerContent = (
    <Box>
      <Toolbar />
      <List>
        {navItems.map((item) => (
          <ListItem key={item.text} disablePadding>
            <ListItemButton 
              component={Link} 
              to={item.path} 
              onClick={isMobile ? handleDrawerToggle : undefined}
              selected={isActiveRoute(item.path)}
              sx={{
                '&.Mui-selected': {
                  backgroundColor: 'action.selected',
                },
                '&.Mui-selected:hover': {
                  backgroundColor: 'action.selected',
                },
              }}
            >
              <ListItemIcon>{item.icon}</ListItemIcon>
              <ListItemText primary={item.text} />
            </ListItemButton>
          </ListItem>
        ))}
      </List>
    </Box>
  );

  if (!token) {
    return (
      <Box sx={{ p: 2, width: '100%' }}>
        <Routes>
          <Route path="/register" element={<RegisterPage />} />
          <Route path="*" element={<LoginPage setToken={setToken} />} />
        </Routes>
      </Box>
    );
  }

  return (
    <>
      <AppBar 
        position="fixed" 
        sx={{ 
          zIndex: (theme) => theme.zIndex.drawer + 1,
          width: { md: `calc(100% - ${drawerWidth}px)` },
          ml: { md: `${drawerWidth}px` }
        }}
      >
        <Toolbar>
          {isMobile && (
            <IconButton 
              edge="start" 
              color="inherit" 
              aria-label="menu" 
              onClick={handleDrawerToggle} 
              sx={{ mr: 2 }}
            >
              <MenuIcon />
            </IconButton>
          )}
          <Typography variant="h6" sx={{ flexGrow: 1 }}>
            {t('app.title')}
          </Typography>
          <Autocomplete
            freeSolo
            size="small"
            options={searchResults}
            getOptionLabel={(option) => 
              typeof option === 'string' 
                ? option 
                : `${option.firstname} ${option.lastname}`
            }
            loading={searchLoading}
            onInputChange={(_, value) => setSearchQuery(value)}
            onChange={(_, value) => {
              if (value && typeof value !== 'string') {
                navigate(`/contacts/${value.ID}`);
                setSearchQuery('');
                setSearchResults([]);
              }
            }}
            onKeyDown={(event) => {
              if (event.key === 'Enter' && searchQuery.trim()) {
                // Navigate to contacts page with search query
                navigate(`/contacts?search=${encodeURIComponent(searchQuery.trim())}`);
                setSearchQuery('');
                setSearchResults([]);
              }
            }}
            inputValue={searchQuery}
            renderInput={(params) => (
              <TextField
                {...params}
                placeholder={t('contacts.search')}
                variant="outlined"
                sx={{
                  width: { xs: 150, sm: 200, md: 250 },
                  mr: 2,
                  '& .MuiOutlinedInput-root': {
                    backgroundColor: 'rgba(255, 255, 255, 0.15)',
                    '&:hover': {
                      backgroundColor: 'rgba(255, 255, 255, 0.25)',
                    },
                    '& fieldset': {
                      borderColor: 'rgba(255, 255, 255, 0.3)',
                    },
                    '&:hover fieldset': {
                      borderColor: 'rgba(255, 255, 255, 0.5)',
                    },
                    '&.Mui-focused fieldset': {
                      borderColor: 'rgba(255, 255, 255, 0.7)',
                    },
                  },
                  '& .MuiInputBase-input': {
                    color: 'white',
                  },
                  '& .MuiInputBase-input::placeholder': {
                    color: 'rgba(255, 255, 255, 0.7)',
                    opacity: 1,
                  },
                }}
                InputProps={{
                  ...params.InputProps,
                  startAdornment: (
                    <InputAdornment position="start">
                      <SearchIcon sx={{ color: 'rgba(255, 255, 255, 0.7)' }} />
                    </InputAdornment>
                  ),
                  endAdornment: searchQuery ? (
                    <InputAdornment position="end">
                      <IconButton
                        size="small"
                        onClick={() => {
                          setSearchQuery('');
                          setSearchResults([]);
                          // Clear search filter if on contacts page
                          if (location.pathname.startsWith('/contacts')) {
                            navigate('/contacts');
                          }
                        }}
                        sx={{ color: 'rgba(255, 255, 255, 0.7)', p: 0.5 }}
                      >
                        <ClearIcon fontSize="small" />
                      </IconButton>
                    </InputAdornment>
                  ) : null,
                }}
              />
            )}
            renderOption={(props, option) => (
              <li {...props} key={option.ID}>
                <Box>
                  <Typography variant="body1">
                    {option.firstname} {option.lastname}
                  </Typography>
                  {option.email && (
                    <Typography variant="caption" color="text.secondary">
                      {option.email}
                    </Typography>
                  )}
                </Box>
              </li>
            )}
          />
          <Button color="inherit" startIcon={<LogoutIcon />} onClick={handleLogout}>
            {t('app.logout')}
          </Button>
        </Toolbar>
      </AppBar>

      {/* Mobile drawer */}
      <Drawer 
        variant="temporary"
        open={mobileDrawerOpen} 
        onClose={handleDrawerToggle}
        sx={{
          display: { xs: 'block', md: 'none' },
          '& .MuiDrawer-paper': { boxSizing: 'border-box', width: drawerWidth }
        }}
      >
        {drawerContent}
      </Drawer>

      {/* Desktop drawer */}
      <Drawer
        variant="permanent"
        sx={{
          display: { xs: 'none', md: 'block' },
          width: drawerWidth,
          flexShrink: 0,
          '& .MuiDrawer-paper': { 
            width: drawerWidth, 
            boxSizing: 'border-box' 
          }
        }}
      >
        {drawerContent}
      </Drawer>

      <Box 
        component="main" 
        sx={{ 
          flexGrow: 1, 
          p: 2,
          width: { md: `calc(100% - ${drawerWidth}px)` },
          mt: 7
        }}
      >
        <Routes>
          <Route path="/contacts" element={<Suspense fallback={<Box display="flex" justifyContent="center" mt={4}><CircularProgress /></Box>}><ContactsPage token={token} /></Suspense>} />
          <Route path="/contacts/:id" element={<Suspense fallback={<Box display="flex" justifyContent="center" mt={4}><CircularProgress /></Box>}><ContactDetailPage token={token} /></Suspense>} />
          <Route path="/notes" element={<Suspense fallback={<Box display="flex" justifyContent="center" mt={4}><CircularProgress /></Box>}><NotesPage token={token} /></Suspense>} />
          <Route path="/activities" element={<Suspense fallback={<Box display="flex" justifyContent="center" mt={4}><CircularProgress /></Box>}><ActivitiesPage token={token} /></Suspense>} />
          <Route path="/settings" element={<Suspense fallback={<Box display="flex" justifyContent="center" mt={4}><CircularProgress /></Box>}><SettingsPage /></Suspense>} />
          <Route path="/reminders" element={<div>{t('pages.reminders')}</div>} />
          <Route path="/" element={<Suspense fallback={<Box display="flex" justifyContent="center" mt={4}><CircularProgress /></Box>}><DashboardPage token={token} /></Suspense>} />
        </Routes>
      </Box>
    </>
  );
}

function App() {
  const [token, setToken] = useState(getToken());
  
  useEffect(() => {
    const onStorage = () => setToken(getToken());
    window.addEventListener('storage', onStorage);
    return () => window.removeEventListener('storage', onStorage);
  }, []);

  return (
    <Router>
      <Box sx={{ display: 'flex' }}>
        <AppContent token={token} setToken={setToken} />
      </Box>
    </Router>
  );
}

export default App;
