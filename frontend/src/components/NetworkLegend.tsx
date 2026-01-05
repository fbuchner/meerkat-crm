import { Box, Typography, useTheme } from '@mui/material';
import { useTranslation } from 'react-i18next';

export default function NetworkLegend() {
  const theme = useTheme();
  const { t } = useTranslation();

  const legendItems = [
    {
      color: theme.palette.primary.main,
      label: t('network.legend.relationships'),
      type: 'line',
    },
    {
      color: theme.palette.secondary.main,
      label: t('network.legend.activities'),
      type: 'line',
    },
    {
      color: theme.palette.primary.main,
      label: t('network.legend.contact'),
      type: 'circle',
      size: 12,
    },
    {
      color: theme.palette.secondary.main,
      label: t('network.legend.activity'),
      type: 'circle',
      size: 8,
    },
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
      {legendItems.map((item, index) => (
        <Box
          key={index}
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
