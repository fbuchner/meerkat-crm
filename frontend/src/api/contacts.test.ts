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
  toContactRecordInput,
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

describe('toContactRecordInput', () => {
  // toLegacyContact/getContact/createContact/updateContact were retired once
  // every contact-editing component migrated onto getContactRecord/
  // updateContactRecord/createContactRecord (docs/fork-plan/95, Tier 0 items
  // 3-7) -- toContactRecordInput itself survives only for e2e test fixtures
  // (e2e/fixtures.ts, e2e/global-setup.ts), which still find it convenient
  // to build nested payloads from simple flat test data.
  test('builds an equivalent nested shape from a flat Contact-like input', () => {
    const input = toContactRecordInput({
      firstname: 'Marie',
      lastname: 'Curie',
      prefix: 'Dr.',
      middle_name: 'Salomea',
      nickname: 'Manya',
      gender: 'other',
      emails: [{ type: 'work', value: 'marie@sorbonne.fr' }],
      phones: [{ type: 'cell', value: '555-0100' }],
      organization: 'Sorbonne University',
      department: 'Physics',
      job_title: 'Professor',
      role: 'Nobel Laureate',
      birthday: '1867-11-07',
      circles: ['Scientists'],
      custom_fields: { favorite_color: 'Radium Green' },
    });

    expect(input.gender).toBe('other');
    expect(input.card.name?.components).toEqual([
      { kind: 'title', value: 'Dr.' },
      { kind: 'given', value: 'Marie' },
      { kind: 'given2', value: 'Salomea' },
      { kind: 'surname', value: 'Curie' },
    ]);
    expect(input.card.nicknames).toEqual([{ name: 'Manya' }]);
    expect(input.card.emails).toEqual([{ address: 'marie@sorbonne.fr', contexts: ['work'] }]);
    expect(input.card.phones).toEqual([{ number: '555-0100', contexts: ['cell'] }]);
    expect(input.card.organizations).toEqual([{ name: 'Sorbonne University', units: [{ name: 'Physics' }] }]);
    expect(input.card.titles).toEqual([{ name: 'Professor', kind: 'title' }, { name: 'Nobel Laureate', kind: 'role' }]);
    expect(input.card.anniversaries).toEqual([{ kind: 'birth', date: { partial: { year: 1867, month: 11, day: 7 } } }]);
    expect(input.crm.circles).toEqual(['Scientists']);
    expect(input.crm.custom_fields).toEqual({ favorite_color: 'Radium Green' });
  });
});
