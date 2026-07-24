import { useTranslation } from 'react-i18next';
import { Box, Typography, Stack, TextField, Autocomplete, IconButton, Button, MenuItem } from '@mui/material';
import DeleteIcon from '@mui/icons-material/Delete';
import AddIcon from '@mui/icons-material/Add';
import { ContactValue } from '../api/contacts';
import { CONTACT_TYPE_OPTIONS } from '../contactFields';
import { useRowKeys } from '../hooks/useRowKeys';

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
  /** i18n key for the type/language column label; defaults to 'contacts.fieldType' */
  typeLabelKey?: string;
  /** When set, the type column is labelled/translated as a language tag (e.g. Pronouns/GramGender) instead of a vCard TYPE */
  typeIsLanguage?: boolean;
  /** When set, the value column becomes a constrained Select of these options instead of free text (e.g. GramGender's enum) */
  valueOptions?: readonly string[];
  /** i18n key prefix for translating valueOptions entries, e.g. 'contacts.gramGender' -> 'contacts.gramGenderFeminine' */
  valueOptionLabel?: (option: string) => string;
}

export default function MultiValueField({
  label,
  value,
  onChange,
  valueType = 'text',
  defaultType = 'home',
  typeOptions = CONTACT_TYPE_OPTIONS,
  freeTextType = false,
  typeLabelKey = 'contacts.fieldType',
  typeIsLanguage = false,
  valueOptions,
  valueOptionLabel,
}: MultiValueFieldProps) {
  const { t } = useTranslation();
  const rowKeys = useRowKeys(value.length);

  const updateRow = (index: number, patch: Partial<ContactValue>) => {
    onChange(value.map((row, i) => (i === index ? { ...row, ...patch } : row)));
  };

  const removeRow = (index: number) => {
    rowKeys.onRemove(index);
    onChange(value.filter((_, i) => i !== index));
  };

  const addRow = () => {
    rowKeys.onAdd();
    onChange([...value, { type: defaultType, value: '' }]);
  };

  return (
    <Box>
      <Typography variant="subtitle2" gutterBottom>
        {label}
      </Typography>
      <Stack spacing={1}>
        {value.map((row, index) => (
          <Stack key={rowKeys.keyAt(index)} direction="row" spacing={1} alignItems="center">
            {freeTextType ? (
              <TextField
                label={t(typeLabelKey)}
                size="small"
                value={row.type}
                onChange={(e) => updateRow(index, { type: e.target.value })}
                sx={{ minWidth: 120 }}
              />
            ) : (
              // Free-solo: pick a standard option or type a custom one. Selecting a
              // standard option stores its token (e.g. "home") for proper i18n;
              // typing a custom value stores the text verbatim. For the vCard TYPE
              // case, custom labels are exported as X-ABLabel and round-trip via CardDAV.
              <Autocomplete
                freeSolo
                options={typeOptions as readonly string[]}
                value={row.type}
                getOptionLabel={(opt) => (typeIsLanguage ? opt : t(`contacts.types.${opt}`, opt))}
                onChange={(_, newValue) => updateRow(index, { type: (newValue ?? '').trim() })}
                onInputChange={(_, newInput, reason) => {
                  if (reason === 'input') updateRow(index, { type: newInput });
                }}
                sx={{ minWidth: 140 }}
                renderInput={(params) => (
                  <TextField {...params} label={t(typeLabelKey)} size="small" />
                )}
              />
            )}
            {valueOptions ? (
              <TextField
                select
                label={label}
                size="small"
                fullWidth
                value={row.value}
                onChange={(e) => updateRow(index, { value: e.target.value })}
              >
                <MenuItem value="">&nbsp;</MenuItem>
                {valueOptions.map((opt) => (
                  <MenuItem key={opt} value={opt}>
                    {valueOptionLabel ? valueOptionLabel(opt) : opt}
                  </MenuItem>
                ))}
              </TextField>
            ) : (
              <TextField
                label={label}
                size="small"
                type={valueType}
                fullWidth
                value={row.value}
                onChange={(e) => updateRow(index, { value: e.target.value })}
              />
            )}
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
