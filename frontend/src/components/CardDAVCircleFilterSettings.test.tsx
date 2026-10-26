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

test('switching back to all drops the circles', async () => {
  vi.mocked(getCardDAVCircleFilter).mockResolvedValue({ mode: 'exclude', circles: ['Work'] });
  renderSettings();

  fireEvent.click(await screen.findByRole('radio', { name: /^sync all contacts$/i }));
  fireEvent.click(screen.getByRole('button', { name: /save filter/i }));

  await waitFor(() =>
    expect(updateCardDAVCircleFilter).toHaveBeenCalledWith({ mode: 'all', circles: [] })
  );
});
