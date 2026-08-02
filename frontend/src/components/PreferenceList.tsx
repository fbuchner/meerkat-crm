import { Box, Typography, IconButton, Stack, Paper, Chip } from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import { useTranslation } from 'react-i18next';
import { Preference } from '../api/preferences';

interface PreferenceListProps {
  preferences: Preference[];
  onEdit: (preference: Preference) => void;
  onDelete: (id: string) => void;
}

export default function PreferenceList({ preferences, onEdit, onDelete }: PreferenceListProps) {
  const { t } = useTranslation();

  if (preferences.length === 0) {
    return (
      <Typography variant="body2" color="text.secondary" sx={{ py: 2, textAlign: 'center' }}>
        {t('preference.empty')}
      </Typography>
    );
  }

  return (
    <Stack spacing={1.5}>
      {preferences.map((pref) => (
        <Paper
          key={pref.id}
          variant="outlined"
          sx={{
            p: 2,
            '&:hover .preference-actions': { opacity: 1 },
          }}
        >
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
            <Box sx={{ flex: 1 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 0.5, flexWrap: 'wrap' }}>
                <Chip
                  label={t(`preference.categories.${pref.category}`, pref.category)}
                  size="small"
                  color="primary"
                  variant="outlined"
                />
                {pref.sensitivity !== 'normal' && (
                  <Chip
                    label={t(`preference.sensitivities.${pref.sensitivity}`)}
                    size="small"
                    sx={{ height: 18 }}
                  />
                )}
              </Box>
              <Typography variant="body1">{pref.value}</Typography>
              {pref.key && (
                <Typography variant="caption" color="text.secondary">
                  {pref.key}
                </Typography>
              )}
            </Box>
            <Box
              className="preference-actions"
              sx={{ display: 'flex', gap: 0.5, opacity: 0, transition: 'opacity 0.2s ease-in-out' }}
            >
              <IconButton size="small" onClick={() => onEdit(pref)} aria-label={t('common.edit')}>
                <EditIcon fontSize="small" />
              </IconButton>
              <IconButton
                size="small"
                color="error"
                onClick={() => onDelete(pref.id)}
                aria-label={t('common.delete')}
              >
                <DeleteIcon fontSize="small" />
              </IconButton>
            </Box>
          </Box>
        </Paper>
      ))}
    </Stack>
  );
}
