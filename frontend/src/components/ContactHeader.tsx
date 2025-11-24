import { Box, Card, CardContent, Avatar, Typography, Chip, IconButton, Stack, TextField, Badge } from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';
import SaveIcon from '@mui/icons-material/Save';
import CloseIcon from '@mui/icons-material/Close';
import DeleteIcon from '@mui/icons-material/Delete';
import AddIcon from '@mui/icons-material/Add';
import CameraAltIcon from '@mui/icons-material/CameraAlt';
import { useTranslation } from 'react-i18next';

interface ContactHeaderProps {
  contact: {
    ID: number;
    firstname: string;
    lastname: string;
    nickname?: string;
    gender?: string;
    circles?: string[];
  };
  profilePic: string;
  editingProfile: boolean;
  profileValues: {
    firstname: string;
    lastname: string;
    nickname: string;
    gender: string;
  };
  editingCircles: boolean;
  newCircleName: string;
  onStartEditProfile: () => void;
  onCancelEditProfile: () => void;
  onSaveProfile: () => void;
  onDeleteContact: () => void;
  onProfileValueChange: (values: any) => void;
  onToggleEditCircles: () => void;
  onAddCircle: () => void;
  onDeleteCircle: (circle: string) => void;
  onNewCircleNameChange: (name: string) => void;
  onUploadProfilePicture: () => void;
}

export default function ContactHeader({
  contact,
  profilePic,
  editingProfile,
  profileValues,
  editingCircles,
  newCircleName,
  onStartEditProfile,
  onCancelEditProfile,
  onSaveProfile,
  onDeleteContact,
  onProfileValueChange,
  onToggleEditCircles,
  onAddCircle,
  onDeleteCircle,
  onNewCircleNameChange,
  onUploadProfilePicture
}: ContactHeaderProps) {
  const { t } = useTranslation();

  return (
    <Card sx={{ mb: 2 }}>
      <CardContent sx={{ py: 2 }}>
        <Box sx={{ display: 'flex', alignItems: 'flex-start', mb: 1.5 }}>
          <Badge
            overlap="circular"
            anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
            badgeContent={
              <IconButton
                size="small"
                onClick={onUploadProfilePicture}
                sx={{
                  bgcolor: 'primary.main',
                  color: 'white',
                  width: 28,
                  height: 28,
                  '&:hover': { bgcolor: 'primary.dark' }
                }}
              >
                <CameraAltIcon sx={{ fontSize: 16 }} />
              </IconButton>
            }
          >
            <Avatar
              src={profilePic || undefined}
              sx={{ 
                width: 80, 
                height: 80,
                cursor: 'pointer',
                '&:hover': { opacity: 0.8 }
              }}
              onClick={onUploadProfilePicture}
            />
          </Badge>
          <Box sx={{ flex: 1, ml: 2 }}>
            {editingProfile ? (
              // Edit Mode
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                <TextField
                  label={t('contactDetail.firstname')}
                  value={profileValues.firstname}
                  onChange={(e) => onProfileValueChange({ ...profileValues, firstname: e.target.value })}
                  size="small"
                  required
                  autoFocus
                />
                <TextField
                  label={t('contactDetail.lastname')}
                  value={profileValues.lastname}
                  onChange={(e) => onProfileValueChange({ ...profileValues, lastname: e.target.value })}
                  size="small"
                  required
                />
                <TextField
                  label={t('contactDetail.nickname')}
                  value={profileValues.nickname}
                  onChange={(e) => onProfileValueChange({ ...profileValues, nickname: e.target.value })}
                  size="small"
                />
                <TextField
                  select
                  label={t('contactDetail.gender')}
                  value={profileValues.gender}
                  onChange={(e) => onProfileValueChange({ ...profileValues, gender: e.target.value })}
                  size="small"
                  SelectProps={{ native: true }}
                >
                  <option value=""></option>
                  <option value="male">{t('contactDetail.male')}</option>
                  <option value="female">{t('contactDetail.female')}</option>
                  <option value="other">{t('contactDetail.other')}</option>
                </TextField>
                <Box sx={{ display: 'flex', gap: 1, justifyContent: 'space-between' }}>
                  <IconButton
                    size="small"
                    color="error"
                    onClick={onDeleteContact}
                    title={t('contactDetail.deleteContact')}
                  >
                    <DeleteIcon />
                  </IconButton>
                  <Box sx={{ display: 'flex', gap: 1 }}>
                    <IconButton size="small" color="primary" onClick={onSaveProfile}>
                      <SaveIcon />
                    </IconButton>
                    <IconButton size="small" onClick={onCancelEditProfile}>
                      <CloseIcon />
                    </IconButton>
                  </Box>
                </Box>
              </Box>
            ) : (
              // View Mode
              <>
                <Box
                  sx={{
                    display: 'flex',
                    alignItems: 'center',
                    '&:hover .edit-icon': {
                      opacity: 1
                    }
                  }}
                >
                  <Typography variant="h4" sx={{ fontWeight: 500 }}>
                    {contact.firstname} {contact.nickname && `"${contact.nickname}"`} {contact.lastname}
                  </Typography>
                  <IconButton
                    className="edit-icon"
                    size="small"
                    onClick={onStartEditProfile}
                    sx={{
                      ml: 2,
                      opacity: 0,
                      transition: 'opacity 0.2s'
                    }}
                  >
                    <EditIcon />
                  </IconButton>
                </Box>
                {contact.gender && (
                  <Typography variant="body1" color="text.secondary" sx={{ mt: 1 }}>
                    {contact.gender}
                  </Typography>
                )}
              </>
            )}

            {/* Circles Section */}
            <Box
              sx={{
                mt: 2,
                '&:hover .edit-icon': {
                  opacity: 1
                }
              }}
            >
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                <Typography variant="caption" color="text.secondary">
                  {t('contactDetail.circles')}
                </Typography>
                <IconButton
                  className="edit-icon"
                  size="small"
                  onClick={onToggleEditCircles}
                  sx={{
                    ml: 'auto',
                    opacity: 0,
                    transition: 'opacity 0.2s'
                  }}
                >
                  <EditIcon fontSize="small" />
                </IconButton>
              </Box>

              {editingCircles ? (
                // Edit Mode
                <Box>
                  <Stack direction="row" spacing={1} flexWrap="wrap" sx={{ gap: 1, mb: 1 }}>
                    {contact.circles && contact.circles.length > 0 ? (
                      contact.circles.map((circle, index) => (
                        <Chip
                          key={index}
                          label={circle}
                          size="small"
                          color="primary"
                          onDelete={() => onDeleteCircle(circle)}
                        />
                      ))
                    ) : (
                      <Typography variant="caption" color="text.secondary">
                        {t('contactDetail.noCircles')}
                      </Typography>
                    )}
                  </Stack>
                  <Box sx={{ display: 'flex', gap: 1, mt: 1 }}>
                    <TextField
                      size="small"
                      placeholder={t('contactDetail.newCircle')}
                      value={newCircleName}
                      onChange={(e) => onNewCircleNameChange(e.target.value)}
                      onKeyPress={(e) => {
                        if (e.key === 'Enter') {
                          onAddCircle();
                        }
                      }}
                      sx={{ flexGrow: 1 }}
                    />
                    <IconButton
                      size="small"
                      color="primary"
                      onClick={onAddCircle}
                      disabled={!newCircleName.trim()}
                    >
                      <AddIcon />
                    </IconButton>
                  </Box>
                </Box>
              ) : (
                // View Mode
                <Stack direction="row" spacing={1} flexWrap="wrap" sx={{ gap: 1 }}>
                  {contact.circles && contact.circles.length > 0 ? (
                    contact.circles.map((circle, index) => (
                      <Chip key={index} label={circle} size="small" color="primary" />
                    ))
                  ) : (
                    <Typography variant="caption" color="text.secondary">
                      {t('contactDetail.noCircles')}
                    </Typography>
                  )}
                </Stack>
              )}
            </Box>
          </Box>
        </Box>
      </CardContent>
    </Card>
  );
}
