import { Card, CardContent, Typography, Divider, Stack } from '@mui/material';
import EmailIcon from '@mui/icons-material/Email';
import PhoneIcon from '@mui/icons-material/Phone';
import CakeIcon from '@mui/icons-material/Cake';
import HomeIcon from '@mui/icons-material/Home';
import WorkIcon from '@mui/icons-material/Work';
import RestaurantIcon from '@mui/icons-material/Restaurant';
import PeopleIcon from '@mui/icons-material/People';
import { useTranslation } from 'react-i18next';
import EditableField from './EditableField';

interface ContactInformationProps {
  contact: {
    email?: string;
    phone?: string;
    birthday?: string;
    address?: string;
    work_information?: string;
    food_preference?: string;
    how_we_met?: string;
    contact_information?: string;
  };
  editingField: string | null;
  editValue: string;
  validationError: string;
  onEditStart: (field: string, value: string) => void;
  onEditCancel: () => void;
  onEditSave: (field: string) => void;
  onEditValueChange: (value: string) => void;
}

export default function ContactInformation({
  contact,
  editingField,
  editValue,
  validationError,
  onEditStart,
  onEditCancel,
  onEditSave,
  onEditValueChange
}: ContactInformationProps) {
  const { t } = useTranslation();

  return (
    <Card sx={{ flex: 1 }}>
      <CardContent sx={{ py: 2 }}>
        <Typography variant="h6" sx={{ mb: 1.5, fontWeight: 500 }}>
          {t('contactDetail.generalInfo')}
        </Typography>
        <Divider sx={{ mb: 1.5 }} />

        <Stack spacing={2}>
          <EditableField
            icon={<EmailIcon sx={{ mr: 1, color: 'text.secondary', fontSize: '1.2rem' }} />}
            label={t('contactDetail.email')}
            field="email"
            value={contact.email || ''}
            isEditing={editingField === 'email'}
            editValue={editValue}
            validationError={validationError}
            onEditStart={onEditStart}
            onEditCancel={onEditCancel}
            onEditSave={onEditSave}
            onEditValueChange={onEditValueChange}
          />

          <EditableField
            icon={<PhoneIcon sx={{ mr: 1, color: 'text.secondary', fontSize: '1.2rem' }} />}
            label={t('contactDetail.phone')}
            field="phone"
            value={contact.phone || ''}
            isEditing={editingField === 'phone'}
            editValue={editValue}
            validationError={validationError}
            onEditStart={onEditStart}
            onEditCancel={onEditCancel}
            onEditSave={onEditSave}
            onEditValueChange={onEditValueChange}
          />

          <EditableField
            icon={<CakeIcon sx={{ mr: 1, color: 'text.secondary', fontSize: '1.2rem' }} />}
            label={t('contactDetail.birthday')}
            field="birthday"
            value={contact.birthday || ''}
            placeholder="DD.MM. or DD.MM.YYYY"
            isEditing={editingField === 'birthday'}
            editValue={editValue}
            validationError={validationError}
            onEditStart={onEditStart}
            onEditCancel={onEditCancel}
            onEditSave={onEditSave}
            onEditValueChange={onEditValueChange}
          />

          <EditableField
            icon={<HomeIcon sx={{ mr: 1, color: 'text.secondary', fontSize: '1.2rem' }} />}
            label={t('contactDetail.address')}
            field="address"
            value={contact.address || ''}
            multiline
            isEditing={editingField === 'address'}
            editValue={editValue}
            validationError={validationError}
            onEditStart={onEditStart}
            onEditCancel={onEditCancel}
            onEditSave={onEditSave}
            onEditValueChange={onEditValueChange}
          />

          <EditableField
            icon={<WorkIcon sx={{ mr: 1, mt: 0.5, color: 'text.secondary', fontSize: '1.2rem' }} />}
            label={t('contactDetail.workInfo')}
            field="work_information"
            value={contact.work_information || ''}
            multiline
            isEditing={editingField === 'work_information'}
            editValue={editValue}
            validationError={validationError}
            onEditStart={onEditStart}
            onEditCancel={onEditCancel}
            onEditSave={onEditSave}
            onEditValueChange={onEditValueChange}
          />

          <EditableField
            icon={<RestaurantIcon sx={{ mr: 1, mt: 0.5, color: 'text.secondary', fontSize: '1.2rem' }} />}
            label={t('contactDetail.foodPreferences')}
            field="food_preference"
            value={contact.food_preference || ''}
            multiline
            isEditing={editingField === 'food_preference'}
            editValue={editValue}
            validationError={validationError}
            onEditStart={onEditStart}
            onEditCancel={onEditCancel}
            onEditSave={onEditSave}
            onEditValueChange={onEditValueChange}
          />

          <EditableField
            icon={<PeopleIcon sx={{ mr: 1, mt: 0.5, color: 'text.secondary', fontSize: '1.2rem' }} />}
            label={t('contactDetail.howWeMet')}
            field="how_we_met"
            value={contact.how_we_met || ''}
            multiline
            isEditing={editingField === 'how_we_met'}
            editValue={editValue}
            validationError={validationError}
            onEditStart={onEditStart}
            onEditCancel={onEditCancel}
            onEditSave={onEditSave}
            onEditValueChange={onEditValueChange}
          />

          <EditableField
            icon={null}
            label={t('contactDetail.additionalInfo')}
            field="contact_information"
            value={contact.contact_information || ''}
            multiline
            isEditing={editingField === 'contact_information'}
            editValue={editValue}
            validationError={validationError}
            onEditStart={onEditStart}
            onEditCancel={onEditCancel}
            onEditSave={onEditSave}
            onEditValueChange={onEditValueChange}
          />
        </Stack>
      </CardContent>
    </Card>
  );
}
