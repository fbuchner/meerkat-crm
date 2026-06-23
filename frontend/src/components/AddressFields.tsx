import { useTranslation } from 'react-i18next';
import {
  Box,
  Typography,
  Stack,
  TextField,
  Autocomplete,
  IconButton,
  Button,
  Paper,
} from '@mui/material';
import DeleteIcon from '@mui/icons-material/Delete';
import AddIcon from '@mui/icons-material/Add';
import { ContactAddress } from '../api/contacts';
import { CONTACT_TYPE_OPTIONS } from '../contactFields';
import { useRowKeys } from '../hooks/useRowKeys';

interface AddressFieldsProps {
  label: string;
  value: ContactAddress[];
  onChange: (next: ContactAddress[]) => void;
}

const EMPTY_ADDRESS: ContactAddress = {
  type: 'home',
  street: '',
  city: '',
  region: '',
  postal: '',
  country: '',
};

export default function AddressFields({ label, value, onChange }: AddressFieldsProps) {
  const { t } = useTranslation();
  const rowKeys = useRowKeys(value.length);

  const updateAddr = (index: number, patch: Partial<ContactAddress>) => {
    onChange(value.map((a, i) => (i === index ? { ...a, ...patch } : a)));
  };

  const removeAddr = (index: number) => {
    rowKeys.onRemove(index);
    onChange(value.filter((_, i) => i !== index));
  };

  const addAddr = () => {
    rowKeys.onAdd();
    onChange([...value, { ...EMPTY_ADDRESS }]);
  };

  return (
    <Box>
      <Typography variant="subtitle2" gutterBottom>
        {label}
      </Typography>
      <Stack spacing={1.5}>
        {value.map((addr, index) => (
          <Paper key={rowKeys.keyAt(index)} variant="outlined" sx={{ p: 1.5 }}>
            <Stack spacing={1}>
              <Stack direction="row" spacing={1} alignItems="center">
                {/* Free-solo: pick a standard type or type a custom label.
                    Custom labels export as vCard X-ABLabel and round-trip via CardDAV. */}
                <Autocomplete
                  freeSolo
                  options={CONTACT_TYPE_OPTIONS as readonly string[]}
                  value={addr.type}
                  getOptionLabel={(opt) => t(`contacts.types.${opt}`, opt)}
                  onChange={(_, newValue) => updateAddr(index, { type: (newValue ?? '').trim() })}
                  onInputChange={(_, newInput, reason) => {
                    if (reason === 'input') updateAddr(index, { type: newInput });
                  }}
                  sx={{ minWidth: 140 }}
                  renderInput={(params) => (
                    <TextField {...params} label={t('contacts.fieldType')} size="small" />
                  )}
                />
                <Box sx={{ flexGrow: 1 }} />
                <IconButton
                  size="small"
                  color="error"
                  onClick={() => removeAddr(index)}
                  aria-label={t('common.delete')}
                >
                  <DeleteIcon fontSize="small" />
                </IconButton>
              </Stack>
              <TextField
                label={t('contacts.addressFields.street')}
                size="small"
                fullWidth
                value={addr.street}
                onChange={(e) => updateAddr(index, { street: e.target.value })}
              />
              <Stack direction="row" spacing={1}>
                <TextField
                  label={t('contacts.addressFields.city')}
                  size="small"
                  fullWidth
                  value={addr.city}
                  onChange={(e) => updateAddr(index, { city: e.target.value })}
                />
                <TextField
                  label={t('contacts.addressFields.region')}
                  size="small"
                  fullWidth
                  value={addr.region}
                  onChange={(e) => updateAddr(index, { region: e.target.value })}
                />
              </Stack>
              <Stack direction="row" spacing={1}>
                <TextField
                  label={t('contacts.addressFields.postal')}
                  size="small"
                  fullWidth
                  value={addr.postal}
                  onChange={(e) => updateAddr(index, { postal: e.target.value })}
                />
                <TextField
                  label={t('contacts.addressFields.country')}
                  size="small"
                  fullWidth
                  value={addr.country}
                  onChange={(e) => updateAddr(index, { country: e.target.value })}
                />
              </Stack>
            </Stack>
          </Paper>
        ))}
        <Box>
          <Button size="small" startIcon={<AddIcon />} onClick={addAddr} variant="outlined">
            {t('common.add')}
          </Button>
        </Box>
      </Stack>
    </Box>
  );
}
