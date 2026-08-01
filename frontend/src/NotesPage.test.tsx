import { test, expect, vi, afterEach, beforeEach } from 'vitest';
import { render, screen, cleanup, waitFor } from '@testing-library/react';
import './i18n/config';
import NotesPage from './NotesPage';
import { SnackbarProvider } from './context/SnackbarContext';
import { DateFormatProvider } from './DateFormatProvider';

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

const emptyNotesResponse = () => ({
  notes: [],
  total: 0,
  page: 1,
  limit: 25,
});

const twoUnfiledNotesResponse = () => ({
  notes: [
    { ID: 1, content: 'First note', date: '2026-01-01T00:00:00Z', contact_id: null, CreatedAt: '2026-01-01T00:00:00Z', UpdatedAt: '2026-01-01T00:00:00Z' },
    { ID: 2, content: 'Second note', date: '2026-01-02T00:00:00Z', contact_id: null, CreatedAt: '2026-01-02T00:00:00Z', UpdatedAt: '2026-01-02T00:00:00Z' },
  ],
  total: 2,
  page: 1,
  limit: 25,
});

function renderPage() {
  return render(
    <SnackbarProvider>
      <DateFormatProvider>
        <NotesPage />
      </DateFormatProvider>
    </SnackbarProvider>
  );
}

test('renders empty inbox when no notes exist', async () => {
  mockFetchByUrl({ '/notes?': emptyNotesResponse });
  renderPage();

  await waitFor(() => {
    expect(screen.getByText('Inbox')).toBeDefined();
  });

  expect(screen.getByText('0')).toBeDefined();
  expect(screen.getByText('No unfiled notes')).toBeDefined();
});

test('renders notes list and shows unfiled count', async () => {
  mockFetchByUrl({ '/notes?': twoUnfiledNotesResponse });
  renderPage();

  await waitFor(() => {
    expect(screen.getByText('First note')).toBeDefined();
  });

  expect(screen.getByText('Second note')).toBeDefined();
  expect(screen.getByText('2')).toBeDefined();
});
