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
  FormControlLabel,
  Switch,
} from '@mui/material';
import ViewListIcon from '@mui/icons-material/ViewList';
import { updateEnabledContactFields, updateContactFieldDisplay } from '../api/users';
import { getCurrentUser } from '../api/admin';
import {
  CONTACT_FIELDS,
  CONTACT_FIELD_GROUPS,
  ContactFieldKey,
  DEFAULT_ENABLED_CONTACT_FIELDS,
} from '../contactFields';

export default function ContactFieldSettings() {
  const { t } = useTranslation();
  const [enabled, setEnabled] = useState<Set<ContactFieldKey>>(new Set(DEFAULT_ENABLED_CONTACT_FIELDS));
  const [hideEmpty, setHideEmpty] = useState(false);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    const load = async () => {
      try {
        setLoading(true);
        // /users/me carries both preferences; null enabled fields means "never configured".
        const user = await getCurrentUser();
        const stored = user.enabled_contact_fields ?? null;
        const keys = stored == null ? DEFAULT_ENABLED_CONTACT_FIELDS : (stored as ContactFieldKey[]);
        setEnabled(new Set(keys));
        setHideEmpty(user.hide_empty_contact_fields ?? false);
      } catch (err) {
        setError(err instanceof Error ? err.message : t('settings.contactFields.loadError'));
      } finally {
        setLoading(false);
      }
    };
    load();
  }, [t]);

  const handleToggle = async (key: ContactFieldKey) => {
    const next = new Set(enabled);
    if (next.has(key)) {
      next.delete(key);
    } else {
      next.add(key);
    }
    setEnabled(next);
    try {
      setSaving(true);
      setError('');
      await updateEnabledContactFields(Array.from(next));
    } catch (err) {
      setError(err instanceof Error ? err.message : t('settings.contactFields.saveError'));
    } finally {
      setSaving(false);
    }
  };

  const handleHideEmptyToggle = async () => {
    const next = !hideEmpty;
    setHideEmpty(next);
    try {
      setSaving(true);
      setError('');
      await updateContactFieldDisplay(next);
    } catch (err) {
      setHideEmpty(!next);
      setError(err instanceof Error ? err.message : t('settings.contactFields.saveError'));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Card sx={{ mb: 2 }}>
      <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
        <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
          <ViewListIcon sx={{ mr: 1, color: 'text.secondary', fontSize: 20 }} />
          <Typography variant="subtitle1" sx={{ fontWeight: 500 }}>
            {t('settings.contactFields.title')}
          </Typography>
        </Box>
        <Divider sx={{ mb: 1.5 }} />

        <Stack spacing={1.5}>
          <Typography variant="body2" sx={{
            color: "text.secondary"
          }}>
            {t('settings.contactFields.description')}
          </Typography>

          {error && <Alert severity="error" sx={{ py: 0 }}>{error}</Alert>}

          {loading ? (
            <Typography variant="body2" sx={{
              color: "text.secondary"
            }}>
              {t('settings.contactFields.loading')}
            </Typography>
          ) : (
            <>
              <Box>
                <FormControlLabel
                  control={
                    <Switch
                      size="small"
                      checked={hideEmpty}
                      disabled={saving}
                      onChange={handleHideEmptyToggle}
                    />
                  }
                  label={t('settings.contactFields.hideEmpty')}
                />
                <Typography variant="body2" sx={{
                  color: "text.secondary"
                }}>
                  {t('settings.contactFields.hideEmptyDescription')}
                </Typography>
              </Box>
              {CONTACT_FIELD_GROUPS.map((group) => {
                const fields = CONTACT_FIELDS.filter((f) => f.group === group);
                if (fields.length === 0) return null;
                return (
                  <Box key={group}>
                    <Typography variant="overline" sx={{
                      color: "text.secondary"
                    }}>
                      {t(`settings.contactFields.groups.${group}`)}
                    </Typography>
                    <Stack>
                      {fields.map((f) => (
                        <FormControlLabel
                          key={f.key}
                          control={
                            <Switch
                              size="small"
                              checked={enabled.has(f.key)}
                              disabled={saving}
                              onChange={() => handleToggle(f.key)}
                            />
                          }
                          label={t(f.labelKey)}
                        />
                      ))}
                    </Stack>
                  </Box>
                );
              })}
            </>
          )}
        </Stack>
      </CardContent>
    </Card>
  );
}
