import { useTranslation } from 'react-i18next';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  Box,
  Autocomplete,
  Chip,
  IconButton
} from '@mui/material';
import DeleteIcon from '@mui/icons-material/Delete';

interface EditTimelineItemDialogProps {
  open: boolean;
  onClose: () => void;
  onSave: () => void;
  onDelete: () => void;
  type: 'note' | 'activity';
  values: {
    noteContent?: string;
    noteDate?: string;
    activityTitle?: string;
    activityDescription?: string;
    activityLocation?: string;
    activityDate?: string;
    activityContacts?: { ID: number; firstname: string; lastname: string; nickname?: string }[];
  };
  onChange: (values: any) => void;
  allContacts: { ID: number; firstname: string; lastname: string; nickname?: string }[];
}

export default function EditTimelineItemDialog({
  open,
  onClose,
  onSave,
  onDelete,
  type,
  values,
  onChange,
  allContacts
}: EditTimelineItemDialogProps) {
  const { t } = useTranslation();

  const handleSave = () => {
    onSave();
  };

  const handleDelete = () => {
    if (window.confirm(t('contactDetail.confirmDelete'))) {
      onDelete();
    }
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>
        {type === 'note' ? t('contactDetail.editNote') : t('contactDetail.editActivity')}
      </DialogTitle>
      <DialogContent>
        {type === 'note' ? (
          // Edit Note
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
            <TextField
              fullWidth
              multiline
              rows={6}
              value={values.noteContent || ''}
              onChange={(e) => onChange({ ...values, noteContent: e.target.value })}
              label={t('noteDialog.content')}
              placeholder={t('noteDialog.contentPlaceholder')}
              autoFocus
            />
            <TextField
              fullWidth
              type="date"
              value={values.noteDate || ''}
              onChange={(e) => onChange({ ...values, noteDate: e.target.value })}
              label={t('noteDialog.date')}
              InputLabelProps={{ shrink: true }}
            />
          </Box>
        ) : (
          // Edit Activity
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
            <TextField
              fullWidth
              value={values.activityTitle || ''}
              onChange={(e) => onChange({ ...values, activityTitle: e.target.value })}
              label={t('activityDialog.activityTitle')}
              autoFocus
            />
            <TextField
              fullWidth
              multiline
              rows={3}
              value={values.activityDescription || ''}
              onChange={(e) => onChange({ ...values, activityDescription: e.target.value })}
              label={t('activityDialog.description')}
            />
            <TextField
              fullWidth
              value={values.activityLocation || ''}
              onChange={(e) => onChange({ ...values, activityLocation: e.target.value })}
              label={t('activityDialog.location')}
            />
            <TextField
              fullWidth
              type="date"
              value={values.activityDate || ''}
              onChange={(e) => onChange({ ...values, activityDate: e.target.value })}
              label={t('activityDialog.date')}
              InputLabelProps={{ shrink: true }}
            />
            <Autocomplete
              multiple
              options={allContacts}
              getOptionLabel={(contact) => `${contact.firstname}${contact.nickname ? ` "${contact.nickname}"` : ''} ${contact.lastname}`}
              value={values.activityContacts || []}
              onChange={(_, newValue) => onChange({ ...values, activityContacts: newValue })}
              renderInput={(params) => (
                <TextField
                  {...params}
                  label={t('activityDialog.contacts')}
                  placeholder={t('activityDialog.selectContacts')}
                />
              )}
              renderTags={(value, getTagProps) =>
                value.map((contact, index) => (
                  <Chip
                    label={`${contact.firstname}${contact.nickname ? ` "${contact.nickname}"` : ''} ${contact.lastname}`}
                    {...getTagProps({ index })}
                    size="small"
                    key={contact.ID}
                  />
                ))
              }
            />
          </Box>
        )}
      </DialogContent>
      <DialogActions sx={{ justifyContent: 'space-between', px: 3, pb: 2 }}>
        <IconButton 
          color="error"
          onClick={handleDelete}
          title={t('contactDetail.delete')}
        >
          <DeleteIcon />
        </IconButton>
        <Box>
          <Button onClick={onClose} sx={{ mr: 1 }}>
            {t('contactDetail.cancel')}
          </Button>
          <Button onClick={handleSave} variant="contained">
            {t('contactDetail.save')}
          </Button>
        </Box>
      </DialogActions>
    </Dialog>
  );
}
