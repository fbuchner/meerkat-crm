import { useState, useMemo } from 'react';
import { Card, CardContent, Divider, Stack, Box, Tabs, Tab, Button, Typography } from '@mui/material';
import EmailIcon from '@mui/icons-material/Email';
import PhoneIcon from '@mui/icons-material/Phone';
import CakeIcon from '@mui/icons-material/Cake';
import CelebrationIcon from '@mui/icons-material/Celebration';
import HomeIcon from '@mui/icons-material/Home';
import WorkIcon from '@mui/icons-material/Work';
import BusinessIcon from '@mui/icons-material/Business';
import BadgeIcon from '@mui/icons-material/Badge';
import LanguageIcon from '@mui/icons-material/Language';
import ChatIcon from '@mui/icons-material/Chat';
import NotesIcon from '@mui/icons-material/Notes';
import ClearAllIcon from '@mui/icons-material/ClearAll';
import PeopleIcon from '@mui/icons-material/People';
import AddIcon from '@mui/icons-material/Add';
import { useTranslation } from 'react-i18next';
import EditableField from './EditableField';
import EditableArrayField from './EditableArrayField';
import MultiValueField from './MultiValueField';
import AddressFields from './AddressFields';
import RelationshipEdgeList from './RelationshipEdgeList';
import LifeEventList from './LifeEventList';
import PreferenceList from './PreferenceList';
import { RelationshipEdge } from '../api/relationshipEdges';
import { LifeEvent } from '../api/lifeEvents';
import { Preference } from '../api/preferences';
import {
  Card as CardModel,
  CRMEnvelope,
  Contact,
  ContactValue,
  ContactAddress,
  cardEmailsToValues,
  valuesToCardEmails,
  cardPhonesToValues,
  valuesToCardPhones,
  cardLinksToValues,
  valuesToCardLinks,
  cardImppToValues,
  valuesToCardImpp,
  cardAddressesToValues,
  valuesToCardAddresses,
  getAnniversaryField,
  getOrganizationFields,
  getTitleField,
} from '../api/contacts';
import { ContactFieldKey, resolveEnabledFields } from '../contactFields';
import { useDateFormat } from '../DateFormatProvider';

interface ContactInformationProps {
  card: CardModel;
  crm: CRMEnvelope;
  editingField: string | null;
  editValue: string;
  validationError: string;
  onEditStart: (field: string, value: string) => void;
  onEditCancel: () => void;
  onEditSave: (field: string) => void;
  onEditValueChange: (value: string) => void;
  onUpdateCard: (patch: Partial<CardModel>) => Promise<void>;
  enabledFields?: Set<ContactFieldKey>;
  // RelationshipEdge props (§3d WP3)
  confirmedEdges?: RelationshipEdge[];
  suggestedEdges?: RelationshipEdge[];
  contactsByUid?: Map<string, Contact>;
  viewedContactUid?: string;
  onAddRelationshipEdge?: () => void;
  onEditRelationshipEdge?: (edge: RelationshipEdge) => void;
  onDeleteRelationshipEdge?: (edgeId: string) => void;
  onAcceptSuggestion?: (edgeId: string) => void;
  onRejectSuggestion?: (edgeId: string) => void;
  // LifeEvent props (T5)
  lifeEvents?: LifeEvent[];
  lifeEventsContactsByUid?: Map<string, Contact>;
  onAddLifeEvent?: () => void;
  onEditLifeEvent?: (event: LifeEvent) => void;
  onDeleteLifeEvent?: (id: string) => void;
  // Preference props (T20a)
  preferences?: Preference[];
  onAddPreference?: () => void;
  onEditPreference?: (preference: Preference) => void;
  onDeletePreference?: (id: string) => void;
  // Custom fields
  customFieldNames?: string[];
}

const iconSx = { mr: 1, color: 'text.secondary', fontSize: '1.2rem' };
const cloneValues = <T extends object>(v: T[]): T[] => v.map((x) => ({ ...x }));

export default function ContactInformation({
  card,
  crm,
  editingField,
  editValue,
  validationError,
  onEditStart,
  onEditCancel,
  onEditSave,
  onEditValueChange,
  onUpdateCard,
  enabledFields,
  confirmedEdges = [],
  suggestedEdges = [],
  contactsByUid,
  viewedContactUid,
  onAddRelationshipEdge,
  onEditRelationshipEdge,
  onDeleteRelationshipEdge,
  onAcceptSuggestion,
  onRejectSuggestion,
  lifeEvents = [],
  lifeEventsContactsByUid,
  onAddLifeEvent,
  onEditLifeEvent,
  onDeleteLifeEvent,
  preferences = [],
  onAddPreference,
  onEditPreference,
  onDeletePreference,
  customFieldNames = [],
}: ContactInformationProps) {
  const { t } = useTranslation();
  const { formatBirthday, getBirthdayPlaceholder, calculateAge } = useDateFormat();
  const [activeTab, setActiveTab] = useState(0);
  const enabled = enabledFields ?? resolveEnabledFields(null);
  const isOn = (key: ContactFieldKey) => enabled.has(key);

  const birthday = getAnniversaryField(card.anniversaries, 'birth') || '';
  const anniversary = getAnniversaryField(card.anniversaries, 'wedding') || '';
  const { organization = '', department = '' } = getOrganizationFields(card.organizations);
  const jobTitle = getTitleField(card.titles, 'title') || '';
  const role = getTitleField(card.titles, 'role') || '';

  const birthdayAgeSuffix = useMemo(() => {
    if (!birthday) return undefined;
    const age = calculateAge(birthday);
    if (age === null) return undefined;
    return t('dashboard.yearsOld', { age });
  }, [birthday, t, calculateAge]);

  const renderValueList = (rows: ContactValue[] | undefined) => {
    if (!rows || rows.length === 0) return <Typography variant="body2" color="text.disabled">—</Typography>;
    return (
      <Stack>
        {rows.map((r, i) => (
          <Typography key={i} variant="body2">
            {r.value}
            {r.type ? ` (${t(`contacts.types.${r.type}`, r.type)})` : ''}
          </Typography>
        ))}
      </Stack>
    );
  };

  const renderAddressList = (rows: ContactAddress[] | undefined) => {
    if (!rows || rows.length === 0) return <Typography variant="body2" color="text.disabled">—</Typography>;
    return (
      <Stack spacing={0.5}>
        {rows.map((a, i) => (
          <Typography key={i} variant="body2">
            {[a.street, a.city, a.region, a.postal, a.country].filter(Boolean).join(', ')}
            {a.type ? ` (${t(`contacts.types.${a.type}`, a.type)})` : ''}
          </Typography>
        ))}
      </Stack>
    );
  };

  return (
    <Card sx={{ flex: 1 }}>
      <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
        <Tabs value={activeTab} onChange={(_, newValue) => setActiveTab(newValue)} aria-label="contact information tabs">
          <Tab label={t('contactDetail.generalInfo')} />
          <Tab label={t('relationships.title')} />
          <Tab label={t('lifeEvent.title')} />
          <Tab label={t('preference.title')} />
        </Tabs>
      </Box>

      {/* General Information Tab */}
      {activeTab === 0 && (
        <CardContent sx={{ py: 2 }}>
          <Stack spacing={2}>
            {isOn('emails') && (
              <EditableArrayField<ContactValue[]>
                icon={<EmailIcon sx={iconSx} />}
                label={t('contactDetail.email')}
                value={cardEmailsToValues(card.emails)}
                cloneValue={cloneValues}
                renderDisplay={renderValueList}
                renderEditor={(draft, setDraft) => (
                  <MultiValueField label={t('contacts.email')} value={draft} onChange={setDraft} valueType="email" defaultType="home" />
                )}
                onSave={(draft) => onUpdateCard({ emails: valuesToCardEmails(draft) })}
              />
            )}

            {isOn('phones') && (
              <EditableArrayField<ContactValue[]>
                icon={<PhoneIcon sx={iconSx} />}
                label={t('contactDetail.phone')}
                value={cardPhonesToValues(card.phones)}
                cloneValue={cloneValues}
                renderDisplay={renderValueList}
                renderEditor={(draft, setDraft) => (
                  <MultiValueField label={t('contacts.phone')} value={draft} onChange={setDraft} valueType="tel" defaultType="cell" />
                )}
                onSave={(draft) => onUpdateCard({ phones: valuesToCardPhones(draft) })}
              />
            )}

            {isOn('addresses') && (
              <EditableArrayField<ContactAddress[]>
                icon={<HomeIcon sx={iconSx} />}
                label={t('contactDetail.address')}
                value={cardAddressesToValues(card.addresses)}
                cloneValue={cloneValues}
                renderDisplay={renderAddressList}
                renderEditor={(draft, setDraft) => (
                  <AddressFields label={t('contacts.address')} value={draft} onChange={setDraft} />
                )}
                onSave={(draft) => onUpdateCard({ addresses: valuesToCardAddresses(draft) })}
              />
            )}

            {isOn('links') && (
              <EditableArrayField<ContactValue[]>
                icon={<LanguageIcon sx={iconSx} />}
                label={t('contacts.urls')}
                value={cardLinksToValues(card.links)}
                cloneValue={cloneValues}
                renderDisplay={renderValueList}
                renderEditor={(draft, setDraft) => (
                  <MultiValueField label={t('contacts.urls')} value={draft} onChange={setDraft} valueType="url" defaultType="home" />
                )}
                onSave={(draft) => onUpdateCard({ links: valuesToCardLinks(draft) })}
              />
            )}

            {isOn('imppAddresses') && (
              <EditableArrayField<ContactValue[]>
                icon={<ChatIcon sx={iconSx} />}
                label={t('contacts.impps')}
                value={cardImppToValues(card.imppAddresses)}
                cloneValue={cloneValues}
                renderDisplay={renderValueList}
                renderEditor={(draft, setDraft) => (
                  <MultiValueField label={t('contacts.impps')} value={draft} onChange={setDraft} defaultType="" freeTextType />
                )}
                onSave={(draft) => onUpdateCard({ imppAddresses: valuesToCardImpp(draft) })}
              />
            )}

            {isOn('birthday') && (
              <EditableField
                icon={<CakeIcon sx={iconSx} />}
                label={t('contactDetail.birthday')}
                field="birthday"
                value={birthday}
                formattedDisplayValue={birthday ? formatBirthday(birthday) : undefined}
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
            )}

            {isOn('anniversary') && (
              <EditableField
                icon={<CelebrationIcon sx={iconSx} />}
                label={t('contacts.anniversary')}
                field="anniversary"
                value={anniversary}
                formattedDisplayValue={anniversary ? formatBirthday(anniversary) : undefined}
                placeholder={getBirthdayPlaceholder()}
                isEditing={editingField === 'anniversary'}
                editValue={editValue}
                validationError={validationError}
                onEditStart={onEditStart}
                onEditCancel={onEditCancel}
                onEditSave={onEditSave}
                onEditValueChange={onEditValueChange}
              />
            )}

            {isOn('organizations') && (
              <>
                <EditableField
                  icon={<BusinessIcon sx={iconSx} />}
                  label={t('contacts.organization')}
                  field="organization"
                  value={organization}
                  isEditing={editingField === 'organization'}
                  editValue={editValue}
                  validationError={validationError}
                  onEditStart={onEditStart}
                  onEditCancel={onEditCancel}
                  onEditSave={onEditSave}
                  onEditValueChange={onEditValueChange}
                />
                <EditableField
                  icon={<BusinessIcon sx={iconSx} />}
                  label={t('contacts.department')}
                  field="department"
                  value={department}
                  isEditing={editingField === 'department'}
                  editValue={editValue}
                  validationError={validationError}
                  onEditStart={onEditStart}
                  onEditCancel={onEditCancel}
                  onEditSave={onEditSave}
                  onEditValueChange={onEditValueChange}
                />
              </>
            )}

            {isOn('titles') && (
              <>
                <EditableField
                  icon={<BadgeIcon sx={iconSx} />}
                  label={t('contacts.jobTitle')}
                  field="job_title"
                  value={jobTitle}
                  isEditing={editingField === 'job_title'}
                  editValue={editValue}
                  validationError={validationError}
                  onEditStart={onEditStart}
                  onEditCancel={onEditCancel}
                  onEditSave={onEditSave}
                  onEditValueChange={onEditValueChange}
                />
                <EditableField
                  icon={<BadgeIcon sx={iconSx} />}
                  label={t('contacts.role')}
                  field="role"
                  value={role}
                  isEditing={editingField === 'role'}
                  editValue={editValue}
                  validationError={validationError}
                  onEditStart={onEditStart}
                  onEditCancel={onEditCancel}
                  onEditSave={onEditSave}
                  onEditValueChange={onEditValueChange}
                />
              </>
            )}

            {isOn('work_information') && (
              <EditableField
                icon={<WorkIcon sx={{ ...iconSx, mt: 0.5 }} />}
                label={t('contactDetail.workInfo')}
                field="work_information"
                value={crm.work_information || ''}
                multiline
                isEditing={editingField === 'work_information'}
                editValue={editValue}
                validationError={validationError}
                onEditStart={onEditStart}
                onEditCancel={onEditCancel}
                onEditSave={onEditSave}
                onEditValueChange={onEditValueChange}
              />
            )}

            {isOn('how_we_met') && (
              <EditableField
                icon={<PeopleIcon sx={{ ...iconSx, mt: 0.5 }} />}
                label={t('contactDetail.howWeMet')}
                field="how_we_met"
                value={crm.how_we_met || ''}
                multiline
                isEditing={editingField === 'how_we_met'}
                editValue={editValue}
                validationError={validationError}
                onEditStart={onEditStart}
                onEditCancel={onEditCancel}
                onEditSave={onEditSave}
                onEditValueChange={onEditValueChange}
              />
            )}

            {isOn('contact_information') && (
              <EditableField
                icon={<NotesIcon sx={{ ...iconSx, mt: 0.5 }} />}
                label={t('contactDetail.additionalInfo')}
                field="contact_information"
                value={crm.contact_information || ''}
                multiline
                isEditing={editingField === 'contact_information'}
                editValue={editValue}
                validationError={validationError}
                onEditStart={onEditStart}
                onEditCancel={onEditCancel}
                onEditSave={onEditSave}
                onEditValueChange={onEditValueChange}
              />
            )}

            {/* Custom Fields */}
            {customFieldNames.map((fieldName) => (
              <EditableField
                key={`custom_field_${fieldName}`}
                icon={<ClearAllIcon sx={{ ...iconSx, mt: 0.5 }} />}
                label={fieldName}
                field={`custom_field_${fieldName}`}
                value={crm.custom_fields?.[fieldName] || ''}
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
              onClick={onAddRelationshipEdge}
              variant="outlined"
              size="small"
            >
              {t('relationships.addRelationship')}
            </Button>
          </Box>
          <Divider sx={{ mb: 1.5 }} />
          <RelationshipEdgeList
            confirmedEdges={confirmedEdges}
            suggestedEdges={suggestedEdges}
            contactsByUid={contactsByUid || new Map()}
            viewedContactUid={viewedContactUid || ''}
            onEdit={onEditRelationshipEdge || (() => {})}
            onDelete={onDeleteRelationshipEdge || (() => {})}
            onAccept={onAcceptSuggestion || (() => {})}
            onReject={onRejectSuggestion || (() => {})}
          />
        </CardContent>
      )}

      {/* Life Events Tab */}
      {activeTab === 2 && (
        <CardContent sx={{ py: 2 }}>
          <Box sx={{ display: 'flex', justifyContent: 'flex-end', mb: 1.5 }}>
            <Button
              startIcon={<AddIcon />}
              onClick={onAddLifeEvent}
              variant="outlined"
              size="small"
            >
              {t('lifeEvent.add')}
            </Button>
          </Box>
          <Divider sx={{ mb: 1.5 }} />
          <LifeEventList
            events={lifeEvents}
            contactsByUid={lifeEventsContactsByUid || new Map()}
            onEdit={onEditLifeEvent || (() => {})}
            onDelete={onDeleteLifeEvent || (() => {})}
          />
        </CardContent>
      )}

      {/* Preferences Tab (T20a) */}
      {activeTab === 3 && (
        <CardContent sx={{ py: 2 }}>
          <Box sx={{ display: 'flex', justifyContent: 'flex-end', mb: 1.5 }}>
            <Button
              startIcon={<AddIcon />}
              onClick={onAddPreference}
              variant="outlined"
              size="small"
            >
              {t('preference.add')}
            </Button>
          </Box>
          <Divider sx={{ mb: 1.5 }} />
          <PreferenceList
            preferences={preferences}
            onEdit={onEditPreference || (() => {})}
            onDelete={onDeletePreference || (() => {})}
          />
        </CardContent>
      )}
    </Card>
  );
}
