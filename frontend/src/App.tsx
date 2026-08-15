import { useState, useEffect, Suspense, useMemo, MouseEvent as ReactMouseEvent } from 'react';
import ContactsPage from './ContactsPage';
import ContactDetailPage from './ContactDetailPage';
import ActivitiesPage from './ActivitiesPage';
import NotesPage from './NotesPage';
import DashboardPage from './DashboardPage';
import SettingsPage from './SettingsPage';
import NetworkPage from './NetworkPage';
import UsersPage from './UsersPage';
import ApiTokensPage from './ApiTokensPage';
import DataSettingsPage from './DataSettingsPage';
import LoginPage from './LoginPage';
import RegisterPage from './RegisterPage';
import { getToken, logoutAndRedirect, isAdmin, fetchAndCacheUserInfo } from './auth';
import { BrowserRouter as Router, Routes, Route, Link, useLocation, useNavigate, Navigate } from 'react-router-dom';
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
  InputAdornment,
  Collapse,
  Tooltip,
  Menu,
  MenuItem
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import ClearIcon from '@mui/icons-material/Clear';
import { getContacts, Contact } from './api/contacts';
import MenuIcon from '@mui/icons-material/Menu';
import DashboardIcon from '@mui/icons-material/Dashboard';
import ContactsIcon from '@mui/icons-material/Contacts';
import EventNoteIcon from '@mui/icons-material/EventNote';
import NoteIcon from '@mui/icons-material/Note';
import HubIcon from '@mui/icons-material/Hub';
import SettingsIcon from '@mui/icons-material/Settings';
import PeopleIcon from '@mui/icons-material/People';
import TokenIcon from '@mui/icons-material/Token';
import StorageIcon from '@mui/icons-material/Storage';
import LogoutIcon from '@mui/icons-material/Logout';
import ExpandLess from '@mui/icons-material/ExpandLess';
import ExpandMore from '@mui/icons-material/ExpandMore';
import ChevronLeftIcon from '@mui/icons-material/ChevronLeft';
import ChevronRightIcon from '@mui/icons-material/ChevronRight';
import './App.css';

const EXPANDED_DRAWER_WIDTH = 240;
const COLLAPSED_DRAWER_WIDTH = 64;
const SIDEBAR_COLLAPSED_STORAGE_KEY = 'sidebarCollapsed';

// Scroll to top on route change
function ScrollToTop() {
  const { pathname } = useLocation();

  useEffect(() => {
    window.scrollTo(0, 0);
  }, [pathname]);

  return null;
}

// Inner component that can use useLocation (must be inside Router)
function AppContent({ token, setToken }: { token: string | null; setToken: (token: string | null) => void }) {
  const { t } = useTranslation();
  const theme = useTheme();
  const location = useLocation();
  const navigate = useNavigate();
  const isMobile = useMediaQuery(theme.breakpoints.down('md'));
  const [mobileDrawerOpen, setMobileDrawerOpen] = useState(false);
  const [settingsMenuOpen, setSettingsMenuOpen] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(
    () => localStorage.getItem(SIDEBAR_COLLAPSED_STORAGE_KEY) === 'true'
  );
  const [settingsFlyoutAnchor, setSettingsFlyoutAnchor] = useState<HTMLElement | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState<Contact[]>([]);
  const [searchLoading, setSearchLoading] = useState(false);

  const drawerWidth = sidebarCollapsed ? COLLAPSED_DRAWER_WIDTH : EXPANDED_DRAWER_WIDTH;

  const handleDrawerToggle = () => {
    setMobileDrawerOpen(!mobileDrawerOpen);
  };

  const handleSidebarCollapseToggle = () => {
    setSidebarCollapsed((prev) => {
      const next = !prev;
      localStorage.setItem(SIDEBAR_COLLAPSED_STORAGE_KEY, String(next));
      return next;
    });
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
        const result = await getContacts({ search: searchQuery, limit: 10 });
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

  const handleLogout = async () => {
    await logoutAndRedirect();
  };

  const handleSearchSubmit = () => {
    if (searchQuery.trim()) {
      navigate(`/contacts?search=${encodeURIComponent(searchQuery.trim())}`);
      setSearchQuery('');
      setSearchResults([]);
    }
  };

  // eslint-disable-next-line react-hooks/exhaustive-deps -- token changes trigger admin status recalculation
  const userIsAdmin = useMemo(() => isAdmin(), [token]);

  const mainNavItems = useMemo(() => [
    { text: t('nav.dashboard'), icon: <DashboardIcon />, path: '/' },
    { text: t('nav.contacts'), icon: <ContactsIcon />, path: '/contacts' },
    { text: t('nav.activities'), icon: <EventNoteIcon />, path: '/activities' },
    { text: t('nav.notes'), icon: <NoteIcon />, path: '/notes' },
    { text: t('nav.network'), icon: <HubIcon />, path: '/network' },
  ], [t]);

  const settingsSubItems = useMemo(() => {
    const items = [
      { text: t('nav.profile'), icon: <SettingsIcon />, path: '/settings' },
      { text: t('nav.data'), icon: <StorageIcon />, path: '/settings/data' },
      { text: t('nav.integrations'), icon: <TokenIcon />, path: '/api-tokens' },
    ];
    if (userIsAdmin) {
      items.push({ text: t('nav.users'), icon: <PeopleIcon />, path: '/users' });
    }
    return items;
  }, [t, userIsAdmin]);

  const handleSettingsMenuToggle = (event: ReactMouseEvent<HTMLElement>, collapsed: boolean) => {
    if (collapsed) {
      setSettingsFlyoutAnchor(event.currentTarget);
    } else {
      setSettingsMenuOpen(!settingsMenuOpen);
    }
  };

  const handleSettingsFlyoutClose = () => {
    setSettingsFlyoutAnchor(null);
  };

  const isSettingsActive = location.pathname.startsWith('/settings') || location.pathname.startsWith('/users') || location.pathname.startsWith('/api-tokens');

  // Check if current path matches the nav item (handle exact match for "/" and prefix match for others)
  const isActiveRoute = (path: string) => {
    if (path === '/') {
      return location.pathname === '/';
    }
    return location.pathname.startsWith(path);
  };

  const renderDrawerContent = (collapsed: boolean) => (
    <Box>
      <Toolbar />
      <List>
        {mainNavItems.map((item) => (
          <ListItem key={item.text} disablePadding>
            <Tooltip title={collapsed ? item.text : ''} placement="right">
              <ListItemButton
                component={Link}
                to={item.path}
                onClick={isMobile ? handleDrawerToggle : undefined}
                selected={isActiveRoute(item.path)}
                sx={{
                  justifyContent: collapsed ? 'center' : 'flex-start',
                  px: collapsed ? 2 : 2.5,
                  '&.Mui-selected': {
                    backgroundColor: 'action.selected',
                  },
                  '&.Mui-selected:hover': {
                    backgroundColor: 'action.selected',
                  },
                }}
              >
                <ListItemIcon sx={{ minWidth: collapsed ? 0 : 40, justifyContent: 'center' }}>{item.icon}</ListItemIcon>
                {!collapsed && <ListItemText primary={item.text} />}
              </ListItemButton>
            </Tooltip>
          </ListItem>
        ))}
        {/* Settings submenu */}
        <ListItem disablePadding>
          <Tooltip title={collapsed ? t('nav.settings') : ''} placement="right">
            <ListItemButton
              onClick={(event) => handleSettingsMenuToggle(event, collapsed)}
              selected={isSettingsActive && !settingsMenuOpen}
              sx={{
                justifyContent: collapsed ? 'center' : 'flex-start',
                px: collapsed ? 2 : 2.5,
                '&.Mui-selected': {
                  backgroundColor: 'action.selected',
                },
                '&.Mui-selected:hover': {
                  backgroundColor: 'action.selected',
                },
              }}
            >
              <ListItemIcon sx={{ minWidth: collapsed ? 0 : 40, justifyContent: 'center' }}><SettingsIcon /></ListItemIcon>
              {!collapsed && <ListItemText primary={t('nav.settings')} />}
              {!collapsed && (settingsMenuOpen ? <ExpandLess /> : <ExpandMore />)}
            </ListItemButton>
          </Tooltip>
        </ListItem>
        {!collapsed && (
          <Collapse in={settingsMenuOpen} timeout="auto" unmountOnExit>
            <List component="div" disablePadding>
              {settingsSubItems.map((item) => (
                <ListItem key={item.text} disablePadding>
                  <ListItemButton
                    component={Link}
                    to={item.path}
                    onClick={isMobile ? handleDrawerToggle : undefined}
                    selected={item.path === '/settings' ? location.pathname === '/settings' : isActiveRoute(item.path)}
                    sx={{
                      pl: 4,
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
          </Collapse>
        )}
      </List>
      {collapsed && (
        <Menu
          anchorEl={settingsFlyoutAnchor}
          open={Boolean(settingsFlyoutAnchor)}
          onClose={handleSettingsFlyoutClose}
          anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
          transformOrigin={{ vertical: 'top', horizontal: 'left' }}
        >
          {settingsSubItems.map((item) => (
            <MenuItem
              key={item.text}
              component={Link}
              to={item.path}
              onClick={handleSettingsFlyoutClose}
              selected={item.path === '/settings' ? location.pathname === '/settings' : isActiveRoute(item.path)}
            >
              <ListItemIcon>{item.icon}</ListItemIcon>
              <ListItemText primary={item.text} />
            </MenuItem>
          ))}
        </Menu>
      )}
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
          ml: { md: `${drawerWidth}px` },
          transition: (theme) => theme.transitions.create(['width', 'margin'], {
            easing: theme.transitions.easing.sharp,
            duration: theme.transitions.duration.enteringScreen,
          }),
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
          <Typography 
            variant="h6" 
            component={Link} 
            to="/" 
            onClick={() => {
              setSearchQuery('');
              setSearchResults([]);
            }}
            sx={{ 
              flexGrow: 1, 
              textDecoration: 'none', 
              color: 'inherit',
              '&:hover': { opacity: 0.8 }
            }}
          >
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
              if (event.key === 'Enter') {
                handleSearchSubmit();
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
                      <IconButton
                        size="small"
                        onClick={handleSearchSubmit}
                        sx={{ color: 'rgba(255, 255, 255, 0.7)', p: 0.5 }}
                      >
                        <SearchIcon />
                      </IconButton>
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
          '& .MuiDrawer-paper': { boxSizing: 'border-box', width: EXPANDED_DRAWER_WIDTH }
        }}
      >
        {renderDrawerContent(false)}
      </Drawer>

      {/* Desktop drawer */}
      <Drawer
        variant="permanent"
        sx={{
          display: { xs: 'none', md: 'block' },
          width: drawerWidth,
          flexShrink: 0,
          whiteSpace: 'nowrap',
          transition: (theme) => theme.transitions.create('width', {
            easing: theme.transitions.easing.sharp,
            duration: theme.transitions.duration.enteringScreen,
          }),
          '& .MuiDrawer-paper': {
            width: drawerWidth,
            boxSizing: 'border-box',
            overflowX: 'hidden',
            transition: (theme) => theme.transitions.create('width', {
              easing: theme.transitions.easing.sharp,
              duration: theme.transitions.duration.enteringScreen,
            }),
          }
        }}
      >
        <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
          <Box sx={{ flexGrow: 1, overflowY: 'auto', overflowX: 'hidden' }}>
            {renderDrawerContent(sidebarCollapsed)}
          </Box>
          <Box sx={{ display: 'flex', justifyContent: sidebarCollapsed ? 'center' : 'flex-end', p: 1 }}>
            <IconButton onClick={handleSidebarCollapseToggle} size="small" aria-label={sidebarCollapsed ? t('nav.expandSidebar') : t('nav.collapseSidebar')}>
              {sidebarCollapsed ? <ChevronRightIcon /> : <ChevronLeftIcon />}
            </IconButton>
          </Box>
        </Box>
      </Drawer>

      <Box
        component="main"
        sx={{
          flexGrow: 1,
          p: 2,
          width: { md: `calc(100% - ${drawerWidth}px)` },
          mt: 7,
          transition: (theme) => theme.transitions.create('width', {
            easing: theme.transitions.easing.sharp,
            duration: theme.transitions.duration.enteringScreen,
          }),
        }}
      >
        <Routes>
          <Route path="/contacts" element={<Suspense fallback={<Box display="flex" justifyContent="center" mt={4}><CircularProgress /></Box>}><ContactsPage /></Suspense>} />
          <Route path="/contacts/:id" element={<Suspense fallback={<Box display="flex" justifyContent="center" mt={4}><CircularProgress /></Box>}><ContactDetailPage /></Suspense>} />
          <Route path="/notes" element={<Suspense fallback={<Box display="flex" justifyContent="center" mt={4}><CircularProgress /></Box>}><NotesPage /></Suspense>} />
          <Route path="/activities" element={<Suspense fallback={<Box display="flex" justifyContent="center" mt={4}><CircularProgress /></Box>}><ActivitiesPage /></Suspense>} />
          <Route path="/settings" element={<Suspense fallback={<Box display="flex" justifyContent="center" mt={4}><CircularProgress /></Box>}><SettingsPage /></Suspense>} />
          <Route path="/settings/data" element={<Suspense fallback={<Box display="flex" justifyContent="center" mt={4}><CircularProgress /></Box>}><DataSettingsPage /></Suspense>} />
          <Route path="/network" element={<Suspense fallback={<Box display="flex" justifyContent="center" mt={4}><CircularProgress /></Box>}><NetworkPage /></Suspense>} />
          <Route path="/users" element={<Suspense fallback={<Box display="flex" justifyContent="center" mt={4}><CircularProgress /></Box>}><UsersPage /></Suspense>} />
          <Route path="/api-tokens" element={<Suspense fallback={<Box display="flex" justifyContent="center" mt={4}><CircularProgress /></Box>}><ApiTokensPage /></Suspense>} />
          <Route path="/reminders" element={<div>{t('pages.reminders')}</div>} />
          <Route path="/" element={<Suspense fallback={<Box display="flex" justifyContent="center" mt={4}><CircularProgress /></Box>}><DashboardPage /></Suspense>} />
          <Route path="/login" element={<Navigate to="/" replace />} />
          <Route path="/register" element={<Navigate to="/" replace />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </Box>
    </>
  );
}

function App() {
  const [token, setToken] = useState(getToken());

  useEffect(() => {
    // Restore session after OIDC redirect: the server sets the auth cookie but
    // localStorage is empty, so we fetch user info once to populate it.
    if (!getToken()) {
      fetchAndCacheUserInfo().then(info => {
        if (info) setToken(getToken());
      });
    }
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    // Listen for storage changes (e.g., logout in another tab)
    const onStorage = (e: StorageEvent) => {
      if (e.key === 'user_info') {
        setToken(getToken());
      }
    };
    window.addEventListener('storage', onStorage);
    return () => window.removeEventListener('storage', onStorage);
  }, []);

  return (
    <Router>
      <ScrollToTop />
      <Box sx={{ display: 'flex' }}>
        <AppContent token={token} setToken={setToken} />
      </Box>
    </Router>
  );
}

export default App;
