import { test, expect, vi, afterEach, beforeEach } from 'vitest';
import { render, screen, cleanup, waitFor } from '@testing-library/react';
import '../i18n/config';
import EditTimelineItemDialog from './EditTimelineItemDialog';
import { SnackbarProvider } from '../context/SnackbarContext';
import { DateFormatProvider } from '../DateFormatProvider';

beforeEach(() => {
  localStorage.setItem('user_info', JSON.stringify({ user_id: 1, username: 'test', is_admin: false }));
});

afterEach(() => {
  cleanup();
  localStorage.clear();
  vi.unstubAllGlobals();
});

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

const contactsResponse = () => ({
  contacts: [
    { ID: 5, uid: 'uid-5', firstname: 'Alice', lastname: 'Anderson' },
    { ID: 6, uid: 'uid-6', firstname: 'Bob', lastname: 'Brown' },
  ],
  total: 2,
  page: 1,
  limit: 40,
});

// T6: When opened in note mode with an already-assigned contact, the dialog
// shows the contact name in a Chip and the Autocomplete is visible for
// changing the assignment.
test('renders assigned contact name when opened with noteContactId and noteContactName', async () => {
  mockFetchByUrl({ '/contacts?': contactsResponse });

  render(
    <SnackbarProvider>
      <DateFormatProvider>
        <EditTimelineItemDialog
          open={true}
          onClose={vi.fn()}
          onSave={vi.fn()}
          onDelete={vi.fn()}
          type="note"
          values={{
            noteContent: 'Test note content',
            noteDate: '2026-01-01',
            noteContactId: 5,
            noteContactName: 'Alice Anderson',
          }}
          onChange={vi.fn()}
          allContacts={[]}
        />
      </DateFormatProvider>
    </SnackbarProvider>
  );

  await waitFor(() => {
    expect(screen.getByText('Assigned to Alice Anderson')).toBeDefined();
  });

  // The content and date fields are present
  expect(screen.getByDisplayValue('Test note content')).toBeDefined();
  expect(screen.getByDisplayValue('2026-01-01')).toBeDefined();
});

// T6b: When the contact has no assignment, the dialog shows no Chip and
// the Autocomplete placeholder is visible.
test('shows no contact chip when note has no assignment', async () => {
  mockFetchByUrl({ '/contacts?': contactsResponse });

  render(
    <SnackbarProvider>
      <DateFormatProvider>
        <EditTimelineItemDialog
          open={true}
          onClose={vi.fn()}
          onSave={vi.fn()}
          onDelete={vi.fn()}
          type="note"
          values={{
            noteContent: 'No contact',
            noteDate: '2026-01-01',
          }}
          onChange={vi.fn()}
          allContacts={[]}
        />
      </DateFormatProvider>
    </SnackbarProvider>
  );

  await waitFor(() => {
    expect(screen.queryByText('Assigned to')).toBeNull();
  });

  // The assign-to-contact Autocomplete placeholder is present
  expect(screen.getByPlaceholderText('Search contacts...')).toBeDefined();
});
