// Single source of truth for the toggleable extended contact fields.
// Used by both the settings UI (ContactFieldSettings) and the contact form/detail.
// firstname/lastname are intentionally NOT listed here — they are always shown.
//
// Keys name the backend's nested Card/CRM concept they gate (see
// backend/openapi.yaml and api/contacts.ts's Card/CRMEnvelope types), not the
// legacy flat Contact shape's field names. Two renames and two consolidations
// fell out of that:
//   - 'urls' -> 'links' (Card.links), 'impps' -> 'imppAddresses'
//     (Card.imppAddresses): same concept, matching the wire name exactly.
//   - 'organization' + 'department' -> a single 'organizations' key: the
//     nested model has one Card.organizations array (department lives as a
//     unit *within* an organization entry), not two independent fields.
//   - 'job_title' + 'role' -> a single 'titles' key: both are entries in
//     Card.titles, differentiated by `kind: 'title' | 'role'`, not separate
//     fields.
// The UI these gate still renders one input per legacy scalar today (the
// adapter shim hasn't been retired yet) — `multiValue` below records which
// concepts are genuinely array-valued in the nested model so the components
// that migrate to real Card arrays (contactFields step 2+) know which
// sections need an add/remove list instead of a single input.

export type ContactFieldKey =
  | 'emails'
  | 'phones'
  | 'addresses'
  | 'links'
  | 'imppAddresses'
  | 'nickname'
  | 'gender'
  | 'birthday'
  | 'anniversary'
  | 'prefix'
  | 'middle_name'
  | 'suffix'
  | 'organizations'
  | 'titles'
  | 'how_we_met'
  | 'work_information'
  | 'contact_information';

export interface ContactFieldDef {
  key: ContactFieldKey;
  /** i18n key for the field label */
  labelKey: string;
  /** group identifier used to render section subheaders in settings */
  group: 'communication' | 'name' | 'work' | 'personal' | 'mycorrhizal';
  /** true if this concept is an array in the nested Card model (vs. a scalar) */
  multiValue: boolean;
}

export const CONTACT_FIELDS: ContactFieldDef[] = [
  { key: 'emails', labelKey: 'contacts.email', group: 'communication', multiValue: true },
  { key: 'phones', labelKey: 'contacts.phone', group: 'communication', multiValue: true },
  { key: 'addresses', labelKey: 'contacts.address', group: 'communication', multiValue: true },
  { key: 'links', labelKey: 'contacts.urls', group: 'communication', multiValue: true },
  { key: 'imppAddresses', labelKey: 'contacts.impps', group: 'communication', multiValue: true },

  { key: 'prefix', labelKey: 'contacts.prefix', group: 'name', multiValue: false },
  { key: 'middle_name', labelKey: 'contacts.middleName', group: 'name', multiValue: false },
  { key: 'suffix', labelKey: 'contacts.suffix', group: 'name', multiValue: false },
  { key: 'nickname', labelKey: 'contacts.nickname', group: 'name', multiValue: false },

  { key: 'organizations', labelKey: 'contacts.organization', group: 'work', multiValue: true },
  { key: 'titles', labelKey: 'contacts.titles', group: 'work', multiValue: true },
  { key: 'work_information', labelKey: 'contacts.workInformation', group: 'work', multiValue: false },

  { key: 'gender', labelKey: 'contacts.gender', group: 'personal', multiValue: false },
  { key: 'birthday', labelKey: 'contacts.birthday', group: 'personal', multiValue: false },
  { key: 'anniversary', labelKey: 'contacts.anniversary', group: 'personal', multiValue: false },

  { key: 'how_we_met', labelKey: 'contacts.howWeMet', group: 'mycorrhizal', multiValue: false },
  { key: 'contact_information', labelKey: 'contacts.contactInformation', group: 'mycorrhizal', multiValue: false },
];

export const CONTACT_FIELD_GROUPS: ContactFieldDef['group'][] = [
  'communication',
  'name',
  'work',
  'personal',
  'mycorrhizal',
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
