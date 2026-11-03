import { test, expect, vi, beforeEach, afterEach } from 'vitest';
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import i18n from '../i18n/config';
import { DateFormatProvider } from '../DateFormatProvider';
import AddActivityDialog from './AddActivityDialog';
import * as contactsApi from '../api/contacts';

vi.mock('../api/contacts');
const mocked = vi.mocked(contactsApi);
const anna = { ID: 7, firstname: 'Anna', lastname: 'Schmidt' } as contactsApi.Contact;

beforeEach(() => vi.resetAllMocks());
afterEach(cleanup);

test('saving waits for the preselected contact, so the activity is not stored without it', async () => {
  let resolve!: (r: contactsApi.ContactsResponse) => void;
  // The first call loads the preselected contact; the debounced search makes its own later.
  mocked.getContacts.mockImplementationOnce(() => new Promise((r) => { resolve = r; }));
  mocked.getContacts.mockResolvedValue({ contacts: [anna], total: 1, page: 1, limit: 100 });
  const onSave = vi.fn().mockResolvedValue(undefined);
  render(
    <I18nextProvider i18n={i18n}>
      <DateFormatProvider>
        <AddActivityDialog open onClose={vi.fn()} onSave={onSave} preselectedContactId={7} />
      </DateFormatProvider>
    </I18nextProvider>
  );
  fireEvent.change(screen.getByRole('textbox', { name: /^Title/ }), { target: { value: 'Coffee' } });
  const save = await screen.findByRole('button', { name: 'Save' });
  await waitFor(() => expect(save).toBeDisabled());

  await act(async () => resolve({ contacts: [anna], total: 1, page: 1, limit: 100 }));
  await waitFor(() => expect(save).toBeEnabled());
  fireEvent.click(save);

  await waitFor(() => expect(onSave).toHaveBeenCalled());
  expect(onSave.mock.calls[0][0].contact_ids).toEqual([7]);
});
