import { useState, useMemo } from 'react';
import { Card, CardContent, Divider, Stack, Box, Tabs, Tab, Button } from '@mui/material';
import EmailIcon from '@mui/icons-material/Email';
import PhoneIcon from '@mui/icons-material/Phone';
import CakeIcon from '@mui/icons-material/Cake';
import HomeIcon from '@mui/icons-material/Home';
import WorkIcon from '@mui/icons-material/Work';
import RestaurantIcon from '@mui/icons-material/Restaurant';
import PeopleIcon from '@mui/icons-material/People';
import AddIcon from '@mui/icons-material/Add';
import TuneIcon from '@mui/icons-material/Tune';
import { useTranslation } from 'react-i18next';
import EditableField from './EditableField';
import RelationshipList from './RelationshipList';
import { Relationship, IncomingRelationship } from '../api/relationships';
import { useDateFormat } from '../DateFormatProvider';


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
    custom_fields?: Record<string, string>;
  };
  editingField: string | null;
  editValue: string;
  validationError: string;
  onEditStart: (field: string, value: string) => void;
  onEditCancel: () => void;
  onEditSave: (field: string) => void;
  onEditValueChange: (value: string) => void;
  // Relationship props
  relationships?: Relationship[];
  incomingRelationships?: IncomingRelationship[];
  onAddRelationship?: () => void;
  onEditRelationship?: (relationship: Relationship) => void;
  onDeleteRelationship?: (relationshipId: number) => void;
  // Custom fields
  customFieldNames?: string[];
}

export default function ContactInformation({
  contact,
  editingField,
  editValue,
  validationError,
  onEditStart,
  onEditCancel,
  onEditSave,
  onEditValueChange,
  relationships = [],
  incomingRelationships = [],
  onAddRelationship,
  onEditRelationship,
  onDeleteRelationship,
  customFieldNames = [],
}: ContactInformationProps) {
  const { t } = useTranslation();
  const { formatBirthday, getBirthdayPlaceholder, calculateAge } = useDateFormat();
  const [activeTab, setActiveTab] = useState(0);

  const birthdayAgeSuffix = useMemo(() => {
    if (!contact.birthday) return undefined;
    const age = calculateAge(contact.birthday);
    if (age === null) return undefined;
    return t('dashboard.yearsOld', { age });
  }, [contact.birthday, t, calculateAge]);

  return (
    <Card sx={{ flex: 1 }}>
      <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
        <Tabs value={activeTab} onChange={(_, newValue) => setActiveTab(newValue)} aria-label="contact information tabs">
          <Tab label={t('contactDetail.generalInfo')} />
          <Tab label={t('relationships.title')} />
        </Tabs>
      </Box>

      {/* General Information Tab */}
      {activeTab === 0 && (
        <CardContent sx={{ py: 2 }}>
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
            formattedDisplayValue={contact.birthday ? formatBirthday(contact.birthday) : undefined}
            placeholder={getBirthdayPlaceholder()}
            displaySuffix={birthdayAgeSuffix}
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

          {/* Custom Fields */}
          {customFieldNames.map((fieldName) => (
            <EditableField
              key={`custom_field_${fieldName}`}
              icon={<TuneIcon sx={{ mr: 1, color: 'text.secondary', fontSize: '1.2rem' }} />}
              label={fieldName}
              field={`custom_field_${fieldName}`}
              value={contact.custom_fields?.[fieldName] || ''}
              multiline
              isEditing={editingField === `custom_field_${fieldName}`}
              editValue={editValue}
              validationError={validationError}
              onEditStart={onEditStart}
              onEditCancel={onEditCancel}
              onEditSave={onEditSave}
              onEditValueChange={onEditValueChange}
            />
          ))}
        </Stack>
      </CardContent>
      )}

      {/* Relationships Tab */}
      {activeTab === 1 && (
        <CardContent sx={{ py: 2 }}>
          <Box sx={{ display: 'flex', justifyContent: 'flex-end', mb: 1.5 }}>
            <Button
              startIcon={<AddIcon />}
              onClick={onAddRelationship}
              variant="outlined"
              size="small"
            >
              {t('relationships.addRelationship')}
            </Button>
          </Box>
          <Divider sx={{ mb: 1.5 }} />
          <RelationshipList
            relationships={relationships}
            incomingRelationships={incomingRelationships}
            onEdit={onEditRelationship || (() => {})}
            onDelete={onDeleteRelationship || (() => {})}
          />
        </CardContent>
      )}
    </Card>
  );
}
