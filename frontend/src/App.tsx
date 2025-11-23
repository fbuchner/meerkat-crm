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
  Button,
  CircularProgress,
  useTheme,
  useMediaQuery
} from '@mui/material';
import MenuIcon from '@mui/icons-material/Menu';
import DashboardIcon from '@mui/icons-material/Dashboard';
import ContactsIcon from '@mui/icons-material/Contacts';
import EventNoteIcon from '@mui/icons-material/EventNote';
import NoteIcon from '@mui/icons-material/Note';
import SettingsIcon from '@mui/icons-material/Settings';
import LogoutIcon from '@mui/icons-material/Logout';
import './App.css';

const drawerWidth = 200;

function App() {
  const { t } = useTranslation();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('md'));
  const [mobileDrawerOpen, setMobileDrawerOpen] = useState(false);
  
  const handleDrawerToggle = () => {
    setMobileDrawerOpen(!mobileDrawerOpen);
  };
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

  const navItems = [
    { text: t('nav.dashboard'), icon: <DashboardIcon />, path: '/' },
    { text: t('nav.contacts'), icon: <ContactsIcon />, path: '/contacts' },
    { text: t('nav.activities'), icon: <EventNoteIcon />, path: '/activities' },
    { text: t('nav.notes'), icon: <NoteIcon />, path: '/notes' },
    { text: t('nav.settings'), icon: <SettingsIcon />, path: '/settings' }
  ];

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
            >
              <ListItemIcon>{item.icon}</ListItemIcon>
              <ListItemText primary={item.text} />
            </ListItemButton>
          </ListItem>
        ))}
      </List>
    </Box>
  );

  // Removed duplicate token declaration. Use state version only.
  return (
    <Router>
      <Box sx={{ display: 'flex' }}>
        {token ? (
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
                p: 3,
                width: { md: `calc(100% - ${drawerWidth}px)` },
                mt: 8
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
        ) : (
          <Box sx={{ p: 2, width: '100%' }}>
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
