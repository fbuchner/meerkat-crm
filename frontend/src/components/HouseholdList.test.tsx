import { test, expect, vi, afterEach } from 'vitest';
import { render, screen, cleanup, fireEvent } from '@testing-library/react';
import '../i18n/config';
import HouseholdList from './HouseholdList';
import { Household, HouseholdMember } from '../api/households';
import { Contact } from '../api/contacts';

afterEach(cleanup);

const household: Household = {
  id: 'h1',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  name: 'Smith Family',
  type: 'family_unit',
};

const members: HouseholdMember[] = [
  { id: 1, household_id: 'h1', member_vcard_uid: 'alice-uid', role: 'head' },
  { id: 2, household_id: 'h1', member_vcard_uid: 'bob-uid', role: 'child' },
];

const contactsByUid = new Map<string, Contact>([
  ['alice-uid', { ID: 1, uid: 'alice-uid', firstname: 'Alice', lastname: 'Anderson' }],
  ['bob-uid', { ID: 2, uid: 'bob-uid', firstname: 'Bob', lastname: 'Brown' }],
]);

function renderList(props: Partial<React.ComponentProps<typeof HouseholdList>> = {}) {
  const defaults: React.ComponentProps<typeof HouseholdList> = {
    households: [household],
    members,
    contactsByUid,
    suggestPendingId: null,
    onEdit: vi.fn(),
    onDelete: vi.fn(),
    onSuggest: vi.fn(),
    onAddMember: vi.fn(),
    onRemoveMember: vi.fn(),
    onRoleChange: vi.fn(),
    ...props,
  };
  return render(<HouseholdList {...defaults} />);
}

test('renders household name, type chip, and resolved member names', () => {
  renderList();

  expect(screen.getByText('Smith Family')).toBeInTheDocument();
  expect(screen.getByText('Family unit')).toBeInTheDocument();
  expect(screen.getByText('Alice Anderson')).toBeInTheDocument();
  expect(screen.getByText('Bob Brown')).toBeInTheDocument();
});

test('the suggest button calls onSuggest with the household', () => {
  const onSuggest = vi.fn();
  renderList({ onSuggest });

  fireEvent.click(screen.getByRole('button', { name: 'Suggest relationships' }));
  expect(onSuggest).toHaveBeenCalledWith(household);
});

test('the suggest button is disabled for a single-member household', () => {
  renderList({ members: [members[0]] });
  expect(screen.getByRole('button', { name: 'Suggest relationships' })).toBeDisabled();
});

test('remove member calls onRemoveMember with the household and vcard uid', () => {
  const onRemoveMember = vi.fn();
  renderList({ onRemoveMember });

  const removeButtons = screen.getAllByLabelText('Remove Member');
  fireEvent.click(removeButtons[0]);
  expect(onRemoveMember).toHaveBeenCalledWith('h1', 'alice-uid');
});

test('Add Member reveals the contact search autocomplete', () => {
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ contacts: [], total: 0, page: 1, limit: 40 }),
    })
  );

  renderList();
  fireEvent.click(screen.getByRole('button', { name: 'Add Member' }));

  expect(screen.getByPlaceholderText('Search contacts…')).toBeInTheDocument();

  vi.unstubAllGlobals();
});

test('changing a member role calls onRoleChange', async () => {
  const onRoleChange = vi.fn();
  renderList({ onRoleChange });

  // Two member rows each have a Role select; the add-member form is closed.
  const roleSelects = screen.getAllByLabelText('Role');
  fireEvent.mouseDown(roleSelects[0]);
  fireEvent.click(await screen.findByRole('option', { name: 'Child' }));

  expect(onRoleChange).toHaveBeenCalledWith('h1', members[0], 'child');
});

test('shows the empty state when there are no households', () => {
  renderList({ households: [], members: [] });
  expect(screen.getByText('No households yet. Create one to start suggesting relationships between the people who live together.')).toBeInTheDocument();
});
