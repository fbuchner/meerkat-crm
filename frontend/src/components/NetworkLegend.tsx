import { Box, Typography, useTheme } from '@mui/material';
import { useTranslation } from 'react-i18next';

interface NetworkLegendProps {
  showCircles?: boolean;
  showActivities?: boolean;
  showRelationships?: boolean;
}

export default function NetworkLegend({ showCircles, showActivities, showRelationships }: NetworkLegendProps) {
  const theme = useTheme();
  const { t } = useTranslation();

  const legendItems = [
    ...(showRelationships ? [{
      color: theme.palette.primary.main,
      label: t('network.legend.relationships'),
      type: 'line',
    }] : []),
    ...(showActivities ? [{
      color: theme.palette.secondary.main,
      label: t('network.legend.activities'),
      type: 'line',
    }] : []),
    ...(showCircles ? [{
      color: theme.palette.warning.main,
      label: t('network.legend.circleEdge'),
      type: 'line',
    }] : []),
    {
      color: theme.palette.primary.main,
      label: t('network.legend.contact'),
      type: 'circle',
      size: 12,
    },
    ...(showActivities ? [{
      color: theme.palette.secondary.main,
      label: t('network.legend.activity'),
      type: 'circle',
      size: 8,
    }] : []),
    ...(showCircles ? [{
      color: theme.palette.warning.main,
      label: t('network.legend.circle'),
      type: 'circle',
      size: 10,
    }] : []),
  ];

  return (
    <Box
      sx={{
        display: 'flex',
        flexWrap: 'wrap',
        gap: 2,
        alignItems: 'center',
      }}
    >
      {legendItems.map((item) => (
        <Box
          key={item.label}
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: 0.75,
          }}
        >
          {item.type === 'line' ? (
            <Box
              sx={{
                width: 20,
                height: 3,
                bgcolor: item.color,
                borderRadius: 1,
              }}
            />
          ) : (
            <Box
              sx={{
                width: item.size,
                height: item.size,
                bgcolor: item.color,
                borderRadius: '50%',
              }}
            />
          )}
          <Typography variant="caption" color="text.secondary">
            {item.label}
          </Typography>
        </Box>
      ))}
    </Box>
  );
}
