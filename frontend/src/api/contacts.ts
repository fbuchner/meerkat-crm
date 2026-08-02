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
  // passthrough preserves address components whose kind is not one of the five
  // rendered fields above (apartment, floor, district, building, room,
  // landmark, etc.) so they survive an edit-and-save cycle through the flat
  // editing shape (T25).
  passthrough?: CardAddressComponent[];
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
  // Contact.VCardUID -- surfaced for callers that need the stable UUID
  // rather than the numeric ID (e.g. RelationshipEdge.SourceID/TargetID,
  // api/relationshipEdges.ts). Already fetched server-side on every summary
  // response; just not previously exposed on this shape.
  uid?: string;
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
  work_information?: string;
  contact_information?: string;
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

export interface NameComponent {
  kind: 'title' | 'given' | 'given2' | 'surname' | 'surname2' | 'credential' | 'generation' | 'separator';
  value: string;
}

export interface CardName {
  components?: NameComponent[];
  full?: string;
}

export interface CardNickname {
  name: string;
}

export interface CardOrgUnit {
  name: string;
}

export interface CardOrganization {
  id?: string;
  name?: string;
  units?: CardOrgUnit[];
}

export interface CardTitle {
  name: string;
  kind?: 'title' | 'role';
}

export interface CardEmail {
  address: string;
  contexts?: string[];
}

export interface CardPhone {
  number: string;
  features?: string[];
  contexts?: string[];
}

export interface CardOnlineService {
  uri?: string;
  service?: string;
  contexts?: string[];
}

export interface CardResource {
  uri: string;
  kind?: string;
  mediaType?: string;
  contexts?: string[];
}

export interface CardAddressComponent {
  kind: string;
  value: string;
}

export interface CardAddress {
  components?: CardAddressComponent[];
  countryCode?: string;
  contexts?: string[];
}

export interface CardPartialDate {
  year?: number | null;
  month?: number | null;
  day?: number | null;
}

export interface CardAnniversaryDate {
  partial?: CardPartialDate;
  timestamp?: string | null;
}

export interface CardAnniversary {
  kind: 'birth' | 'death' | 'wedding';
  date: CardAnniversaryDate;
}

export interface Card {
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

export interface CRMEnvelope {
  how_we_met?: string;
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
  archived: boolean;
}

// ---------------------------------------------------------------------------
// Adapter: nested wire shape <-> legacy flat Contact shape.
// ---------------------------------------------------------------------------

export function nameComponentValue(components: NameComponent[] | undefined, kind: NameComponent['kind']): string | undefined {
  return components?.find((c) => c.kind === kind)?.value;
}

// formatAnniversaryDate turns a CardAnniversaryDate into the ISO YYYY-MM-DD
// / year-less --MM-DD string convention used throughout this app for
// birthday/anniversary fields (never a full RFC3339 timestamp for a
// date-only field).
export function formatAnniversaryDate(date: CardAnniversaryDate | undefined): string | undefined {
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
export function parseAnniversaryDate(value: string): CardAnniversaryDate {
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

// ---------------------------------------------------------------------------
// Per-concept Card <-> ContactValue/ContactAddress helpers. Shared by
// toLegacyContact/toContactRecordInput below AND by components that have
// migrated to reading/writing Card/CRMEnvelope directly (ContactInformation)
// -- a single source of truth for the mapping rules so a migrated
// component's edits can never drift from what the shim itself would produce.
// ---------------------------------------------------------------------------

export function cardEmailsToValues(emails: CardEmail[] | undefined): ContactValue[] {
  return (emails || []).map((e) => ({ type: e.contexts?.[0] || '', value: e.address }));
}
export function valuesToCardEmails(values: ContactValue[]): CardEmail[] {
  return values.filter((e) => e.value.trim()).map((e) => ({ address: e.value, contexts: e.type ? [e.type] : undefined }));
}

export function cardPhonesToValues(phones: CardPhone[] | undefined): ContactValue[] {
  return (phones || []).map((p) => ({ type: p.features?.[0] || p.contexts?.[0] || '', value: p.number }));
}
export function valuesToCardPhones(values: ContactValue[]): CardPhone[] {
  return values.filter((p) => p.value.trim()).map((p) => ({ number: p.value, contexts: p.type ? [p.type] : undefined }));
}

export function cardLinksToValues(links: CardResource[] | undefined): ContactValue[] {
  return (links || []).map((l) => ({ type: l.contexts?.[0] || '', value: l.uri }));
}
export function valuesToCardLinks(values: ContactValue[]): CardResource[] {
  return values.filter((u) => u.value.trim()).map((u) => ({ uri: u.value, contexts: u.type ? [u.type] : undefined }));
}

export function cardImppToValues(impps: CardOnlineService[] | undefined): ContactValue[] {
  return (impps || []).map((i) => ({ type: i.contexts?.[0] || '', value: i.uri || '' }));
}
export function valuesToCardImpp(values: ContactValue[]): CardOnlineService[] {
  return values.filter((i) => i.value.trim()).map((i) => ({ uri: i.value, contexts: i.type ? [i.type] : undefined }));
}

export function cardAddressesToValues(addresses: CardAddress[] | undefined): ContactAddress[] {
  return (addresses || []).map((a) => {
    const comps = a.components || [];
    const find = (kind: string) => comps.find((c) => c.kind === kind)?.value || '';
    // Preserve components not mapped to the five rendered fields so they
    // survive an edit-and-save round trip (T25).
    const knownKinds = new Set(['name', 'number', 'locality', 'region', 'postcode', 'country']);
    const passthrough = comps.filter((c) => !knownKinds.has(c.kind));
    return {
      type: a.contexts?.[0] || '',
      street: find('name') || find('number'),
      city: find('locality'),
      region: find('region'),
      postal: find('postcode'),
      country: find('country') || a.countryCode || '',
      passthrough: passthrough.length > 0 ? passthrough : undefined,
    };
  });
}
export function valuesToCardAddresses(values: ContactAddress[]): CardAddress[] {
  return values
    .filter((a) => a.street.trim() || a.city.trim() || a.region.trim() || a.postal.trim() || a.country.trim())
    .map((a) => {
      const components: CardAddressComponent[] = [];
      if (a.street) components.push({ kind: 'name', value: a.street });
      if (a.city) components.push({ kind: 'locality', value: a.city });
      if (a.region) components.push({ kind: 'region', value: a.region });
      if (a.postal) components.push({ kind: 'postcode', value: a.postal });
      if (a.country) components.push({ kind: 'country', value: a.country });
      // Re-emit passthrough components that were preserved from the original address
      if (a.passthrough) components.push(...a.passthrough);
      return { components, contexts: a.type ? [a.type] : undefined };
    });
}

export function getAnniversaryField(anniversaries: CardAnniversary[] | undefined, kind: 'birth' | 'wedding'): string | undefined {
  return formatAnniversaryDate((anniversaries || []).find((a) => a.kind === kind)?.date);
}
// withAnniversary replaces the single entry of `kind` (if any) with one
// parsed from `value`, or drops it entirely when `value` is empty -- the
// other kind (birth vs. wedding) is left untouched.
export function withAnniversary(
  anniversaries: CardAnniversary[] | undefined,
  kind: 'birth' | 'wedding',
  value: string
): CardAnniversary[] {
  const rest = (anniversaries || []).filter((a) => a.kind !== kind);
  return value ? [...rest, { kind, date: parseAnniversaryDate(value) }] : rest;
}

// Only the first organization/title entries are surfaced -- this app's UI
// (like the legacy Contact shape it grew from) only ever edits one
// organization and one title/role pair, even though the nested model
// supports arrays of both.
export function getOrganizationFields(organizations: CardOrganization[] | undefined): { organization?: string; department?: string } {
  const org = organizations?.[0];
  return { organization: org?.name, department: org?.units?.[0]?.name };
}
export function withOrganization(organization: string, department: string): CardOrganization[] {
  return organization ? [{ name: organization, units: department ? [{ name: department }] : undefined }] : [];
}

export function getTitleField(titles: CardTitle[] | undefined, kind: 'title' | 'role'): string | undefined {
  if (kind === 'title') return titles?.find((t) => t.kind === 'title' || !t.kind)?.name;
  return titles?.find((t) => t.kind === 'role')?.name;
}
export function withTitles(jobTitle: string, role: string): CardTitle[] {
  const titles: CardTitle[] = [];
  if (jobTitle) titles.push({ name: jobTitle, kind: 'title' });
  if (role) titles.push({ name: role, kind: 'role' });
  return titles;
}

// summaryToLegacyContact maps the slim GET /contacts list item shape down
// into the same flat Contact shape ContactsPage/DashboardPage etc. render.
function summaryToLegacyContact(summary: ContactSummaryDTO): Contact {
  return {
    ID: summary.id,
    uid: summary.uid,
    firstname: summary.firstname,
    lastname: summary.lastname,
    nickname: summary.nickname || undefined,
    email: summary.primary_email || undefined,
    phone: summary.primary_phone || undefined,
    birthday: summary.birthday || undefined,
    photo: summary.photo || undefined,
    photo_thumbnail: summary.photo_thumbnail || undefined,
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

  const emailValues = data.emails && data.emails.length > 0 ? data.emails : data.email ? [{ type: '', value: data.email }] : [];
  const phoneValues = data.phones && data.phones.length > 0 ? data.phones : data.phone ? [{ type: '', value: data.phone }] : [];
  const addressValues = data.addresses && data.addresses.length > 0
    ? data.addresses
    : data.address
      ? [{ type: '', street: data.address, city: '', region: '', postal: '', country: '' }]
      : [];

  const emails = valuesToCardEmails(emailValues);
  const phones = valuesToCardPhones(phoneValues);
  const links = valuesToCardLinks(data.urls || []);
  const imppAddresses = valuesToCardImpp(data.impps || []);
  const addresses = valuesToCardAddresses(addressValues);

  const anniversaries = withAnniversary(
    withAnniversary(undefined, 'birth', data.birthday || ''),
    'wedding',
    data.anniversary || ''
  );

  const organizations = withOrganization(data.organization || '', data.department || '');
  const titles = withTitles(data.job_title || '', data.role || '');

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
      how_we_met: data.how_we_met,
      work_information: data.work_information,
      contact_information: data.contact_information,
      custom_fields: data.custom_fields,
    },
  };
}

export interface Birthday {
  type: 'contact';
  name: string;
  birthday: string;
  photo_thumbnail?: string;
  contact_id: number;
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

// Resolves a batch of Contact.VCardUID values to full Contact objects in one
// request, via GET /contacts' ?vcard_uid= filter (§3d WP0). Needed because
// RelationshipEdge.SourceID/TargetID (api/relationshipEdges.ts) are bare
// VCardUID strings with no nested contact data. includeArchived: true so an
// edge pointing at a since-archived contact still resolves instead of
// silently vanishing (GetContacts excludes archived by default).
export async function getContactsByUid(uids: string[]): Promise<Map<string, Contact>> {
  const wanted = uids.filter(Boolean);
  const map = new Map<string, Contact>();
  if (wanted.length === 0) return map;

  const queryParams = new URLSearchParams({ include_archived: 'true' });
  wanted.forEach((uid) => queryParams.append('vcard_uid', uid));

  const response = await apiFetch(`${API_BASE_URL}/contacts?${queryParams.toString()}`, { headers: getAuthHeaders() });
  if (!response.ok) throw await parseErrorResponse(response);

  const data: { contacts: ContactSummaryDTO[] } = await response.json();
  for (const dto of data.contacts) {
    map.set(dto.uid, summaryToLegacyContact(dto));
  }
  return map;
}

// getContactRecord/updateContactRecord/createContactRecord read and write
// Card/CRMEnvelope directly. Every contact-editing component has migrated
// onto these (see docs/fork-plan/95, Tier 0 items 3-6) -- toContactRecordInput
// (below) still exists for the e2e test fixtures' convenience, but nothing
// in the app itself round-trips a full record through the flat Contact shape
// anymore.
export async function getContactRecord(id: string | number): Promise<ContactRecordResponse> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/${id}`,
    { headers: getAuthHeaders() }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  return response.json();
}

export async function updateContactRecord(id: string | number, input: ContactRecordInput): Promise<ContactRecordResponse> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/${id}`,
    {
      method: 'PUT',
      headers: getAuthHeaders(),
      body: JSON.stringify(input),
    }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  return response.json();
}

export async function createContactRecord(input: ContactRecordInput): Promise<ContactRecordResponse> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts`,
    {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify(input),
    }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  const result = await response.json();
  return result.contact || result;
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

// Temporary: reads legacy strings from the old flat Contact.Circles JSON
// column. Used by the T2 triage page during migration. Remove after migration.
export async function getLegacyCircles(): Promise<string[]> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/circles?legacy=true`,
    { headers: getAuthHeaders() }
  );
  if (!response.ok) throw await parseErrorResponse(response);
  const data = await response.json();
  return Array.isArray(data) ? data : [];
}

// Temporary: filters contacts by a legacy flat-circle string. Used by the
// T2 triage page's member-add step. Remove after migration.
export async function getContactsByLegacyCircle(circle: string): Promise<{ contacts: Contact[]; total: number }> {
  const queryParams = new URLSearchParams({
    page: '1', limit: '500', circle_legacy: circle,
  });
  const response = await apiFetch(
    `${API_BASE_URL}/contacts?${queryParams.toString()}`,
    { headers: getAuthHeaders() }
  );
  if (!response.ok) throw await parseErrorResponse(response);
  return response.json();
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

// Get upcoming birthdays (returns up to 10 birthdays for contacts)
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
