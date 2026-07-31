import { test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, cleanup, fireEvent, waitFor } from '@testing-library/react';
import '../i18n/config';
import MergeContactsDialog from './MergeContactsDialog';
import { SnackbarProvider } from '../context/SnackbarContext';

afterEach(cleanup);

function renderDialog(props: Partial<React.ComponentProps<typeof MergeContactsDialog>> = {}) {
  const defaults: React.ComponentProps<typeof MergeContactsDialog> = {
    open: true,
    onClose: vi.fn(),
    onMerged: vi.fn(),
    currentContactId: 1,
    currentContactUid: 'alice-uid',
    currentContactName: 'Alice Anderson',
    ...props,
  };
  return render(
    <SnackbarProvider>
      <MergeContactsDialog {...defaults} />
    </SnackbarProvider>
  );
}

function mockFetchByUrl(handlers: Record<string, () => unknown>) {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (url: string) => {
      for (const [pattern, respond] of Object.entries(handlers)) {
        if (url.includes(pattern)) {
          return { ok: true, json: async () => respond() };
        }
      }
      throw new Error(`unexpected fetch: ${url}`);
    })
  );
}

const bobContactsResponse = () => ({
  contacts: [{ id: 2, uid: 'bob-uid', firstname: 'Bob', lastname: 'Brown' }],
  total: 1,
  page: 1,
  limit: 100,
});

const emptyAssociationCounts = {
  notes: 0, activities: 0, reminders: 0, reminder_completions: 0, relationship_edges: 0,
  household_memberships: 0, circle_memberships: 0, tags: 0, life_events: 0,
  life_event_references: 0, field_values: 0, contact_sync_links: 0,
};

async function selectBob() {
  const input = await screen.findByLabelText('Merge into');
  fireEvent.change(input, { target: { value: 'Bob' } });
  const option = await screen.findByText('Bob Brown');
  fireEvent.click(option);
}

beforeEach(() => {
  mockFetchByUrl({ '/contacts?': bobContactsResponse });
});

afterEach(() => {
  vi.unstubAllGlobals();
});

test('renders the title and target-contact picker', async () => {
  renderDialog();

  expect(screen.getByText('Merge Contacts')).toBeInTheDocument();
  await waitFor(() => expect(fetch).toHaveBeenCalled());
});

test('merge button is disabled until a target contact is selected', async () => {
  renderDialog();
  await waitFor(() => expect(fetch).toHaveBeenCalled());

  expect(screen.getByRole('button', { name: 'Merge' })).toBeDisabled();
});

test('selecting a target with no conflicts shows the no-conflicts message and enables merge', async () => {
  mockFetchByUrl({
    '/contacts?': bobContactsResponse,
    '/contacts/merge/preview': () => ({
      keep_id: 2,
      merge_id: 1,
      resolution: {
        emails: [], phones: [], addresses: [], urls: [], impps: [], circles: [],
        custom_fields: {}, resolved_scalars: {}, conflicts: [], field_value_conflicts: [],
      },
      association_counts: emptyAssociationCounts,
    }),
  });

  renderDialog();
  await selectBob();

  await screen.findByText('No conflicts to resolve. Ready to merge.');
  expect(screen.getByRole('button', { name: 'Merge' })).not.toBeDisabled();
});

test('handles a real backend response where empty Go slices serialize as null, not []', async () => {
  // Regression test: found by real-browser verification, not by the mocked
  // tests above (which all hand-wrote `[]`). A Go nil slice marshals to
  // JSON `null`, not `[]` -- omitempty does not change this. The preview
  // response's conflicts/field_value_conflicts (and every other slice/map
  // field) come back `null` whenever empty, and the dialog must not crash
  // spreading/mapping over them.
  mockFetchByUrl({
    '/contacts?': bobContactsResponse,
    '/contacts/merge/preview': () => ({
      keep_id: 2,
      merge_id: 1,
      resolution: {
        emails: null, phones: null, addresses: null, urls: null, impps: null, circles: null,
        custom_fields: null, resolved_scalars: null, conflicts: null, field_value_conflicts: null,
      },
      association_counts: emptyAssociationCounts,
    }),
  });

  renderDialog();
  await selectBob();

  await screen.findByText('No conflicts to resolve. Ready to merge.');
  expect(screen.getByRole('button', { name: 'Merge' })).not.toBeDisabled();
});

test('a scalar conflict keeps merge disabled until resolved', async () => {
  mockFetchByUrl({
    '/contacts?': bobContactsResponse,
    '/contacts/merge/preview': () => ({
      keep_id: 2,
      merge_id: 1,
      resolution: {
        emails: [], phones: [], addresses: [], urls: [], impps: [], circles: [],
        custom_fields: {}, resolved_scalars: {},
        conflicts: [{ field: 'firstname', label: 'First Name', keeper_value: 'Bob', loser_value: 'Robert' }],
        field_value_conflicts: [],
      },
      association_counts: emptyAssociationCounts,
    }),
  });

  renderDialog();
  await selectBob();

  await screen.findByText('First Name');
  const mergeButton = screen.getByRole('button', { name: 'Merge' });
  expect(mergeButton).toBeDisabled();

  fireEvent.click(screen.getByLabelText('Bob: Bob'));
  await waitFor(() => expect(mergeButton).not.toBeDisabled());
});
