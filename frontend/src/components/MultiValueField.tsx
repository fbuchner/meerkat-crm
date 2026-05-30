import { useTranslation } from 'react-i18next';
import { Box, Typography, Stack, TextField, MenuItem, IconButton, Button } from '@mui/material';
import DeleteIcon from '@mui/icons-material/Delete';
import AddIcon from '@mui/icons-material/Add';
import { ContactValue } from '../api/contacts';
import { CONTACT_TYPE_OPTIONS } from '../contactFields';

interface MultiValueFieldProps {
  label: string;
  value: ContactValue[];
  onChange: (next: ContactValue[]) => void;
  /** HTML input type for the value field */
  valueType?: 'text' | 'email' | 'tel' | 'url';
  /** Default type token for newly added rows */
  defaultType?: string;
  /** Available type tokens; defaults to the standard vCard set */
  typeOptions?: readonly string[];
  /** When true, the type column is a free-text field (used for IMPP service names) */
  freeTextType?: boolean;
}

export default function MultiValueField({
  label,
  value,
  onChange,
  valueType = 'text',
  defaultType = 'home',
  typeOptions = CONTACT_TYPE_OPTIONS,
  freeTextType = false,
}: MultiValueFieldProps) {
  const { t } = useTranslation();

  const updateRow = (index: number, patch: Partial<ContactValue>) => {
    onChange(value.map((row, i) => (i === index ? { ...row, ...patch } : row)));
  };

  const removeRow = (index: number) => {
    onChange(value.filter((_, i) => i !== index));
  };

  const addRow = () => {
    onChange([...value, { type: defaultType, value: '' }]);
  };

  return (
    <Box>
      <Typography variant="subtitle2" gutterBottom>
        {label}
      </Typography>
      <Stack spacing={1}>
        {value.map((row, index) => (
          <Stack key={index} direction="row" spacing={1} alignItems="center">
            {freeTextType ? (
              <TextField
                label={t('contacts.fieldType')}
                size="small"
                value={row.type}
                onChange={(e) => updateRow(index, { type: e.target.value })}
                sx={{ minWidth: 120 }}
              />
            ) : (
              <TextField
                select
                label={t('contacts.fieldType')}
                size="small"
                value={typeOptions.includes(row.type as never) ? row.type : 'other'}
                onChange={(e) => updateRow(index, { type: e.target.value })}
                sx={{ minWidth: 120 }}
              >
                {typeOptions.map((opt) => (
                  <MenuItem key={opt} value={opt}>
                    {t(`contacts.types.${opt}`, opt)}
                  </MenuItem>
                ))}
              </TextField>
            )}
            <TextField
              label={label}
              size="small"
              type={valueType}
              fullWidth
              value={row.value}
              onChange={(e) => updateRow(index, { value: e.target.value })}
            />
            <IconButton
              size="small"
              color="error"
              onClick={() => removeRow(index)}
              aria-label={t('common.delete')}
            >
              <DeleteIcon fontSize="small" />
            </IconButton>
          </Stack>
        ))}
        <Box>
          <Button size="small" startIcon={<AddIcon />} onClick={addRow} variant="outlined">
            {t('common.add')}
          </Button>
        </Box>
      </Stack>
    </Box>
  );
}
