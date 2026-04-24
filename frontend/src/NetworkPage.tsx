import { useState, useMemo, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  Box,
  Card,
  Typography,
  CircularProgress,
  FormControlLabel,
  Switch,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Alert,
  SelectChangeEvent,
  useMediaQuery,
  useTheme,
  Autocomplete,
  TextField,
} from '@mui/material';
import NetworkGraph from './components/NetworkGraph';
import NetworkLegend from './components/NetworkLegend';
import EditTimelineItemDialog from './components/EditTimelineItemDialog';
import { useGraph } from './hooks/useGraph';
import { GraphNode } from './types/graph';
import { Activity, getActivity, updateActivity, deleteActivity } from './api/activities';
import { Contact, getContacts } from './api/contacts';

export default function NetworkPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('md'));
  const { data, loading, error } = useGraph();

  const [selectedCircle, setSelectedCircle] = useState<string>('');
  const [showRelationships, setShowRelationships] = useState(true);
  const [showActivities, setShowActivities] = useState(true);
  const [showCircles, setShowCircles] = useState(false);
  const [centeredNodeId, setCenteredNodeId] = useState<string | null>(() => {
    return localStorage.getItem('network-centered-node-id');
  });
  const [editingActivity, setEditingActivity] = useState<Activity | null>(null);
  const [editValues, setEditValues] = useState<{
    activityTitle?: string;
    activityDescription?: string;
    activityLocation?: string;
    activityDate?: string;
    activityContacts?: Contact[];
  }>({});
  const [allContacts, setAllContacts] = useState<Contact[]>([]);

  useEffect(() => {
    if (centeredNodeId !== null) {
      localStorage.setItem('network-centered-node-id', centeredNodeId);
    } else {
      localStorage.removeItem('network-centered-node-id');
    }
  }, [centeredNodeId]);

  useEffect(() => {
    if (!data || !centeredNodeId) return;
    const nodeExists = data.nodes.some(n => n.id === centeredNodeId);
    if (!nodeExists) {
      setCenteredNodeId(null);
    }
  }, [data, centeredNodeId]);

  // Extract unique circles from contacts
  const circles = useMemo(() => {
    if (!data) return [];
    const allCircles = new Set<string>();
    data.nodes.forEach(n => {
      if (n.type === 'contact' && n.circles) {
        n.circles.forEach(c => allCircles.add(c));
      }
    });
    return Array.from(allCircles).sort();
  }, [data]);

  // Contact nodes for the center-on-contact autocomplete
  const contactNodes = useMemo(() => {
    if (!data) return [];
    return data.nodes.filter(n => n.type === 'contact').sort((a, b) => a.label.localeCompare(b.label));
  }, [data]);

  // Handle node click - navigate to contact detail
  const handleNodeClick = (node: GraphNode) => {
    if (node.type === 'contact') {
      const contactId = node.id.replace('c-', '');
      navigate(`/contacts/${contactId}`);
    }
  };

  const handleActivityNodeClick = async (node: GraphNode) => {
    const id = parseInt(node.id.replace('a-', ''), 10);
    try {
      const activity = await getActivity(id);
      setEditingActivity(activity);
      if (allContacts.length === 0) {
        const contactsResponse = await getContacts({ page: 1, limit: 1000 });
        setAllContacts(contactsResponse.contacts || []);
      }
      setEditValues({
        activityTitle: activity.title || '',
        activityDescription: activity.description || '',
        activityLocation: activity.location || '',
        activityDate: activity.date ? new Date(activity.date).toISOString().split('T')[0] : '',
        activityContacts: activity.contacts || [],
      });
    } catch (err) {
      console.error('Failed to fetch activity:', err);
    }
  };

  const handleActivityEditClose = () => {
    setEditingActivity(null);
    setEditValues({});
  };

  const handleActivitySave = async () => {
    if (!editingActivity || !editValues.activityTitle?.trim()) return;
    try {
      await updateActivity(editingActivity.ID, {
        title: editValues.activityTitle,
        description: editValues.activityDescription || '',
        location: editValues.activityLocation || '',
        date: editValues.activityDate ? new Date(editValues.activityDate).toISOString() : new Date().toISOString(),
        contact_ids: editValues.activityContacts?.map(c => c.ID) || [],
      });
      handleActivityEditClose();
    } catch (err) {
      console.error('Failed to update activity:', err);
    }
  };

  const handleActivityDelete = async () => {
    if (!editingActivity) return;
    try {
      await deleteActivity(editingActivity.ID);
      handleActivityEditClose();
    } catch (err) {
      console.error('Failed to delete activity:', err);
    }
  };

  const handleCircleChange = (event: SelectChangeEvent<string>) => {
    setSelectedCircle(event.target.value);
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="60vh">
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Box p={2}>
        <Alert severity="error">{error}</Alert>
      </Box>
    );
  }

  if (!data || data.nodes.length === 0) {
    return (
      <Box p={2}>
        <Alert severity="info">{t('network.noData')}</Alert>
      </Box>
    );
  }

  return (
    <Box sx={{ height: 'calc(100vh - 100px)', display: 'flex', flexDirection: 'column', mt: 2, p: 2 }}>
      <Typography variant="h5" gutterBottom sx={{ mb: 2 }}>
        {t('network.title')}
      </Typography>
      {/* Controls */}
      <Card
        sx={{
          p: 2,
          mb: 2,
          display: 'flex',
          gap: 2,
          flexWrap: 'wrap',
          alignItems: 'center',
          flexDirection: isMobile ? 'column' : 'row',
          flexShrink: 0,
          overflow: 'visible',
        }}
      >
        <Autocomplete
          size="small"
          sx={{ minWidth: 200 }}
          options={contactNodes}
          getOptionLabel={(n) => n.label}
          value={contactNodes.find(n => n.id === centeredNodeId) ?? null}
          onChange={(_e, node) => setCenteredNodeId(node ? node.id : null)}
          renderInput={(params) => (
            <TextField {...params} label={t('network.filterByContact')} />
          )}
          clearOnEscape
        />

        <FormControl size="small" sx={{ minWidth: 150 }}>
          <InputLabel>{t('network.filterByCircle')}</InputLabel>
          <Select
            value={selectedCircle}
            onChange={handleCircleChange}
            label={t('network.filterByCircle')}
          >
            <MenuItem value="">{t('network.allCircles')}</MenuItem>
            {circles.map(c => (
              <MenuItem key={c} value={c}>{c}</MenuItem>
            ))}
          </Select>
        </FormControl>

        <FormControlLabel
          control={
            <Switch
              checked={showRelationships}
              onChange={(e) => setShowRelationships(e.target.checked)}
              color="primary"
            />
          }
          label={t('network.showRelationships')}
        />

        <FormControlLabel
          control={
            <Switch
              checked={showActivities}
              onChange={(e) => setShowActivities(e.target.checked)}
              color="secondary"
            />
          }
          label={t('network.showActivities')}
        />

        <FormControlLabel
          control={
            <Switch
              checked={showCircles}
              onChange={(e) => setShowCircles(e.target.checked)}
              color="warning"
            />
          }
          label={t('network.showCircles')}
        />

        {!isMobile && (
          <Box sx={{ flexBasis: '100%' }}>
            <NetworkLegend showCircles={showCircles} showActivities={showActivities} showRelationships={showRelationships} />
          </Box>
        )}
      </Card>

      {isMobile && (
        <Card sx={{ p: 1.5, mb: 2 }}>
          <NetworkLegend showCircles={showCircles} showActivities={showActivities} showRelationships={showRelationships} />
        </Card>
      )}

      {/* Graph */}
      <Card sx={{ flex: 1, position: 'relative', overflow: 'hidden', minHeight: 400 }}>
        <NetworkGraph
          data={data}
          onNodeClick={handleNodeClick}
          onActivityClick={handleActivityNodeClick}
          selectedCircle={selectedCircle || undefined}
          showRelationships={showRelationships}
          showActivities={showActivities}
          showCircles={showCircles}
          centeredNodeId={centeredNodeId ?? undefined}
        />
      </Card>
      {editingActivity && (
        <EditTimelineItemDialog
          open
          onClose={handleActivityEditClose}
          onSave={handleActivitySave}
          onDelete={handleActivityDelete}
          type="activity"
          values={editValues}
          onChange={setEditValues}
          allContacts={allContacts}
        />
      )}
    </Box>
  );
}
