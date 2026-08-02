import { test, expect, vi, afterEach, beforeEach } from 'vitest';
import { render, screen, cleanup, waitFor } from '@testing-library/react';
import '../i18n/config';
import AddNoteDialog from './AddNoteDialog';
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

// T7: AddNoteDialog renders the contact Autocomplete for optional assignment
// at creation time.
test('renders contact autocomplete for assigning a contact at creation', async () => {
  mockFetchByUrl({ '/contacts?': contactsResponse });

  render(
    <SnackbarProvider>
      <DateFormatProvider>
        <AddNoteDialog
          open={true}
          onClose={vi.fn()}
          onSave={vi.fn().mockResolvedValue(undefined)}
        />
      </DateFormatProvider>
    </SnackbarProvider>
  );

  // The contact Autocomplete placeholder is rendered
  await waitFor(() => {
    expect(screen.getByPlaceholderText('Search contacts...')).toBeDefined();
  });

  // Content and date fields exist
  expect(screen.getByLabelText('Content *')).toBeDefined();
  expect(screen.getByLabelText('Date')).toBeDefined();

  // Save and cancel buttons exist
  expect(screen.getByText('Save')).toBeDefined();
  expect(screen.getByText('Cancel')).toBeDefined();
});

// T7b: Dialog title is present.
test('renders the dialog title', async () => {
  mockFetchByUrl({ '/contacts?': contactsResponse });

  render(
    <SnackbarProvider>
      <DateFormatProvider>
        <AddNoteDialog
          open={true}
          onClose={vi.fn()}
          onSave={vi.fn().mockResolvedValue(undefined)}
        />
      </DateFormatProvider>
    </SnackbarProvider>
  );

  await waitFor(() => {
    expect(screen.getByText('Add Note')).toBeDefined();
  });
});
