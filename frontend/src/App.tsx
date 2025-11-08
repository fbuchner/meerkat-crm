import { useState, useEffect, Suspense } from 'react';
import ContactsPage from './ContactsPage';
import ContactDetailPage from './ContactDetailPage';
import ActivitiesPage from './ActivitiesPage';
import NotesPage from './NotesPage';
import DashboardPage from './DashboardPage';
import LoginPage from './LoginPage';
import RegisterPage from './RegisterPage';
import { getToken, logoutUser } from './auth';
import { BrowserRouter as Router, Routes, Route, Link } from 'react-router-dom';
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
  MenuItem,
  Button,
  Select,
  FormControl,
  InputLabel,
  CircularProgress
} from '@mui/material';
import MenuIcon from '@mui/icons-material/Menu';
import DashboardIcon from '@mui/icons-material/Dashboard';
import ContactsIcon from '@mui/icons-material/Contacts';
import EventNoteIcon from '@mui/icons-material/EventNote';
import NoteIcon from '@mui/icons-material/Note';
import LogoutIcon from '@mui/icons-material/Logout';
import LanguageIcon from '@mui/icons-material/Language';
import './App.css';

function App() {
  const { t, i18n } = useTranslation();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const handleDrawerOpen = () => setDrawerOpen(true);
  const handleDrawerClose = () => setDrawerOpen(false);
  // Remove the custom handler and use inline in Select
    const [token, setToken] = useState(getToken());
    useEffect(() => {
      const onStorage = () => setToken(getToken());
      window.addEventListener('storage', onStorage);
      return () => window.removeEventListener('storage', onStorage);
    }, []);
  const handleLogout = () => {
    logoutUser();
    window.location.href = '/login';
  };

  const handleLanguageChange = (newLang: string) => {
    i18n.changeLanguage(newLang);
  };

  const navItems = [
    { text: t('nav.dashboard'), icon: <DashboardIcon />, path: '/' },
    { text: t('nav.contacts'), icon: <ContactsIcon />, path: '/contacts' },
    { text: t('nav.activities'), icon: <EventNoteIcon />, path: '/activities' },
    { text: t('nav.notes'), icon: <NoteIcon />, path: '/notes' }
  ];

  // Removed duplicate token declaration. Use state version only.
  return (
    <Router>
      <Box sx={{ flexGrow: 1 }}>
        {token ? (
          <>
            <AppBar position="static">
              <Toolbar>
                <IconButton edge="start" color="inherit" aria-label="menu" onClick={handleDrawerOpen} sx={{ mr: 2 }}>
                  <MenuIcon />
                </IconButton>
                <Typography variant="h6" sx={{ flexGrow: 1 }}>
                  {t('app.title')}
                </Typography>
                <FormControl variant="standard" sx={{ minWidth: 80, mr: 2 }}>
                  <InputLabel id="lang-select-label">
                    <LanguageIcon fontSize="small" />
                  </InputLabel>
                  <Select
                    labelId="lang-select-label"
                    id="lang-select"
                    value={i18n.language}
                    onChange={(event) => handleLanguageChange(event.target.value as string)}
                    label="Language"
                    sx={{ color: 'white' }}
                  >
                    <MenuItem value={'en'}>EN</MenuItem>
                    <MenuItem value={'de'}>DE</MenuItem>
                  </Select>
                </FormControl>
                <Button color="inherit" startIcon={<LogoutIcon />} onClick={handleLogout}>
                  {t('app.logout')}
                </Button>
              </Toolbar>
            </AppBar>
            <Drawer anchor="left" open={drawerOpen} onClose={handleDrawerClose}>
              <List>
                {navItems.map((item) => (
                  <ListItem key={item.text} disablePadding>
                    <ListItemButton component={Link} to={item.path} onClick={handleDrawerClose}>
                      <ListItemIcon>{item.icon}</ListItemIcon>
                      <ListItemText primary={item.text} />
                    </ListItemButton>
                  </ListItem>
                ))}
              </List>
            </Drawer>
            <Box sx={{ p: 2 }}>
              <Routes>
                <Route path="/contacts" element={<Suspense fallback={<Box display="flex" justifyContent="center" mt={4}><CircularProgress /></Box>}><ContactsPage token={token} /></Suspense>} />
                <Route path="/contacts/:id" element={<Suspense fallback={<Box display="flex" justifyContent="center" mt={4}><CircularProgress /></Box>}><ContactDetailPage token={token} /></Suspense>} />
                <Route path="/notes" element={<Suspense fallback={<Box display="flex" justifyContent="center" mt={4}><CircularProgress /></Box>}><NotesPage token={token} /></Suspense>} />
                <Route path="/activities" element={<Suspense fallback={<Box display="flex" justifyContent="center" mt={4}><CircularProgress /></Box>}><ActivitiesPage token={token} /></Suspense>} />
                <Route path="/reminders" element={<div>{t('pages.reminders')}</div>} />
                <Route path="/" element={<Suspense fallback={<Box display="flex" justifyContent="center" mt={4}><CircularProgress /></Box>}><DashboardPage token={token} /></Suspense>} />
              </Routes>
            </Box>
          </>
        ) : (
          <Box sx={{ p: 2 }}>
            <Routes>
              <Route path="/register" element={<RegisterPage />} />
              <Route path="*" element={<LoginPage setToken={setToken} />} />
            </Routes>
          </Box>
        )}
      </Box>
    </Router>
  );
}

export default App;
