import { useState, useMemo } from 'react';
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
} from '@mui/material';
import NetworkGraph from './components/NetworkGraph';
import NetworkLegend from './components/NetworkLegend';
import { useGraph } from './hooks/useGraph';
import { GraphNode } from './types/graph';

export default function NetworkPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('md'));
  const { data, loading, error } = useGraph();

  const [selectedCircle, setSelectedCircle] = useState<string>('');
  const [showRelationships, setShowRelationships] = useState(true);
  const [showActivities, setShowActivities] = useState(true);

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

  // Handle node click - navigate to contact detail
  const handleNodeClick = (node: GraphNode) => {
    // Extract contact ID from node ID (format: "c-{id}")
    const contactId = node.id.replace('c-', '');
    navigate(`/contacts/${contactId}`);
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
    <Box sx={{ height: 'calc(100vh - 100px)', display: 'flex', flexDirection: 'column', p: 2 }}>
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
        <Typography variant="h6" sx={{ flexGrow: isMobile ? 0 : 1 }}>
          {t('network.title')}
        </Typography>

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

        {!isMobile && <NetworkLegend />}
      </Card>

      {isMobile && (
        <Card sx={{ p: 1.5, mb: 2 }}>
          <NetworkLegend />
        </Card>
      )}

      {/* Graph */}
      <Card sx={{ flex: 1, position: 'relative', overflow: 'hidden', minHeight: 400 }}>
        <NetworkGraph
          data={data}
          onNodeClick={handleNodeClick}
          selectedCircle={selectedCircle || undefined}
          showRelationships={showRelationships}
          showActivities={showActivities}
        />
      </Card>
    </Box>
  );
}
