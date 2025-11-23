import { useTranslation } from 'react-i18next';
import {
  Box,
  Card,
  CardContent,
  Typography,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Divider
} from '@mui/material';
import LanguageIcon from '@mui/icons-material/Language';

export default function SettingsPage() {
  const { t, i18n } = useTranslation();

  const handleLanguageChange = (event: any) => {
    i18n.changeLanguage(event.target.value);
  };

  return (
    <Box sx={{ maxWidth: 800, mx: 'auto', mt: 4, p: 2 }}>
      <Typography variant="h4" sx={{ mb: 3 }}>
        {t('settings.title')}
      </Typography>

      {/* Language Settings */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
            <LanguageIcon sx={{ mr: 1, color: 'text.secondary' }} />
            <Typography variant="h6" sx={{ fontWeight: 500 }}>
              {t('settings.language.title')}
            </Typography>
          </Box>
          <Divider sx={{ mb: 3 }} />
          
          <FormControl fullWidth>
            <InputLabel id="language-select-label">
              {t('settings.language.label')}
            </InputLabel>
            <Select
              labelId="language-select-label"
              value={i18n.language}
              label={t('settings.language.label')}
              onChange={handleLanguageChange}
            >
              <MenuItem value="en">English</MenuItem>
              <MenuItem value="de">Deutsch</MenuItem>
            </Select>
          </FormControl>
          
          <Typography variant="caption" color="text.secondary" sx={{ mt: 2, display: 'block' }}>
            {t('settings.language.description')}
          </Typography>
        </CardContent>
      </Card>

      {/* Placeholder for future settings */}
      <Card>
        <CardContent>
          <Typography variant="h6" sx={{ fontWeight: 500, mb: 2 }}>
            {t('settings.moreComingSoon')}
          </Typography>
          <Typography variant="body2" color="text.secondary">
            {t('settings.moreComingSoonDescription')}
          </Typography>
        </CardContent>
      </Card>
    </Box>
  );
}
