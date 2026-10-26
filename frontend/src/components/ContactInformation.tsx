import { useState, useMemo, useEffect } from 'react';
import { Card, CardContent, Divider, Stack, Box, Tabs, Tab, Button, Typography, FormControlLabel, Switch } from '@mui/material';
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
import RestaurantIcon from '@mui/icons-material/Restaurant';
import PeopleIcon from '@mui/icons-material/People';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import VisibilityOffIcon from '@mui/icons-material/VisibilityOff';
import { Checkbox } from '@mui/material';
import HeartBrokenIcon from '@mui/icons-material/HeartBroken';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';
import { DatePicker } from '@mui/x-date-pickers/DatePicker';
import { format, parse, isValid } from 'date-fns';
import { useTranslation } from 'react-i18next';
import EditableField from './EditableField';
import EditableArrayField from './EditableArrayField';
import MultiValueField from './MultiValueField';
import AddressFields from './AddressFields';
import RelationshipList from './RelationshipList';
import { Relationship, IncomingRelationship } from '../api/relationships';
import { Contact, ContactValue, ContactAddress } from '../api/contacts';
import { Reminder } from '../api/reminders';
import { ContactFieldKey, resolveEnabledFields } from '../contactFields';
import { useDateFormat, calculateAgeAtDate } from '../DateFormatProvider';
import { findBirthdayReminder } from '../utils/birthdayReminder';

interface ContactInformationProps {
  contact: Partial<Contact>;
  editingField: string | null;
  editValue: string;
  validationError: string;
  onEditStart: (field: string, value: string) => void;
  onEditCancel: () => void;
  onEditSave: (field: string) => void;
  onEditValueChange: (value: string) => void;
  onUpdateContact: (partial: Partial<Contact>) => Promise<void>;
  enabledFields?: Set<ContactFieldKey>;
  // When true, empty fields stay hidden until the user expands them
  hideEmptyFields?: boolean;
  // Relationship props
  relationships?: Relationship[];
  incomingRelationships?: IncomingRelationship[];
  onAddRelationship?: () => void;
  onEditRelationship?: (relationship: Relationship) => void;
  onDeleteRelationship?: (relationshipId: number) => void;
  // Custom fields
  customFieldNames?: string[];
  // Birthday reminder
  reminders?: Reminder[];
  onToggleBirthdayReminder?: () => void;
}

const iconSx = { mr: 1, color: 'text.secondary', fontSize: '1.2rem' };
const cloneValues = <T extends object>(v: T[]): T[] => v.map((x) => ({ ...x }));

// The registry fields this card renders; name parts and gender live in ContactHeader.
const INFO_FIELD_KEYS: ContactFieldKey[] = [
  'emails', 'phones', 'addresses', 'urls', 'impps', 'birthday', 'anniversary', 'is_deceased',
  'organization', 'department', 'job_title', 'role', 'work_information', 'food_preference',
  'how_we_met', 'contact_information',
];

export default function ContactInformation({
  contact,
  editingField,
  editValue,
  validationError,
  onEditStart,
  onEditCancel,
  onEditSave,
  onEditValueChange,
  onUpdateContact,
  enabledFields,
  hideEmptyFields = false,
  relationships = [],
  incomingRelationships = [],
  onAddRelationship,
  onEditRelationship,
  onDeleteRelationship,
  customFieldNames = [],
  reminders = [],
  onToggleBirthdayReminder,
}: ContactInformationProps) {
  const { t } = useTranslation();
  const { formatBirthday, getBirthdayPlaceholder, getDateFnsFormat, calculateAge } = useDateFormat();
  const [activeTab, setActiveTab] = useState(0);
  const enabled = enabledFields ?? resolveEnabledFields(null);
  const isOn = (key: ContactFieldKey) => enabled.has(key);

  // Expanded state is per contact: navigating to another contact collapses again.
  const [showAllFields, setShowAllFields] = useState(false);
  useEffect(() => setShowAllFields(false), [contact.ID]);
  const collapsed = hideEmptyFields && !showAllFields;

  const isFilled = (key: ContactFieldKey): boolean => {
    switch (key) {
      case 'emails':
      case 'phones':
      case 'addresses':
      case 'urls':
      case 'impps':
        return (contact[key]?.length ?? 0) > 0;
      case 'is_deceased':
        return !!contact.is_deceased;
      default: {
        const value = contact[key];
        return typeof value === 'string' && value.trim() !== '';
      }
    }
  };
  const isCustomFilled = (name: string) => (contact.custom_fields?.[name] ?? '').trim() !== '';

  // A field being edited stays visible even if it is (or becomes) empty.
  const show = (key: ContactFieldKey) => isOn(key) && (!collapsed || isFilled(key) || editingField === key);
  const showCustom = (name: string) =>
    !collapsed || isCustomFilled(name) || editingField === `custom_field_${name}`;

  const shownKeys = INFO_FIELD_KEYS.filter(isOn);
  const hasEmptyFields =
    shownKeys.some((key) => !isFilled(key)) || customFieldNames.some((name) => !isCustomFilled(name));
  const nothingShown =
    collapsed && !shownKeys.some(show) && !customFieldNames.some(showCustom);

  const hasBirthdayReminder = useMemo(
    () => !!(contact.birthday && findBirthdayReminder(reminders, contact.birthday)),
    [reminders, contact.birthday]
  );

  const birthdayAgeSuffix = useMemo(() => {
    if (!contact.birthday) return undefined;
    const age = contact.is_deceased && contact.deceased_date
      ? calculateAgeAtDate(contact.birthday, contact.deceased_date)
      : calculateAge(contact.birthday);
    if (age === null) return undefined;
    return t('dashboard.yearsOld', { age });
  }, [contact.birthday, contact.is_deceased, contact.deceased_date, t, calculateAge]);

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
        </Tabs>
      </Box>

      {/* General Information Tab */}
      {activeTab === 0 && (
        <CardContent sx={{ py: 2 }}>
          {hideEmptyFields && (showAllFields || hasEmptyFields) && (
            <Box sx={{ display: 'flex', justifyContent: 'flex-end', mb: 1.5 }}>
              <Button
                startIcon={showAllFields ? <VisibilityOffIcon /> : <EditIcon />}
                onClick={() => setShowAllFields(!showAllFields)}
                variant="outlined"
                size="small"
              >
                {showAllFields ? t('contactDetail.hideEmptyFields') : t('contactDetail.showAllFields')}
              </Button>
            </Box>
          )}
          {nothingShown && (
            <Typography variant="body2" color="text.secondary">
              {t('contactDetail.noFilledFields')}
            </Typography>
          )}
          <Stack spacing={2}>
            {show('emails') && (
              <EditableArrayField<ContactValue[]>
                icon={<EmailIcon sx={iconSx} />}
                label={t('contactDetail.email')}
                value={contact.emails || []}
                cloneValue={cloneValues}
                renderDisplay={renderValueList}
                renderEditor={(draft, setDraft) => (
                  <MultiValueField label={t('contacts.email')} value={draft} onChange={setDraft} valueType="email" defaultType="home" />
                )}
                onSave={(draft) => {
                  const clean = draft.filter((e) => e.value.trim());
                  return onUpdateContact({ emails: clean, email: clean[0]?.value || '' });
                }}
              />
            )}

            {show('phones') && (
              <EditableArrayField<ContactValue[]>
                icon={<PhoneIcon sx={iconSx} />}
                label={t('contactDetail.phone')}
                value={contact.phones || []}
                cloneValue={cloneValues}
                renderDisplay={renderValueList}
                renderEditor={(draft, setDraft) => (
                  <MultiValueField label={t('contacts.phone')} value={draft} onChange={setDraft} valueType="tel" defaultType="cell" />
                )}
                onSave={(draft) => {
                  const clean = draft.filter((p) => p.value.trim());
                  return onUpdateContact({ phones: clean, phone: clean[0]?.value || '' });
                }}
              />
            )}

            {show('addresses') && (
              <EditableArrayField<ContactAddress[]>
                icon={<HomeIcon sx={iconSx} />}
                label={t('contactDetail.address')}
                value={contact.addresses || []}
                cloneValue={cloneValues}
                renderDisplay={renderAddressList}
                renderEditor={(draft, setDraft) => (
                  <AddressFields label={t('contacts.address')} value={draft} onChange={setDraft} />
                )}
                onSave={(draft) => {
                  const clean = draft.filter(
                    (a) => a.street.trim() || a.city.trim() || a.region.trim() || a.postal.trim() || a.country.trim()
                  );
                  return onUpdateContact({ addresses: clean, address: clean.length === 0 ? '' : undefined });
                }}
              />
            )}

            {show('urls') && (
              <EditableArrayField<ContactValue[]>
                icon={<LanguageIcon sx={iconSx} />}
                label={t('contacts.urls')}
                value={contact.urls || []}
                cloneValue={cloneValues}
                renderDisplay={renderValueList}
                renderEditor={(draft, setDraft) => (
                  <MultiValueField label={t('contacts.urls')} value={draft} onChange={setDraft} valueType="url" defaultType="home" />
                )}
                onSave={(draft) => onUpdateContact({ urls: draft.filter((u) => u.value.trim()) })}
              />
            )}

            {show('impps') && (
              <EditableArrayField<ContactValue[]>
                icon={<ChatIcon sx={iconSx} />}
                label={t('contacts.impps')}
                value={contact.impps || []}
                cloneValue={cloneValues}
                renderDisplay={renderValueList}
                renderEditor={(draft, setDraft) => (
                  <MultiValueField label={t('contacts.impps')} value={draft} onChange={setDraft} defaultType="" freeTextType />
                )}
                onSave={(draft) => onUpdateContact({ impps: draft.filter((i) => i.value.trim()) })}
              />
            )}

            {show('birthday') && (
              <>
                <EditableField
                  icon={<CakeIcon sx={iconSx} />}
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
                {contact.birthday && onToggleBirthdayReminder && (
                  <FormControlLabel
                    sx={{ ml: 4 }}
                    control={
                      <Switch
                        checked={hasBirthdayReminder}
                        onChange={onToggleBirthdayReminder}
                      />
                    }
                    label={t('contactDetail.birthdayReminder')}
                  />
                )}
              </>
            )}

            {show('anniversary') && (
              <EditableField
                icon={<CelebrationIcon sx={iconSx} />}
                label={t('contacts.anniversary')}
                field="anniversary"
                value={contact.anniversary || ''}
                formattedDisplayValue={contact.anniversary ? formatBirthday(contact.anniversary) : undefined}
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

            {show('is_deceased') && (
              <EditableArrayField<{ isDeceased: boolean; deceasedDate: string }>
                icon={<HeartBrokenIcon sx={iconSx} />}
                label={t('contactDetail.deceased')}
                value={{
                  isDeceased: contact.is_deceased ?? false,
                  deceasedDate: contact.deceased_date ?? '',
                }}
                cloneValue={(v) => ({ ...v })}
                renderDisplay={(v) => {
                  if (!v.isDeceased) {
                    return <Typography variant="body2" color="text.disabled">—</Typography>;
                  }
                  const age = v.deceasedDate && contact.birthday
                    ? calculateAgeAtDate(contact.birthday, v.deceasedDate)
                    : null;
                  return (
                    <Typography variant="body2">
                      {v.deceasedDate ? formatBirthday(v.deceasedDate) : t('contactDetail.deceasedYes')}
                      {age !== null && ` ${t('dashboard.yearsOld', { age })}`}
                    </Typography>
                  );
                }}
                renderEditor={(draft, setDraft) => (
                  <Box>
                    <FormControlLabel
                      control={
                        <Checkbox
                          checked={draft.isDeceased}
                          onChange={(e) =>
                            setDraft({
                              isDeceased: e.target.checked,
                              deceasedDate: e.target.checked ? draft.deceasedDate : '',
                            })
                          }
                        />
                      }
                      label={t('contactDetail.deceased')}
                    />
                    {draft.isDeceased && (
                      <LocalizationProvider dateAdapter={AdapterDateFns}>
                        <DatePicker
                          label={t('contactDetail.deceasedDate')}
                          format={getDateFnsFormat()}
                          value={draft.deceasedDate ? parse(draft.deceasedDate, 'yyyy-MM-dd', new Date()) : null}
                          maxDate={new Date()}
                          minDate={contact.birthday && !contact.birthday.startsWith('--') ? parse(contact.birthday, 'yyyy-MM-dd', new Date()) : undefined}
                          onChange={(date) =>
                            setDraft({
                              ...draft,
                              deceasedDate: date && isValid(date) ? format(date, 'yyyy-MM-dd') : '',
                            })
                          }
                          slotProps={{ textField: { size: 'small', fullWidth: true, sx: { mt: 1 } } }}
                        />
                      </LocalizationProvider>
                    )}
                  </Box>
                )}
                onSave={(draft) =>
                  onUpdateContact({
                    is_deceased: draft.isDeceased,
                    deceased_date: draft.isDeceased ? draft.deceasedDate : '',
                  })
                }
              />
            )}

            {show('organization') && (
              <EditableField
                icon={<BusinessIcon sx={iconSx} />}
                label={t('contacts.organization')}
                field="organization"
                value={contact.organization || ''}
                isEditing={editingField === 'organization'}
                editValue={editValue}
                validationError={validationError}
                onEditStart={onEditStart}
                onEditCancel={onEditCancel}
                onEditSave={onEditSave}
                onEditValueChange={onEditValueChange}
              />
            )}

            {show('department') && (
              <EditableField
                icon={<BusinessIcon sx={iconSx} />}
                label={t('contacts.department')}
                field="department"
                value={contact.department || ''}
                isEditing={editingField === 'department'}
                editValue={editValue}
                validationError={validationError}
                onEditStart={onEditStart}
                onEditCancel={onEditCancel}
                onEditSave={onEditSave}
                onEditValueChange={onEditValueChange}
              />
            )}

            {show('job_title') && (
              <EditableField
                icon={<BadgeIcon sx={iconSx} />}
                label={t('contacts.jobTitle')}
                field="job_title"
                value={contact.job_title || ''}
                isEditing={editingField === 'job_title'}
                editValue={editValue}
                validationError={validationError}
                onEditStart={onEditStart}
                onEditCancel={onEditCancel}
                onEditSave={onEditSave}
                onEditValueChange={onEditValueChange}
              />
            )}

            {show('role') && (
              <EditableField
                icon={<BadgeIcon sx={iconSx} />}
                label={t('contacts.role')}
                field="role"
                value={contact.role || ''}
                isEditing={editingField === 'role'}
                editValue={editValue}
                validationError={validationError}
                onEditStart={onEditStart}
                onEditCancel={onEditCancel}
                onEditSave={onEditSave}
                onEditValueChange={onEditValueChange}
              />
            )}

            {show('work_information') && (
              <EditableField
                icon={<WorkIcon sx={{ ...iconSx, mt: 0.5 }} />}
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
            )}

            {show('food_preference') && (
              <EditableField
                icon={<RestaurantIcon sx={{ ...iconSx, mt: 0.5 }} />}
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
            )}

            {show('how_we_met') && (
              <EditableField
                icon={<PeopleIcon sx={{ ...iconSx, mt: 0.5 }} />}
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
            )}

            {show('contact_information') && (
              <EditableField
                icon={<NotesIcon sx={{ ...iconSx, mt: 0.5 }} />}
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
            )}

            {/* Custom Fields */}
            {customFieldNames.filter(showCustom).map((fieldName) => (
              <EditableField
                key={`custom_field_${fieldName}`}
                icon={<ClearAllIcon sx={{ ...iconSx, mt: 0.5 }} />}
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
