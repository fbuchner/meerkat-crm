import { Box, Card, CardContent, Avatar, Typography, Chip, IconButton, Stack, TextField, Autocomplete, Button } from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';
import SaveIcon from '@mui/icons-material/Save';
import CloseIcon from '@mui/icons-material/Close';
import DeleteIcon from '@mui/icons-material/Delete';
import AddIcon from '@mui/icons-material/Add';
import CameraAltIcon from '@mui/icons-material/CameraAlt';
import AutoModeIcon from '@mui/icons-material/AutoMode';
import ArchiveIcon from '@mui/icons-material/Archive';
import UnarchiveIcon from '@mui/icons-material/Unarchive';
import { useTranslation } from 'react-i18next';
import { ContactFieldKey, resolveEnabledFields } from '../contactFields';

export interface ProfileValues {
  prefix: string;
  firstname: string;
  middle_name: string;
  lastname: string;
  suffix: string;
  nickname: string;
  gender: string;
}

interface ContactHeaderProps {
  contact: {
    ID: number;
    prefix?: string;
    firstname: string;
    middle_name?: string;
    lastname: string;
    suffix?: string;
    nickname?: string;
    gender?: string;
    circles?: string[];
    archived?: boolean;
  };
  profilePic: string;
  editingProfile: boolean;
  profileValues: ProfileValues;
  enabledFields?: Set<ContactFieldKey>;
  editingCircles: boolean;
  newCircleName: string;
  availableCircles: string[];
  onStartEditProfile: () => void;
  onCancelEditProfile: () => void;
  onSaveProfile: () => void;
  onDeleteContact: () => void;
  onProfileValueChange: (values: ProfileValues) => void;
  onToggleEditCircles: () => void;
  onAddCircle: (circleName?: string) => void;
  onDeleteCircle: (circle: string) => void;
  onNewCircleNameChange: (name: string) => void;
  onUploadProfilePicture: () => void;
  onStayInTouch?: () => void;
  onArchiveContact?: () => void;
  onUnarchiveContact?: () => void;
}

export default function ContactHeader({
  contact,
  profilePic,
  editingProfile,
  profileValues,
  enabledFields,
  editingCircles,
  newCircleName,
  availableCircles,
  onStartEditProfile,
  onCancelEditProfile,
  onSaveProfile,
  onDeleteContact,
  onProfileValueChange,
  onToggleEditCircles,
  onAddCircle,
  onDeleteCircle,
  onNewCircleNameChange,
  onUploadProfilePicture,
  onStayInTouch,
  onArchiveContact,
  onUnarchiveContact
}: ContactHeaderProps) {
  const { t } = useTranslation();
  const enabled = enabledFields ?? resolveEnabledFields(null);
  const isOn = (key: ContactFieldKey) => enabled.has(key);

  const displayName = [
    contact.prefix,
    contact.firstname,
    contact.nickname ? `"${contact.nickname}"` : '',
    contact.middle_name,
    contact.lastname,
    contact.suffix,
  ]
    .map((part) => part?.trim())
    .filter(Boolean)
    .join(' ');

  return (
    <Card sx={{ mb: 1.5 }}>
      <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
        <Box sx={{ display: 'flex', alignItems: 'flex-start' }}>
          <Box
            sx={{
              position: 'relative',
              '&:hover .camera-badge': {
                opacity: 1
              }
            }}
          >
            <Avatar
              src={profilePic || undefined}
              sx={{
                width: 90,
                height: 90,
                cursor: 'pointer',
                bgcolor: 'primary.main',
                fontSize: '2rem',
                '&:hover': { opacity: 0.8 }
              }}
              onClick={onUploadProfilePicture}
            >
              {contact.firstname.charAt(0)}
            </Avatar>
            <IconButton
              className="camera-badge"
              size="small"
              onClick={onUploadProfilePicture}
              sx={{
                position: 'absolute',
                bottom: -4,
                right: -4,
                bgcolor: 'primary.main',
                color: 'white',
                width: 22,
                height: 22,
                opacity: 0,
                transition: 'opacity 0.2s',
                '&:hover': { bgcolor: 'primary.dark' }
              }}
            >
              <CameraAltIcon sx={{ fontSize: 14 }} />
            </IconButton>
          </Box>
          <Box sx={{ flex: 1, ml: 1.5 }}>
            {editingProfile ? (
              // Edit Mode
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                {isOn('prefix') && (
                  <TextField
                    label={t('contacts.prefix')}
                    value={profileValues.prefix}
                    onChange={(e) => onProfileValueChange({ ...profileValues, prefix: e.target.value })}
                    size="small"
                  />
                )}
                <TextField
                  label={t('contactDetail.firstname')}
                  value={profileValues.firstname}
                  onChange={(e) => onProfileValueChange({ ...profileValues, firstname: e.target.value })}
                  size="small"
                  required
                  autoFocus
                />
                {isOn('middle_name') && (
                  <TextField
                    label={t('contacts.middleName')}
                    value={profileValues.middle_name}
                    onChange={(e) => onProfileValueChange({ ...profileValues, middle_name: e.target.value })}
                    size="small"
                  />
                )}
                <TextField
                  label={t('contactDetail.lastname')}
                  value={profileValues.lastname}
                  onChange={(e) => onProfileValueChange({ ...profileValues, lastname: e.target.value })}
                  size="small"
                />
                {isOn('suffix') && (
                  <TextField
                    label={t('contacts.suffix')}
                    value={profileValues.suffix}
                    onChange={(e) => onProfileValueChange({ ...profileValues, suffix: e.target.value })}
                    size="small"
                  />
                )}
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
                {contact.archived && (
                  <Chip
                    label={t('contactDetail.archivedBadge')}
                    color="warning"
                    size="small"
                    sx={{ mb: 1 }}
                  />
                )}
                <Box
                  sx={{
                    display: 'flex',
                    alignItems: 'flex-start',
                    justifyContent: 'space-between',
                    flexWrap: 'wrap',
                    gap: 0.5,
                    '&:hover .edit-icon': {
                      opacity: 1
                    }
                  }}
                >
                  <Box sx={{ display: 'flex', alignItems: 'center' }}>
                    <Typography variant="h5" sx={{ fontWeight: 500, lineHeight: 1.2 }}>
                      {displayName}
                    </Typography>
                    <IconButton
                      className="edit-icon"
                      size="small"
                      onClick={onStartEditProfile}
                      sx={{
                        ml: 1,
                        opacity: 0,
                        transition: 'opacity 0.2s'
                      }}
                    >
                      <EditIcon fontSize="small" />
                    </IconButton>
                  </Box>
                  <Box sx={{ display: 'flex', gap: 1 }}>
                    {contact.archived ? (
                      onUnarchiveContact && (
                        <Button
                          variant="outlined"
                          size="small"
                          color="success"
                          startIcon={<UnarchiveIcon />}
                          onClick={onUnarchiveContact}
                        >
                          {t('contactDetail.unarchive')}
                        </Button>
                      )
                    ) : (
                      <>
                        {onStayInTouch && (
                          <Button
                            variant="outlined"
                            size="small"
                            startIcon={<AutoModeIcon />}
                            onClick={onStayInTouch}
                          >
                            {t('contactDetail.stayInTouch')}
                          </Button>
                        )}
                        {onArchiveContact && (
                          <Button
                            variant="outlined"
                            size="small"
                            color="warning"
                            startIcon={<ArchiveIcon />}
                            onClick={onArchiveContact}
                          >
                            {t('contactDetail.archive')}
                          </Button>
                        )}
                      </>
                    )}
                  </Box>
                </Box>
                {contact.gender && (
                  <Typography variant="body2" color="text.secondary" sx={{ mt: 0.25 }}>
                    {t(`contactDetail.${contact.gender}`)}
                  </Typography>
                )}
              </>
            )}

            {/* Circles Section */}
            <Box
              sx={{
                mt: 1,
                '&:hover .edit-icon': {
                  opacity: 1
                }
              }}
            >
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 0.5 }}>
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
                  <Stack direction="row" spacing={1} sx={{ mt: 1 }}>
                    <Autocomplete
                      key={contact.circles?.length ?? 0}
                      size="small"
                      options={availableCircles.filter(c => !contact.circles?.includes(c))}
                      value={null}
                      onChange={(_, value) => {
                        if (value) {
                          onAddCircle(value);
                        }
                      }}
                      blurOnSelect
                      sx={{ minWidth: 200 }}
                      renderInput={(params) => (
                        <TextField
                          {...params}
                          label={t('contacts.selectCircle')}
                          size="small"
                        />
                      )}
                    />
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
                      onClick={() => onAddCircle()}
                      disabled={!newCircleName.trim()}
                    >
                      <AddIcon />
                    </IconButton>
                  </Stack>
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
