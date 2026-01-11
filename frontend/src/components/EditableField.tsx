import { Box, Typography, TextField, IconButton } from '@mui/material';
import SaveIcon from '@mui/icons-material/Save';
import CloseIcon from '@mui/icons-material/Close';
import EditIcon from '@mui/icons-material/Edit';

interface EditableFieldProps {
  icon: React.ReactNode;
  label: string;
  field: string;
  value: string;
  multiline?: boolean;
  placeholder?: string;
  displaySuffix?: string;
  isEditing: boolean;
  editValue: string;
  validationError: string;
  onEditStart: (field: string, value: string) => void;
  onEditCancel: () => void;
  onEditSave: (field: string) => void;
  onEditValueChange: (value: string) => void;
}

export default function EditableField({
  icon,
  label,
  field,
  value,
  multiline = false,
  placeholder = '',
  displaySuffix,
  isEditing,
  editValue,
  validationError,
  onEditStart,
  onEditCancel,
  onEditSave,
  onEditValueChange
}: EditableFieldProps) {
  const displayValue = value ? (displaySuffix ? `${value} ${displaySuffix}` : value) : '-';
  const showError = isEditing && validationError;

  return (
    <Box
      sx={{
        position: 'relative',
        '&:hover .edit-icon': {
          opacity: 1
        }
      }}
    >
      <Box sx={{ display: 'flex', alignItems: multiline ? 'flex-start' : 'center' }}>
        {icon}
        <Box sx={{ flex: 1 }}>
          <Typography variant="caption" color="text.secondary">
            {label}
          </Typography>
          {isEditing ? (
            <Box>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 0.5 }}>
                <TextField
                  value={editValue}
                  onChange={(e) => onEditValueChange(e.target.value)}
                  size="small"
                  fullWidth
                  multiline={multiline}
                  rows={multiline ? 3 : 1}
                  autoFocus
                  error={!!showError}
                  placeholder={placeholder}
                />
                <IconButton size="small" color="primary" onClick={() => onEditSave(field)}>
                  <SaveIcon fontSize="small" />
                </IconButton>
                <IconButton size="small" onClick={onEditCancel}>
                  <CloseIcon fontSize="small" />
                </IconButton>
              </Box>
              {showError && (
                <Typography variant="caption" color="error" sx={{ mt: 0.5, display: 'block' }}>
                  {validationError}
                </Typography>
              )}
            </Box>
          ) : (
            <Typography variant="body1" sx={{ wordBreak: 'break-word' }}>
              {displayValue}
            </Typography>
          )}
        </Box>
        {!isEditing && (
          <IconButton
            className="edit-icon"
            size="small"
            onClick={() => onEditStart(field, value)}
            sx={{
              opacity: 0,
              transition: 'opacity 0.2s',
              ml: 1
            }}
          >
            <EditIcon fontSize="small" />
          </IconButton>
        )}
      </Box>
    </Box>
  );
}
