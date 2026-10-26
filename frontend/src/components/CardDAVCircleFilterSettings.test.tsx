import { test, expect, vi, afterEach, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import i18n from '../i18n/config';
import CardDAVCircleFilterSettings from './CardDAVCircleFilterSettings';
import { getCardDAVCircleFilter, updateCardDAVCircleFilter } from '../api/users';
import { getCircles } from '../api/contacts';

vi.mock('../api/users', () => ({
  getCardDAVCircleFilter: vi.fn(),
  updateCardDAVCircleFilter: vi.fn(),
}));
vi.mock('../api/contacts', () => ({
  getCircles: vi.fn(),
}));

beforeEach(() => {
  vi.mocked(getCircles).mockResolvedValue(['Family', 'Work']);
  vi.mocked(updateCardDAVCircleFilter).mockImplementation(async (filter) => filter);
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function renderSettings() {
  return render(
    <I18nextProvider i18n={i18n}>
      <CardDAVCircleFilterSettings />
    </I18nextProvider>
  );
}

test('loads the stored filter and saves it unchanged', async () => {
  vi.mocked(getCardDAVCircleFilter).mockResolvedValue({ mode: 'include', circles: ['Family'] });
  renderSettings();

  expect(await screen.findByText('Family')).toBeInTheDocument();
  expect(screen.getByRole('radio', { name: /only sync contacts in these circles/i })).toBeChecked();

  fireEvent.click(screen.getByRole('button', { name: /save filter/i }));
  await waitFor(() =>
    expect(updateCardDAVCircleFilter).toHaveBeenCalledWith({ mode: 'include', circles: ['Family'] })
  );
});

test('a filter mode cannot be saved without circles', async () => {
  vi.mocked(getCardDAVCircleFilter).mockResolvedValue({ mode: 'all', circles: [] });
  renderSettings();

  fireEvent.click(await screen.findByRole('radio', { name: /except those in these circles/i }));

  expect(screen.getByText(/select at least one circle/i)).toBeInTheDocument();
  expect(screen.getByRole('button', { name: /save filter/i })).toBeDisabled();
});

test('a failed filter load hides the form so the default cannot be saved over the real filter', async () => {
  vi.mocked(getCardDAVCircleFilter).mockRejectedValue(new Error('network down'));
  renderSettings();

  expect(await screen.findByText('network down')).toBeInTheDocument();
  expect(screen.queryByRole('radio')).not.toBeInTheDocument();
  expect(screen.queryByRole('button', { name: /save filter/i })).not.toBeInTheDocument();
});

test('a failed circle list load still shows the stored filter', async () => {
  vi.mocked(getCardDAVCircleFilter).mockResolvedValue({ mode: 'include', circles: ['Family'] });
  vi.mocked(getCircles).mockRejectedValue(new Error('network down'));
  renderSettings();

  expect(await screen.findByText('Family')).toBeInTheDocument();
  expect(screen.getByRole('button', { name: /save filter/i })).toBeEnabled();
  expect(screen.queryByText(/no contact is in/i)).not.toBeInTheDocument();
});

test('warns when every included circle has no contacts left', async () => {
  vi.mocked(getCardDAVCircleFilter).mockResolvedValue({ mode: 'include', circles: ['Old friends'] });
  renderSettings();

  expect(await screen.findByText(/devices would show no contacts at all/i)).toBeInTheDocument();
});

test('names unused circles and matches circle names case-insensitively', async () => {
  vi.mocked(getCardDAVCircleFilter).mockResolvedValue({ mode: 'exclude', circles: ['work', 'Old friends'] });
  renderSettings();

  expect(await screen.findByText('No contact is in these circles any more: Old friends')).toBeInTheDocument();
});

test('switching back to all drops the circles', async () => {
  vi.mocked(getCardDAVCircleFilter).mockResolvedValue({ mode: 'exclude', circles: ['Work'] });
  renderSettings();

  fireEvent.click(await screen.findByRole('radio', { name: /^sync all contacts$/i }));
  fireEvent.click(screen.getByRole('button', { name: /save filter/i }));

  await waitFor(() =>
    expect(updateCardDAVCircleFilter).toHaveBeenCalledWith({ mode: 'all', circles: [] })
  );
});
