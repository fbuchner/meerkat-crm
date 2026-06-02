import { useState, ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { Box, Stack, Typography, IconButton, Button } from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';

interface EditableArrayFieldProps<T> {
  icon: ReactNode;
  label: string;
  value: T;
  /** Renders the read-only view of the current value */
  renderDisplay: (value: T) => ReactNode;
  /** Renders the editor; receives the working draft and a setter */
  renderEditor: (draft: T, setDraft: (next: T) => void) => ReactNode;
  /** Deep-clones the value into an editable draft */
  cloneValue: (value: T) => T;
  onSave: (draft: T) => Promise<void>;
}

export default function EditableArrayField<T>({
  icon,
  label,
  value,
  renderDisplay,
  renderEditor,
  cloneValue,
  onSave,
}: EditableArrayFieldProps<T>) {
  const { t } = useTranslation();
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState<T>(value);
  const [saving, setSaving] = useState(false);

  const startEdit = () => {
    setDraft(cloneValue(value));
    setEditing(true);
  };

  const cancel = () => {
    setEditing(false);
  };

  const save = async () => {
    try {
      setSaving(true);
      await onSave(draft);
      setEditing(false);
    } finally {
      setSaving(false);
    }
  };

  if (editing) {
    return (
      <Box>
        <Stack direction="row" spacing={1} alignItems="flex-start">
          <Box sx={{ pt: 0.5 }}>{icon}</Box>
          <Box sx={{ flexGrow: 1 }}>
            {renderEditor(draft, setDraft)}
            <Stack direction="row" spacing={1} sx={{ mt: 1 }}>
              <Button size="small" variant="contained" onClick={save} disabled={saving}>
                {saving ? t('common.saving') : t('common.save')}
              </Button>
              <Button size="small" onClick={cancel} disabled={saving}>
                {t('common.cancel')}
              </Button>
            </Stack>
          </Box>
        </Stack>
      </Box>
    );
  }

  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'flex-start',
        '&:hover .edit-button': { opacity: 1 },
      }}
    >
      {icon}
      <Box sx={{ flexGrow: 1, minWidth: 0 }}>
        <Typography variant="caption" color="text.secondary">
          {label}
        </Typography>
        <Box sx={{ wordBreak: 'break-word' }}>{renderDisplay(value)}</Box>
      </Box>
      <IconButton
        className="edit-button"
        size="small"
        onClick={startEdit}
        aria-label={t('common.edit')}
        sx={{ opacity: 0, transition: 'opacity 0.2s' }}
      >
        <EditIcon fontSize="small" />
      </IconButton>
    </Box>
  );
}
