// Contact-related API calls
import { apiFetch, API_BASE_URL, getAuthHeaders, parseErrorResponse } from './client';

export interface ContactValue {
  type: string;
  value: string;
}

export interface ContactAddress {
  type: string;
  street: string;
  city: string;
  region: string;
  postal: string;
  country: string;
}

// Contact is the flat shape every existing component (ContactDetailPage,
// AddContactDialog, ContactInformation, ContactHeader, ...) still consumes.
// The backend no longer speaks this shape on the wire -- see the
// toLegacyContact / toContactRecordInput adapter below, which is the
// temporary translation layer that lets those components keep working
// unmodified while the wire format underneath is the real nested card/crm
// shape. This shim (and this Contact type) goes away once every consumer
// is migrated to the nested model directly (see the frontend migration
// task list); do not add new features here that assume it's permanent.
export interface Contact {
  ID: number;
  firstname: string;
  lastname: string;
  nickname?: string;
  gender?: string;
  email?: string;
  phone?: string;
  birthday?: string;
  photo?: string;
  address?: string;
  how_we_met?: string;
  food_preference?: string;
  work_information?: string;
  contact_information?: string;
  circles?: string[];
  photo_thumbnail?: string;
  custom_fields?: Record<string, string>;
  archived?: boolean;
  // Multi-valued vCard fields
  emails?: ContactValue[];
  phones?: ContactValue[];
  addresses?: ContactAddress[];
  urls?: ContactValue[];
  impps?: ContactValue[];
  // Structured name parts
  prefix?: string;
  middle_name?: string;
  suffix?: string;
  // Organizational fields
  organization?: string;
  department?: string;
  job_title?: string;
  role?: string;
  anniversary?: string;
}

// ---------------------------------------------------------------------------
// Nested wire types (mirror backend/openapi.yaml's Card/CRMEnvelope/
// ContactRecordInput/ContactRecordResponse schemas). Deliberately partial --
// only the fields the adapter below actually reads/writes are typed; the
// backend may send more (e.g. localizations, relatedTo) that this shim
// ignores rather than losing the app's ability to build if a field is
// missing here.
// ---------------------------------------------------------------------------

interface NameComponent {
  kind: 'title' | 'given' | 'given2' | 'surname' | 'surname2' | 'credential' | 'generation' | 'separator';
  value: string;
}

interface CardName {
  components?: NameComponent[];
  full?: string;
}

interface CardNickname {
  name: string;
}

interface CardOrgUnit {
  name: string;
}

interface CardOrganization {
  id?: string;
  name?: string;
  units?: CardOrgUnit[];
}

interface CardTitle {
  name: string;
  kind?: 'title' | 'role';
}

interface CardEmail {
  address: string;
  contexts?: string[];
}

interface CardPhone {
  number: string;
  features?: string[];
  contexts?: string[];
}

interface CardOnlineService {
  uri?: string;
  service?: string;
  contexts?: string[];
}

interface CardResource {
  uri: string;
  kind?: string;
  mediaType?: string;
  contexts?: string[];
}

interface CardAddressComponent {
  kind: string;
  value: string;
}

interface CardAddress {
  components?: CardAddressComponent[];
  countryCode?: string;
  contexts?: string[];
}

interface CardPartialDate {
  year?: number | null;
  month?: number | null;
  day?: number | null;
}

interface CardAnniversaryDate {
  partial?: CardPartialDate;
  timestamp?: string | null;
}

interface CardAnniversary {
  kind: 'birth' | 'death' | 'wedding';
  date: CardAnniversaryDate;
}

interface Card {
  name?: CardName;
  nicknames?: CardNickname[];
  organizations?: CardOrganization[];
  titles?: CardTitle[];
  emails?: CardEmail[];
  phones?: CardPhone[];
  imppAddresses?: CardOnlineService[];
  addresses?: CardAddress[];
  anniversaries?: CardAnniversary[];
  keywords?: string[];
  media?: CardResource[];
  links?: CardResource[];
}

interface CRMEnvelope {
  circles?: string[];
  how_we_met?: string;
  food_preference?: string;
  work_information?: string;
  contact_information?: string;
  custom_fields?: Record<string, string>;
}

export interface ContactRecordInput {
  gender?: string;
  card: Card;
  crm: CRMEnvelope;
}

export interface ContactRecordResponse {
  id: number;
  uid: string;
  etag: string;
  gender?: string;
  card: Card;
  crm: CRMEnvelope;
  photo?: string;
  photo_thumbnail?: string;
  archived?: boolean;
}

// ContactSummaryDTO mirrors the backend's slim GET /contacts list
// projection exactly (models.ContactSummary) -- distinct from the legacy
// Contact shape above, which the adapter maps summaries down into.
interface ContactSummaryDTO {
  id: number;
  uid: string;
  firstname: string;
  lastname: string;
  nickname: string;
  fn: string;
  primary_email: string;
  primary_phone: string;
  birthday: string;
  org: string;
  photo: string;
  photo_thumbnail: string;
  circles: string[];
  archived: boolean;
}

// ---------------------------------------------------------------------------
// Adapter: nested wire shape <-> legacy flat Contact shape.
// ---------------------------------------------------------------------------

function nameComponentValue(components: NameComponent[] | undefined, kind: NameComponent['kind']): string | undefined {
  return components?.find((c) => c.kind === kind)?.value;
}

// formatAnniversaryDate turns a CardAnniversaryDate into the ISO YYYY-MM-DD
// / year-less --MM-DD string convention used throughout this app for
// birthday/anniversary fields (never a full RFC3339 timestamp for a
// date-only field).
function formatAnniversaryDate(date: CardAnniversaryDate | undefined): string | undefined {
  if (!date) return undefined;
  if (date.partial) {
    const { year, month, day } = date.partial;
    const mm = month != null ? String(month).padStart(2, '0') : undefined;
    const dd = day != null ? String(day).padStart(2, '0') : undefined;
    if (mm && dd) {
      return year != null ? `${String(year).padStart(4, '0')}-${mm}-${dd}` : `--${mm}-${dd}`;
    }
  }
  if (date.timestamp) {
    return date.timestamp.slice(0, 10);
  }
  return undefined;
}

// parseAnniversaryDate is formatAnniversaryDate's inverse: a
// YYYY-MM-DD or --MM-DD string into a CardAnniversaryDate.
function parseAnniversaryDate(value: string): CardAnniversaryDate {
  const yearLess = /^--(\d{2})-(\d{2})$/.exec(value);
  if (yearLess) {
    return { partial: { month: parseInt(yearLess[1], 10), day: parseInt(yearLess[2], 10) } };
  }
  const full = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
  if (full) {
    return { partial: { year: parseInt(full[1], 10), month: parseInt(full[2], 10), day: parseInt(full[3], 10) } };
  }
  // Unparseable input: pass through as a raw timestamp rather than
  // dropping it silently, matching the backend's own degradation policy.
  return { timestamp: value };
}

// toLegacyContact maps a full ContactRecordResponse (GET/POST/PUT
// /contacts/{id}) down into the flat Contact shape.
export function toLegacyContact(record: ContactRecordResponse): Contact {
  const card = record.card || {};
  const crm = record.crm || {};

  const emails = (card.emails || []).map((e) => ({ type: e.contexts?.[0] || '', value: e.address }));
  const phones = (card.phones || []).map((p) => ({ type: p.features?.[0] || p.contexts?.[0] || '', value: p.number }));
  const urls = (card.links || []).map((l) => ({ type: l.contexts?.[0] || '', value: l.uri }));
  const impps = (card.imppAddresses || []).map((i) => ({ type: i.contexts?.[0] || '', value: i.uri || '' }));
  const addresses = (card.addresses || []).map((a) => {
    const comps = a.components || [];
    const find = (kind: string) => comps.find((c) => c.kind === kind)?.value || '';
    return {
      type: a.contexts?.[0] || '',
      street: find('name') || find('number'),
      city: find('locality'),
      region: find('region'),
      postal: find('postcode'),
      country: find('country') || a.countryCode || '',
    };
  });

  const birthAnniversary = (card.anniversaries || []).find((a) => a.kind === 'birth');
  const weddingAnniversary = (card.anniversaries || []).find((a) => a.kind === 'wedding');

  const org = card.organizations?.[0];
  const titleEntry = card.titles?.find((t) => t.kind === 'title' || !t.kind);
  const roleEntry = card.titles?.find((t) => t.kind === 'role');

  return {
    ID: record.id,
    firstname: nameComponentValue(card.name?.components, 'given') || '',
    lastname: nameComponentValue(card.name?.components, 'surname') || '',
    nickname: card.nicknames?.[0]?.name,
    prefix: nameComponentValue(card.name?.components, 'title'),
    middle_name: nameComponentValue(card.name?.components, 'given2'),
    suffix: nameComponentValue(card.name?.components, 'generation'),
    gender: record.gender,
    email: emails[0]?.value,
    phone: phones[0]?.value,
    birthday: formatAnniversaryDate(birthAnniversary?.date),
    anniversary: formatAnniversaryDate(weddingAnniversary?.date),
    photo: record.photo,
    photo_thumbnail: record.photo_thumbnail,
    address: addresses[0]?.street,
    how_we_met: crm.how_we_met,
    food_preference: crm.food_preference,
    work_information: crm.work_information,
    contact_information: crm.contact_information,
    circles: crm.circles,
    custom_fields: crm.custom_fields,
    archived: record.archived,
    emails,
    phones,
    addresses,
    urls,
    impps,
    organization: org?.name,
    department: org?.units?.[0]?.name,
    job_title: titleEntry?.name,
    role: roleEntry?.name,
  };
}

// summaryToLegacyContact maps the slim GET /contacts list item shape down
// into the same flat Contact shape ContactsPage/DashboardPage etc. render.
function summaryToLegacyContact(summary: ContactSummaryDTO): Contact {
  return {
    ID: summary.id,
    firstname: summary.firstname,
    lastname: summary.lastname,
    nickname: summary.nickname || undefined,
    email: summary.primary_email || undefined,
    phone: summary.primary_phone || undefined,
    birthday: summary.birthday || undefined,
    photo: summary.photo || undefined,
    photo_thumbnail: summary.photo_thumbnail || undefined,
    circles: summary.circles,
    organization: summary.org || undefined,
    archived: summary.archived,
  };
}

// toContactRecordInput builds the nested request body from the flat
// form-data shape AddContactDialog/ContactDetailPage construct today.
export function toContactRecordInput(data: Partial<Contact>): ContactRecordInput {
  const nameComponents: NameComponent[] = [];
  if (data.prefix) nameComponents.push({ kind: 'title', value: data.prefix });
  if (data.firstname) nameComponents.push({ kind: 'given', value: data.firstname });
  if (data.middle_name) nameComponents.push({ kind: 'given2', value: data.middle_name });
  if (data.lastname) nameComponents.push({ kind: 'surname', value: data.lastname });
  if (data.suffix) nameComponents.push({ kind: 'generation', value: data.suffix });

  const emails: CardEmail[] = (data.emails && data.emails.length > 0
    ? data.emails
    : data.email
      ? [{ type: '', value: data.email }]
      : []
  ).map((e) => ({ address: e.value, contexts: e.type ? [e.type] : undefined }));

  const phones: CardPhone[] = (data.phones && data.phones.length > 0
    ? data.phones
    : data.phone
      ? [{ type: '', value: data.phone }]
      : []
  ).map((p) => ({ number: p.value, contexts: p.type ? [p.type] : undefined }));

  const links: CardResource[] = (data.urls || []).map((u) => ({ uri: u.value, contexts: u.type ? [u.type] : undefined }));
  const imppAddresses: CardOnlineService[] = (data.impps || []).map((i) => ({ uri: i.value, contexts: i.type ? [i.type] : undefined }));

  const addresses: CardAddress[] = (data.addresses && data.addresses.length > 0
    ? data.addresses
    : data.address
      ? [{ type: '', street: data.address, city: '', region: '', postal: '', country: '' }]
      : []
  ).map((a) => {
    const components: CardAddressComponent[] = [];
    if (a.street) components.push({ kind: 'name', value: a.street });
    if (a.city) components.push({ kind: 'locality', value: a.city });
    if (a.region) components.push({ kind: 'region', value: a.region });
    if (a.postal) components.push({ kind: 'postcode', value: a.postal });
    if (a.country) components.push({ kind: 'country', value: a.country });
    return { components, contexts: a.type ? [a.type] : undefined };
  });

  const anniversaries: CardAnniversary[] = [];
  if (data.birthday) anniversaries.push({ kind: 'birth', date: parseAnniversaryDate(data.birthday) });
  if (data.anniversary) anniversaries.push({ kind: 'wedding', date: parseAnniversaryDate(data.anniversary) });

  const organizations: CardOrganization[] = data.organization
    ? [{ name: data.organization, units: data.department ? [{ name: data.department }] : undefined }]
    : [];
  const titles: CardTitle[] = [];
  if (data.job_title) titles.push({ name: data.job_title, kind: 'title' });
  if (data.role) titles.push({ name: data.role, kind: 'role' });

  return {
    gender: data.gender,
    card: {
      name: nameComponents.length > 0 ? { components: nameComponents } : undefined,
      nicknames: data.nickname ? [{ name: data.nickname }] : undefined,
      emails: emails.length > 0 ? emails : undefined,
      phones: phones.length > 0 ? phones : undefined,
      links: links.length > 0 ? links : undefined,
      imppAddresses: imppAddresses.length > 0 ? imppAddresses : undefined,
      addresses: addresses.length > 0 ? addresses : undefined,
      anniversaries: anniversaries.length > 0 ? anniversaries : undefined,
      organizations: organizations.length > 0 ? organizations : undefined,
      titles: titles.length > 0 ? titles : undefined,
    },
    crm: {
      circles: data.circles,
      how_we_met: data.how_we_met,
      food_preference: data.food_preference,
      work_information: data.work_information,
      contact_information: data.contact_information,
      custom_fields: data.custom_fields,
    },
  };
}

export interface Birthday {
  type: 'contact' | 'relationship';
  name: string;
  birthday: string;
  photo_thumbnail?: string;
  contact_id: number;
  relationship_type?: string;
  associated_contact_name?: string;
}

export interface ContactsResponse {
  contacts: Contact[];
  total: number;
  page: number;
  limit: number;
}

export interface GetContactsParams {
  page?: number;
  limit?: number;
  search?: string;
  circle?: string;
  sort?: string;
  order?: string;
  includeArchived?: boolean;
  archived?: boolean;
}

// Get all contacts with pagination and filters
export async function getContacts(
  params: GetContactsParams
): Promise<ContactsResponse> {
  const { page = 1, limit = 25, search = '', circle = '', sort, order, includeArchived, archived } = params;

  const queryParams = new URLSearchParams({
    page: page.toString(),
    limit: limit.toString(),
  });

  if (search) queryParams.append('search', search);
  if (circle) queryParams.append('circle', circle);
  if (sort) queryParams.append('sort', sort);
  if (order) queryParams.append('order', order);
  if (includeArchived) queryParams.append('include_archived', 'true');
  if (archived !== undefined) queryParams.append('archived', archived.toString());

  const response = await apiFetch(
    `${API_BASE_URL}/contacts?${queryParams.toString()}`,
    { headers: getAuthHeaders() }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  const data: { contacts: ContactSummaryDTO[]; total: number; page: number; limit: number } = await response.json();
  return {
    contacts: data.contacts.map(summaryToLegacyContact),
    total: data.total,
    page: data.page,
    limit: data.limit,
  };
}

// Get single contact. The backend always returns the full
// ContactRecordResponse now (the old fields= partial-projection param is
// gone); toLegacyContact narrows it down to the flat shape callers expect.
export async function getContact(
  id: string | number
): Promise<Contact> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/${id}`,
    { headers: getAuthHeaders() }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  const record: ContactRecordResponse = await response.json();
  return toLegacyContact(record);
}

// Create contact
export async function createContact(
  data: Partial<Contact>
): Promise<Contact> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts`,
    {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify(toContactRecordInput(data)),
    }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  const result = await response.json();
  const record: ContactRecordResponse = result.contact || result;
  return toLegacyContact(record);
}

// Update contact
export async function updateContact(
  id: string | number,
  data: Partial<Contact>
): Promise<Contact> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/${id}`,
    {
      method: 'PUT',
      headers: getAuthHeaders(),
      body: JSON.stringify(toContactRecordInput(data)),
    }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  const record: ContactRecordResponse = await response.json();
  return toLegacyContact(record);
}

// Delete contact
export async function deleteContact(
  id: string | number
): Promise<void> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/${id}`,
    {
      method: 'DELETE',
      headers: getAuthHeaders(),
    }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }
}

// Get contact profile picture
export async function getContactProfilePicture(
  id: string | number,
  thumbnail: boolean = false
): Promise<Blob | null> {
  const url = thumbnail
    ? `${API_BASE_URL}/contacts/${id}/profile_picture?thumbnail=true`
    : `${API_BASE_URL}/contacts/${id}/profile_picture`;
  const response = await apiFetch(url);

  if (!response.ok) {
    return null;
  }

  return response.blob();
}

// Upload contact profile picture
export async function uploadProfilePicture(
  id: string | number,
  imageBlob: Blob
): Promise<void> {
  const formData = new FormData();
  formData.append('photo', imageBlob, 'profile.jpg');

  const response = await apiFetch(
    `${API_BASE_URL}/contacts/${id}/profile_picture`,
    {
      method: 'POST',
      body: formData
    }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }
}

// Get all circles
export async function getCircles(): Promise<string[]> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/circles`,
    { headers: getAuthHeaders() }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  const data = await response.json();
  // Backend returns array directly, not wrapped in object
  return Array.isArray(data) ? data : [];
}

// Get random contacts (returns 5 contacts). NOTE: unlike every other
// endpoint in this file, GetContactsRandom was deliberately left out of the
// WP-71 nested-Card API migration on the backend (see
// docs/fork-plan/50-integration-and-rebrand.md) -- it still serializes
// models.Contact's raw GORM struct directly, which is already the flat
// legacy shape (down to gorm.Model's untagged "ID" field matching this
// type's capital ID). Do NOT route this through toLegacyContact/
// ContactRecordResponse -- there is no card/crm nesting to unwrap here.
export async function getRandomContacts(): Promise<Contact[]> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/random`,
    { headers: getAuthHeaders() }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  const data = await response.json();
  return data.contacts || [];
}

// Get upcoming birthdays (returns up to 10 birthdays for contacts and relationships)
export async function getUpcomingBirthdays(): Promise<Birthday[]> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/birthdays`,
    { headers: getAuthHeaders() }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  const data = await response.json();
  return data.birthdays || [];
}

// Archive a contact (deletes all reminders). Like GetContactsRandom above,
// ArchiveContact/UnarchiveContact were deliberately left out of the WP-71
// nested-Card API migration and still return models.Contact's raw flat
// JSON directly -- no toLegacyContact translation needed or correct here.
export async function archiveContact(
  id: string | number
): Promise<Contact> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/${id}/archive`,
    {
      method: 'POST',
      headers: getAuthHeaders(),
    }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  return response.json();
}

// Unarchive a contact
export async function unarchiveContact(
  id: string | number
): Promise<Contact> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/${id}/unarchive`,
    {
      method: 'POST',
      headers: getAuthHeaders(),
    }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  return response.json();
}
