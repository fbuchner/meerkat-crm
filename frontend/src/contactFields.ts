// Single source of truth for the toggleable extended contact fields.
// Used by both the settings UI (ContactFieldSettings) and the contact form/detail.
// firstname/lastname are intentionally NOT listed here — they are always shown.

export type ContactFieldKey =
  | 'emails'
  | 'phones'
  | 'addresses'
  | 'urls'
  | 'impps'
  | 'nickname'
  | 'gender'
  | 'birthday'
  | 'anniversary'
  | 'prefix'
  | 'middle_name'
  | 'suffix'
  | 'organization'
  | 'department'
  | 'job_title'
  | 'role'
  | 'how_we_met'
  | 'food_preference'
  | 'work_information'
  | 'contact_information';

export interface ContactFieldDef {
  key: ContactFieldKey;
  /** i18n key for the field label */
  labelKey: string;
  /** group identifier used to render section subheaders in settings */
  group: 'communication' | 'name' | 'work' | 'personal' | 'meerkat';
}

export const CONTACT_FIELDS: ContactFieldDef[] = [
  { key: 'emails', labelKey: 'contacts.email', group: 'communication' },
  { key: 'phones', labelKey: 'contacts.phone', group: 'communication' },
  { key: 'addresses', labelKey: 'contacts.address', group: 'communication' },
  { key: 'urls', labelKey: 'contacts.urls', group: 'communication' },
  { key: 'impps', labelKey: 'contacts.impps', group: 'communication' },

  { key: 'prefix', labelKey: 'contacts.prefix', group: 'name' },
  { key: 'middle_name', labelKey: 'contacts.middleName', group: 'name' },
  { key: 'suffix', labelKey: 'contacts.suffix', group: 'name' },
  { key: 'nickname', labelKey: 'contacts.nickname', group: 'name' },

  { key: 'organization', labelKey: 'contacts.organization', group: 'work' },
  { key: 'department', labelKey: 'contacts.department', group: 'work' },
  { key: 'job_title', labelKey: 'contacts.jobTitle', group: 'work' },
  { key: 'role', labelKey: 'contacts.role', group: 'work' },
  { key: 'work_information', labelKey: 'contacts.workInformation', group: 'work' },

  { key: 'gender', labelKey: 'contacts.gender', group: 'personal' },
  { key: 'birthday', labelKey: 'contacts.birthday', group: 'personal' },
  { key: 'anniversary', labelKey: 'contacts.anniversary', group: 'personal' },

  { key: 'how_we_met', labelKey: 'contacts.howWeMet', group: 'meerkat' },
  { key: 'food_preference', labelKey: 'contacts.foodPreference', group: 'meerkat' },
  { key: 'contact_information', labelKey: 'contacts.contactInformation', group: 'meerkat' },
];

export const CONTACT_FIELD_GROUPS: ContactFieldDef['group'][] = [
  'communication',
  'name',
  'work',
  'personal',
  'meerkat',
];

// Default-enabled set = the fields shown today, so existing users see no change.
// New vCard fields (prefix/middle/suffix, org/dept/title/role, urls, impps, anniversary)
// are opt-in via Settings.
export const DEFAULT_ENABLED_CONTACT_FIELDS: ContactFieldKey[] = [
  'emails',
  'phones',
  'addresses',
  'nickname',
  'gender',
  'birthday',
  'how_we_met',
  'food_preference',
  'work_information',
  'contact_information',
];

// vCard TYPE options for typed multi-value fields (emails/phones/addresses/urls).
export const CONTACT_TYPE_OPTIONS = ['home', 'work', 'cell', 'fax', 'other'] as const;

// Resolves the stored setting (null/undefined => defaults) into a concrete enabled set.
export function resolveEnabledFields(stored: string[] | null | undefined): Set<ContactFieldKey> {
  const keys = stored == null ? DEFAULT_ENABLED_CONTACT_FIELDS : (stored as ContactFieldKey[]);
  return new Set(keys);
}
