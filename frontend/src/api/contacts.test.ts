import { describe, test, expect } from 'vitest';
import {
  cardEmailsToValues,
  valuesToCardEmails,
  cardPhonesToValues,
  valuesToCardPhones,
  cardAddressesToValues,
  valuesToCardAddresses,
  getAnniversaryField,
  withAnniversary,
  getOrganizationFields,
  withOrganization,
  getTitleField,
  withTitles,
  formatAnniversaryDate,
  parseAnniversaryDate,
  toLegacyContact,
  toContactRecordInput,
  ContactRecordResponse,
} from './contacts';

describe('email conversion', () => {
  test('round-trips multiple emails with contexts', () => {
    const values = cardEmailsToValues([
      { address: 'work@example.com', contexts: ['work'] },
      { address: 'home@example.com', contexts: ['home'] },
    ]);
    expect(values).toEqual([
      { type: 'work', value: 'work@example.com' },
      { type: 'home', value: 'home@example.com' },
    ]);
    expect(valuesToCardEmails(values)).toEqual([
      { address: 'work@example.com', contexts: ['work'] },
      { address: 'home@example.com', contexts: ['home'] },
    ]);
  });

  test('drops rows with an empty value when converting back', () => {
    expect(valuesToCardEmails([{ type: 'home', value: '  ' }, { type: '', value: 'a@b.com' }])).toEqual([
      { address: 'a@b.com', contexts: undefined },
    ]);
  });

  test('handles an empty/undefined array', () => {
    expect(cardEmailsToValues(undefined)).toEqual([]);
  });
});

describe('phone conversion', () => {
  test('display prefers features over contexts (vCard feature tokens like cell/fax)', () => {
    expect(cardPhonesToValues([{ number: '555-1234', features: ['cell'], contexts: ['work'] }])).toEqual([
      { type: 'cell', value: '555-1234' },
    ]);
  });

  test('falls back to contexts when no features are set', () => {
    expect(cardPhonesToValues([{ number: '555-1234', contexts: ['work'] }])).toEqual([
      { type: 'work', value: '555-1234' },
    ]);
  });

  test('valuesToCardPhones writes the type into contexts', () => {
    expect(valuesToCardPhones([{ type: 'cell', value: '555-1234' }])).toEqual([
      { number: '555-1234', contexts: ['cell'] },
    ]);
  });
});

describe('address conversion', () => {
  test('round-trips a full address', () => {
    const card = [
      {
        contexts: ['home'],
        components: [
          { kind: 'name', value: '123 Main St' },
          { kind: 'locality', value: 'Springfield' },
          { kind: 'region', value: 'IL' },
          { kind: 'postcode', value: '62704' },
          { kind: 'country', value: 'USA' },
        ],
      },
    ];
    const values = cardAddressesToValues(card);
    expect(values).toEqual([
      { type: 'home', street: '123 Main St', city: 'Springfield', region: 'IL', postal: '62704', country: 'USA' },
    ]);
    expect(valuesToCardAddresses(values)).toEqual(card);
  });

  test('drops an address with every field blank', () => {
    expect(valuesToCardAddresses([{ type: 'home', street: '', city: '', region: '', postal: '', country: '' }])).toEqual([]);
  });
});

describe('anniversary date formatting', () => {
  test('formats a full date', () => {
    expect(formatAnniversaryDate({ partial: { year: 1990, month: 3, day: 15 } })).toBe('1990-03-15');
  });

  test('formats a year-less date', () => {
    expect(formatAnniversaryDate({ partial: { month: 3, day: 15 } })).toBe('--03-15');
  });

  test('parses both formats back losslessly', () => {
    expect(parseAnniversaryDate('1990-03-15')).toEqual({ partial: { year: 1990, month: 3, day: 15 } });
    expect(parseAnniversaryDate('--03-15')).toEqual({ partial: { month: 3, day: 15 } });
  });
});

describe('getAnniversaryField / withAnniversary', () => {
  test('reads the entry matching the requested kind only', () => {
    const anniversaries = [
      { kind: 'birth' as const, date: { partial: { year: 1990, month: 3, day: 15 } } },
      { kind: 'wedding' as const, date: { partial: { year: 2015, month: 6, day: 1 } } },
    ];
    expect(getAnniversaryField(anniversaries, 'birth')).toBe('1990-03-15');
    expect(getAnniversaryField(anniversaries, 'wedding')).toBe('2015-06-01');
  });

  test('withAnniversary replaces only the given kind, leaving the other untouched', () => {
    const anniversaries = [
      { kind: 'birth' as const, date: { partial: { year: 1990, month: 3, day: 15 } } },
      { kind: 'wedding' as const, date: { partial: { year: 2015, month: 6, day: 1 } } },
    ];
    const updated = withAnniversary(anniversaries, 'birth', '1991-04-16');
    expect(getAnniversaryField(updated, 'birth')).toBe('1991-04-16');
    expect(getAnniversaryField(updated, 'wedding')).toBe('2015-06-01');
  });

  test('withAnniversary drops the entry when given an empty value', () => {
    const anniversaries = [{ kind: 'birth' as const, date: { partial: { year: 1990, month: 3, day: 15 } } }];
    expect(withAnniversary(anniversaries, 'birth', '')).toEqual([]);
  });
});

describe('organization fields', () => {
  test('getOrganizationFields reads name + first unit as department', () => {
    expect(getOrganizationFields([{ name: 'Acme', units: [{ name: 'R&D' }] }])).toEqual({
      organization: 'Acme',
      department: 'R&D',
    });
  });

  test('withOrganization preserves department when only organization changes', () => {
    // Simulates ContactDetailPage's buildRecordPatch: read the current
    // department, then patch organization while passing it back through.
    const current = getOrganizationFields([{ name: 'Acme', units: [{ name: 'R&D' }] }]);
    const updated = withOrganization('Globex', current.department || '');
    expect(getOrganizationFields(updated)).toEqual({ organization: 'Globex', department: 'R&D' });
  });

  test('withOrganization returns an empty array when organization is blank', () => {
    expect(withOrganization('', 'R&D')).toEqual([]);
  });
});

describe('title fields', () => {
  test('getTitleField distinguishes title from role by kind', () => {
    const titles = [{ name: 'Engineer', kind: 'title' as const }, { name: 'Lead', kind: 'role' as const }];
    expect(getTitleField(titles, 'title')).toBe('Engineer');
    expect(getTitleField(titles, 'role')).toBe('Lead');
  });

  test('withTitles preserves role when only job title changes', () => {
    const current = { role: getTitleField([{ name: 'Lead', kind: 'role' }], 'role') };
    const updated = withTitles('Senior Engineer', current.role || '');
    expect(getTitleField(updated, 'title')).toBe('Senior Engineer');
    expect(getTitleField(updated, 'role')).toBe('Lead');
  });
});

describe('toLegacyContact / toContactRecordInput round-trip', () => {
  const record: ContactRecordResponse = {
    id: 42,
    uid: 'uid-42',
    etag: 'etag-1',
    gender: 'other',
    card: {
      name: {
        components: [
          { kind: 'title', value: 'Dr.' },
          { kind: 'given', value: 'Marie' },
          { kind: 'given2', value: 'Salomea' },
          { kind: 'surname', value: 'Curie' },
        ],
      },
      nicknames: [{ name: 'Manya' }],
      emails: [{ address: 'marie@sorbonne.fr', contexts: ['work'] }],
      phones: [{ number: '555-0100', features: ['cell'] }],
      organizations: [{ name: 'Sorbonne University', units: [{ name: 'Physics' }] }],
      titles: [{ name: 'Professor', kind: 'title' }, { name: 'Nobel Laureate', kind: 'role' }],
      anniversaries: [{ kind: 'birth', date: { partial: { year: 1867, month: 11, day: 7 } } }],
    },
    crm: {
      circles: ['Scientists'],
      how_we_met: 'Conference',
      custom_fields: { favorite_color: 'Radium Green' },
    },
    photo: '',
    photo_thumbnail: '',
    archived: false,
  };

  test('toLegacyContact flattens every field correctly', () => {
    const contact = toLegacyContact(record);
    expect(contact.firstname).toBe('Marie');
    expect(contact.lastname).toBe('Curie');
    expect(contact.prefix).toBe('Dr.');
    expect(contact.middle_name).toBe('Salomea');
    expect(contact.nickname).toBe('Manya');
    expect(contact.emails).toEqual([{ type: 'work', value: 'marie@sorbonne.fr' }]);
    expect(contact.phones).toEqual([{ type: 'cell', value: '555-0100' }]);
    expect(contact.organization).toBe('Sorbonne University');
    expect(contact.department).toBe('Physics');
    expect(contact.job_title).toBe('Professor');
    expect(contact.role).toBe('Nobel Laureate');
    expect(contact.birthday).toBe('1867-11-07');
    expect(contact.circles).toEqual(['Scientists']);
    expect(contact.custom_fields).toEqual({ favorite_color: 'Radium Green' });
  });

  test('toContactRecordInput rebuilds an equivalent nested shape from the flattened contact', () => {
    const contact = toLegacyContact(record);
    const input = toContactRecordInput(contact);
    expect(input.gender).toBe('other');
    expect(input.card.emails).toEqual(record.card.emails);
    expect(input.card.phones).toEqual([{ number: '555-0100', contexts: ['cell'] }]);
    expect(input.card.organizations).toEqual(record.card.organizations);
    expect(input.card.titles).toEqual(record.card.titles);
    expect(input.card.anniversaries).toEqual(record.card.anniversaries);
    expect(input.crm.circles).toEqual(['Scientists']);
    expect(input.crm.custom_fields).toEqual({ favorite_color: 'Radium Green' });
  });
});
