import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Divider,
  Stack,
  Alert,
  Button,
  Radio,
  RadioGroup,
  FormControlLabel,
  Autocomplete,
  TextField,
  Chip,
  CircularProgress,
} from '@mui/material';
import PhoneIphoneIcon from '@mui/icons-material/PhoneIphone';
import {
  CardDAVCircleMode,
  getCardDAVCircleFilter,
  updateCardDAVCircleFilter,
} from '../api/users';
import { getCircles } from '../api/contacts';

export default function CardDAVCircleFilterSettings() {
  const { t } = useTranslation();
  const [mode, setMode] = useState<CardDAVCircleMode>('all');
  const [circles, setCircles] = useState<string[]>([]);
  const [availableCircles, setAvailableCircles] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  // Without the stored filter the form would show a default that saving would
  // write over the real setting, so it stays hidden until the filter loads.
  const [filterLoaded, setFilterLoaded] = useState(false);
  const [circlesLoaded, setCirclesLoaded] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  useEffect(() => {
    const load = async () => {
      setLoading(true);
      const [filterResult, circlesResult] = await Promise.allSettled([getCardDAVCircleFilter(), getCircles()]);
      if (filterResult.status === 'fulfilled') {
        setMode(filterResult.value.mode);
        setCircles(filterResult.value.circles);
        setFilterLoaded(true);
      } else {
        const err = filterResult.reason;
        setError(err instanceof Error ? err.message : t('settings.carddavCircleFilter.loadError'));
      }
      // Without the circle list the stored circles still show and can be removed.
      if (circlesResult.status === 'fulfilled') {
        setAvailableCircles(circlesResult.value);
        setCirclesLoaded(true);
      }
      setLoading(false);
    };
    load();
  }, [t]);

  const handleSave = async () => {
    try {
      setSaving(true);
      setError('');
      setSuccess('');
      const saved = await updateCardDAVCircleFilter({ mode, circles: mode === 'all' ? [] : circles });
      setMode(saved.mode);
      setCircles(saved.circles);
      setSuccess(t('settings.carddavCircleFilter.saveSuccess'));
    } catch (err) {
      setError(err instanceof Error ? err.message : t('settings.carddavCircleFilter.saveError'));
    } finally {
      setSaving(false);
    }
  };

  const needsCircles = mode !== 'all' && circles.length === 0;
  const isInUse = (circle: string) =>
    availableCircles.some((c) => c.toLowerCase() === circle.toLowerCase());
  // Circles no contact belongs to any more. An include filter made only of
  // these would empty every device's address book.
  const unusedCircles = circlesLoaded ? circles.filter((c) => !isInUse(c)) : [];
  const options = [...availableCircles, ...circles.filter((c) => !isInUse(c))];

  return (
    <Card sx={{ mb: 2 }}>
      <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
        <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
          <PhoneIphoneIcon sx={{ mr: 1, color: 'text.secondary', fontSize: 20 }} />
          <Typography variant="subtitle1" sx={{ fontWeight: 500 }}>
            {t('settings.carddavCircleFilter.title')}
          </Typography>
        </Box>
        <Divider sx={{ mb: 1.5 }} />

        <Stack spacing={1.5}>
          <Typography variant="body2" color="text.secondary">
            {t('settings.carddavCircleFilter.description')}
          </Typography>

          {error && <Alert severity="error" sx={{ py: 0 }}>{error}</Alert>}
          {success && <Alert severity="success" sx={{ py: 0 }}>{success}</Alert>}

          {loading ? (
            <Typography variant="body2" color="text.secondary">
              {t('settings.carddavCircleFilter.loading')}
            </Typography>
          ) : !filterLoaded ? null : (
            <>
              <RadioGroup
                value={mode}
                onChange={(e) => {
                  setMode(e.target.value as CardDAVCircleMode);
                  setSuccess('');
                }}
              >
                <FormControlLabel value="all" control={<Radio size="small" />} label={t('settings.carddavCircleFilter.modeAll')} />
                <FormControlLabel value="include" control={<Radio size="small" />} label={t('settings.carddavCircleFilter.modeInclude')} />
                <FormControlLabel value="exclude" control={<Radio size="small" />} label={t('settings.carddavCircleFilter.modeExclude')} />
              </RadioGroup>

              {mode !== 'all' && (
                <Autocomplete
                  multiple
                  size="small"
                  options={options}
                  value={circles}
                  onChange={(_, value) => {
                    setCircles(value);
                    setSuccess('');
                  }}
                  renderTags={(value, getTagProps) =>
                    value.map((option, index) => {
                      const { key, ...tagProps } = getTagProps({ index });
                      return (
                        <Chip
                          key={key}
                          label={option}
                          size="small"
                          color={unusedCircles.includes(option) ? 'warning' : 'default'}
                          {...tagProps}
                        />
                      );
                    })
                  }
                  renderInput={(params) => (
                    <TextField
                      {...params}
                      label={t('settings.carddavCircleFilter.circlesLabel')}
                      error={needsCircles}
                      helperText={needsCircles ? t('settings.carddavCircleFilter.circlesRequired') : undefined}
                    />
                  )}
                  noOptionsText={t('settings.carddavCircleFilter.noCircles')}
                />
              )}

              {mode !== 'all' && unusedCircles.length > 0 && (
                <Alert severity="warning" sx={{ py: 0 }}>
                  {mode === 'include' && unusedCircles.length === circles.length
                    ? t('settings.carddavCircleFilter.allCirclesUnused')
                    : t('settings.carddavCircleFilter.unusedCircles', { circles: unusedCircles.join(', ') })}
                </Alert>
              )}
              {mode === 'include' && (
                <Alert severity="info" sx={{ py: 0 }}>
                  {t('settings.carddavCircleFilter.includeNewContactsHint')}
                </Alert>
              )}
              {mode !== 'all' && (
                <Typography variant="body2" color="text.secondary">
                  {t('settings.carddavCircleFilter.removalHint')}
                </Typography>
              )}

              <Box>
                <Button
                  variant="contained"
                  size="small"
                  onClick={handleSave}
                  disabled={saving || needsCircles}
                  startIcon={saving ? <CircularProgress size={16} color="inherit" /> : undefined}
                >
                  {t('settings.carddavCircleFilter.save')}
                </Button>
              </Box>
            </>
          )}
        </Stack>
      </CardContent>
    </Card>
  );
}
