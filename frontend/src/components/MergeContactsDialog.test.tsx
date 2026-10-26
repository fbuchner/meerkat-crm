import { test, expect, vi, afterEach, beforeEach } from 'vitest';
import { render, screen, fireEvent, cleanup, waitFor } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import i18n from '../i18n/config';
import { DateFormatProvider } from '../DateFormatProvider';
import MergeContactsDialog from './MergeContactsDialog';
import type { MergePreview } from '../api/contacts';

const api = vi.hoisted(() => ({
  getMergePreview: vi.fn(),
  mergeContacts: vi.fn(),
  getContacts: vi.fn(),
}));
vi.mock('../api/contacts', () => api);

const preview = (targetId: number, sourceId: number): MergePreview => ({
  target: { ID: targetId, firstname: 'Anna', lastname: targetId === 1 ? 'Schmidt' : 'Weber' },
  source: { ID: sourceId, firstname: 'Anna', lastname: sourceId === 1 ? 'Schmidt' : 'Weber' },
  conflicts: [
    { field: 'lastname', target_value: targetId === 1 ? 'Schmidt' : 'Weber', source_value: sourceId === 1 ? 'Schmidt' : 'Weber' },
    { field: 'birthday', target_value: '1990-01-02', source_value: '1991-03-04' },
    { field: 'custom_fields.Shoe size', target_value: '38', source_value: '39' },
  ],
  moved: { notes: 2, activities: 1, reminders: 0, relationships: 0 },
});

beforeEach(() => {
  i18n.changeLanguage('en');
  window.localStorage.setItem('dateFormat', 'eu');
  api.getMergePreview.mockImplementation((t: number, s: number) => Promise.resolve(preview(t, s)));
  api.mergeContacts.mockImplementation((t: number) => Promise.resolve({ ID: t, firstname: 'Anna', lastname: 'Weber' }));
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function renderDialog(onMerged = vi.fn()) {
  render(
    <I18nextProvider i18n={i18n}>
      <DateFormatProvider>
        <MergeContactsDialog open onClose={vi.fn()} contactId={1} otherContactId={2} onMerged={onMerged} />
      </DateFormatProvider>
    </I18nextProvider>
  );
  return onMerged;
}

test('lists conflicts with labels and user-formatted dates', async () => {
  renderDialog();

  expect(await screen.findByText('Last Name')).toBeInTheDocument();
  expect(screen.getByText('Shoe size')).toBeInTheDocument();
  expect(screen.getByText('02.01.1990')).toBeInTheDocument();
  expect(screen.getByText('04.03.1991')).toBeInTheDocument();
  expect(screen.getByText('Also moves 2 notes, 1 activity.')).toBeInTheDocument();
  expect(screen.getByText(/Anna Weber will be deleted/)).toBeInTheDocument();
});

test('sends the chosen value per conflict, defaulting to the kept contact', async () => {
  const onMerged = renderDialog();

  fireEvent.click(await screen.findByLabelText('Weber'));
  fireEvent.click(screen.getByRole('button', { name: 'Merge contacts' }));

  await waitFor(() => expect(onMerged).toHaveBeenCalled());
  expect(api.mergeContacts).toHaveBeenCalledWith(1, 2, {
    lastname: 'source',
    birthday: 'target',
    'custom_fields.Shoe size': 'target',
  });
  expect(onMerged).toHaveBeenCalledWith(expect.objectContaining({ ID: 1 }), 2);
});

test('swapping keeps the other contact instead', async () => {
  const onMerged = renderDialog();

  await screen.findByText('Last Name');
  fireEvent.click(screen.getByRole('button', { name: 'Swap which contact is kept' }));
  await waitFor(() => expect(api.getMergePreview).toHaveBeenLastCalledWith(2, 1));
  expect(await screen.findByText(/Anna Schmidt will be deleted/)).toBeInTheDocument();

  fireEvent.click(screen.getByRole('button', { name: 'Merge contacts' }));
  await waitFor(() => expect(onMerged).toHaveBeenCalledWith(expect.objectContaining({ ID: 2 }), 1));
});

test('labels deceased and archived conflicts instead of showing true/false', async () => {
  api.getMergePreview.mockResolvedValueOnce({
    ...preview(1, 2),
    conflicts: [
      { field: 'is_deceased', target_value: 'false', source_value: 'true' },
      { field: 'archived', target_value: 'true', source_value: 'false' },
    ],
  });
  renderDialog();

  expect(await screen.findByText('Not deceased')).toBeInTheDocument();
  expect(screen.getAllByText('Deceased').length).toBeGreaterThan(0);
  expect(screen.getByText('Archived (its reminders are deleted)')).toBeInTheDocument();
  expect(screen.getByText('Active')).toBeInTheDocument();
  expect(screen.queryByText('true')).not.toBeInTheDocument();
});
