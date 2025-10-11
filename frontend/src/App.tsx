import React, { useState } from 'react';
import ContactsPage from './ContactsPage';
import LoginPage from './LoginPage';
import RegisterPage from './RegisterPage';
import { getToken, logoutUser } from './auth';
import { BrowserRouter as Router, Routes, Route, Link } from 'react-router-dom';
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
  Menu,
  MenuItem,
  Button,
  Select,
  FormControl,
  InputLabel
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
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [lang, setLang] = useState('en');
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const handleDrawerOpen = () => setDrawerOpen(true);
  const handleDrawerClose = () => setDrawerOpen(false);
  // Remove the custom handler and use inline in Select
    const [token, setToken] = useState(getToken());
    React.useEffect(() => {
      const onStorage = () => setToken(getToken());
      window.addEventListener('storage', onStorage);
      return () => window.removeEventListener('storage', onStorage);
    }, []);
  const handleMenuClose = () => setAnchorEl(null);
  const handleLogout = () => {
    logoutUser();
    window.location.href = '/login';
  };

  const navItems = [
    { text: 'Dashboard', icon: <DashboardIcon />, path: '/' },
    { text: 'Contacts', icon: <ContactsIcon />, path: '/contacts' },
    { text: 'Activities', icon: <EventNoteIcon />, path: '/activities' },
    { text: 'Notes', icon: <NoteIcon />, path: '/notes' }
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
                  Perema CRM
                </Typography>
                <FormControl variant="standard" sx={{ minWidth: 80, mr: 2 }}>
                  <InputLabel id="lang-select-label">
                    <LanguageIcon fontSize="small" />
                  </InputLabel>
                  <Select
                    labelId="lang-select-label"
                    id="lang-select"
                    value={lang}
                    onChange={(event) => setLang(event.target.value as string)}
                    label="Language"
                    sx={{ color: 'white' }}
                  >
                    <MenuItem value={'en'}>EN</MenuItem>
                    <MenuItem value={'de'}>DE</MenuItem>
                    <MenuItem value={'fr'}>FR</MenuItem>
                  </Select>
                </FormControl>
                <Button color="inherit" startIcon={<LogoutIcon />} onClick={handleLogout}>
                  Logout
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
                <Route path="/contacts" element={<React.Suspense fallback={<div>Loading...</div>}><ContactsPage token={token} /></React.Suspense>} />
                <Route path="/notes" element={<div>Notes Page</div>} />
                <Route path="/activities" element={<div>Activities Page</div>} />
                <Route path="/reminders" element={<div>Reminders Page</div>} />
                <Route path="/" element={<div>Welcome to Perema CRM Dashboard</div>} />
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
