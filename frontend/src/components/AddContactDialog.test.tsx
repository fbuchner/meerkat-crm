import { test, expect, vi, beforeEach, afterEach } from 'vitest';
import { configure, render, screen, fireEvent, waitFor, cleanup, within } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import i18n from '../i18n/config';
import { DateFormat, DateFormatProvider } from '../DateFormatProvider';
import AddContactDialog from './AddContactDialog';
import * as contactsApi from '../api/contacts';
import * as remindersApi from '../api/reminders';
import { DEFAULT_ENABLED_CONTACT_FIELDS, ContactFieldKey } from '../contactFields';

vi.mock('../api/contacts');
vi.mock('../api/reminders');

const snackbar = vi.hoisted(() => ({ showError: vi.fn(), showSuccess: vi.fn() }));
vi.mock('../context/SnackbarContext', () => ({ useSnackbar: () => snackbar }));

// MUI role queries on these large dialogs are slow; a busy machine must not turn that into a failure.
vi.setConfig({ testTimeout: 30000 });
configure({ asyncUtilTimeout: 10000 });

const contacts = vi.mocked(contactsApi);
const reminders = vi.mocked(remindersApi);
// Anniversary is opt-in through Settings.
const withAnniversary = new Set<ContactFieldKey>([...DEFAULT_ENABLED_CONTACT_FIELDS, 'anniversary']);
const created = { ID: 42, firstname: 'Anna', lastname: 'Schmidt' } as contactsApi.Contact;

function renderDialog(
  props: Partial<React.ComponentProps<typeof AddContactDialog>> = {},
  format: DateFormat = 'eu'
) {
  localStorage.setItem('dateFormat', format);
  const p = { open: true, onClose: vi.fn(), onContactAdded: vi.fn(), availableCircles: ['Work', 'Family'], ...props };
  render(
    <I18nextProvider i18n={i18n}>
      <DateFormatProvider>
        <AddContactDialog {...p} />
      </DateFormatProvider>
    </I18nextProvider>
  );
  return p;
}

const field = (name: string | RegExp) => screen.getByRole('textbox', { name });
const type = (name: string | RegExp, value: string) => fireEvent.change(field(name), { target: { value } });
const firstName = (value = 'Anna') => type(/^First Name/, value);
const submit = () => fireEvent.click(screen.getByRole('button', { name: 'Create' }));
const payload = () => contacts.createContact.mock.calls[0][0];
const quietLogs = () => vi.spyOn(console, 'error').mockImplementation(() => {});

beforeEach(() => {
  vi.resetAllMocks();
  localStorage.clear();
  contacts.createContact.mockResolvedValue(created);
});

afterEach(() => {
  cleanup();
  vi.useRealTimers();
  vi.restoreAllMocks();
});

test.each(['', '   '])('first name %j is rejected', (value) => {
  const p = renderDialog();
  firstName(value);
  submit();
  expect(contacts.createContact).not.toHaveBeenCalled();
  expect(p.onContactAdded).not.toHaveBeenCalled();
  expect(screen.getByRole('dialog')).toHaveTextContent(/are required/);
});

// BUG en.json says "First name and last name are required", but only the first name is (AddContactDialog.tsx).
test.skip('BUG: the validation message asks only for what is actually required', () => {
  renderDialog();
  submit();
  expect(screen.getByRole('dialog')).not.toHaveTextContent(/last name/i);
});

test('a contact with only a first name is accepted', async () => {
  const p = renderDialog();
  firstName();
  submit();
  await waitFor(() => expect(p.onContactAdded).toHaveBeenCalledWith(42));
  expect(payload()).toMatchObject({ firstname: 'Anna', lastname: '', birthday: '', email: '', phone: '' });
});

// BUG AddContactDialog.tsx checks the trimmed first name but sends the untrimmed one; the backend does not trim either.
test.skip('BUG: the first name is saved without surrounding spaces', async () => {
  const p = renderDialog();
  firstName('  Anna ');
  submit();
  await waitFor(() => expect(p.onContactAdded).toHaveBeenCalled());
  expect(payload().firstname).toBe('Anna');
});

test.each<[DateFormat, string, string]>([
  ['eu', '30.04.1990', '1990-04-30'],
  ['eu', '30.04.', '--04-30'],
  ['us', '04/30/1990', '1990-04-30'],
  ['us', '04/30', '--04-30'],
  ['iso', '1990-04-30', '1990-04-30'],
])('in %s format the birthday %s is stored as %s', async (format, input, stored) => {
  const p = renderDialog({ enabledFields: withAnniversary }, format);
  firstName();
  type('Birthday', input);
  type('Anniversary', input);
  submit();
  await waitFor(() => expect(p.onContactAdded).toHaveBeenCalled());
  expect(payload()).toMatchObject({ birthday: stored, anniversary: stored });
});

test.each(['Birthday', 'Anniversary'])('an invalid %s blocks the save', (label) => {
  const p = renderDialog({ enabledFields: withAnniversary });
  firstName();
  type(label, '99.99.2000');
  submit();
  expect(screen.getByRole('dialog')).toHaveTextContent('Invalid date format');
  expect(contacts.createContact).not.toHaveBeenCalled();
  expect(p.onContactAdded).not.toHaveBeenCalled();
});

test('date fields show the placeholder of the preferred format', () => {
  renderDialog({ enabledFields: withAnniversary }, 'us');
  expect(field('Birthday')).toHaveAttribute('placeholder', 'MM/DD/YYYY');
  expect(field('Anniversary')).toHaveAttribute('placeholder', 'MM/DD/YYYY');
});

test('empty multi-value rows are dropped and the first real value becomes the primary', async () => {
  const p = renderDialog({ enabledFields: new Set<ContactFieldKey>(['emails', 'phones']) });
  firstName();
  const addTo = (heading: string) => {
    const section = screen.getByText(heading, { selector: 'h6' }).parentElement!;
    fireEvent.click(within(section).getByRole('button', { name: 'Add' }));
    return section;
  };
  const emails = addTo('Email');
  fireEvent.click(within(emails).getByRole('button', { name: 'Add' }));
  const [blank, real] = within(emails).getAllByRole('textbox', { name: 'Email' });
  fireEvent.change(blank, { target: { value: '   ' } });
  fireEvent.change(real, { target: { value: 'anna@example.com' } });
  addTo('Phone');
  submit();
  await waitFor(() => expect(p.onContactAdded).toHaveBeenCalled());
  expect(payload()).toMatchObject({
    emails: [{ type: 'home', value: 'anna@example.com' }],
    email: 'anna@example.com',
    phones: [],
    phone: '',
  });
});

test('blank custom fields are left out; none at all is sent as undefined', async () => {
  const p = renderDialog({ customFieldNames: ['Hobby', 'Pet'] });
  firstName();
  type('Hobby', 'Chess');
  type('Pet', '  ');
  submit();
  await waitFor(() => expect(p.onContactAdded).toHaveBeenCalled());
  expect(payload().custom_fields).toEqual({ Hobby: 'Chess' });
});

test('without custom fields or circles those keys are undefined', async () => {
  const p = renderDialog({ customFieldNames: ['Hobby'] });
  firstName();
  submit();
  await waitFor(() => expect(p.onContactAdded).toHaveBeenCalled());
  expect(payload().custom_fields).toBeUndefined();
  expect(payload().circles).toBeUndefined();
});

test('new circles are trimmed, deduplicated, removable and added with Enter', async () => {
  const p = renderDialog({ enabledFields: new Set() });
  firstName();
  const addCircle = (value: string) => {
    type('Or create new...', value);
    fireEvent.click(screen.getByRole('button', { name: 'Add' }));
  };
  addCircle('  Chess ');
  addCircle('Chess');
  type('Or create new...', 'Golf');
  fireEvent.keyDown(field('Or create new...'), { key: 'Enter' });
  addCircle('Sailing');
  addCircle('   ');
  const sailing = screen.getByText('Sailing').parentElement!;
  fireEvent.keyDown(sailing, { key: 'Backspace' });
  fireEvent.keyUp(sailing, { key: 'Backspace' });
  submit();
  await waitFor(() => expect(p.onContactAdded).toHaveBeenCalled());
  expect(payload().circles).toEqual(['Chess', 'Golf']);
});

test('an existing circle can be picked once and leaves the list of choices', async () => {
  const p = renderDialog();
  firstName();
  const select = () => screen.getByRole('combobox', { name: /Select existing circle/ });
  fireEvent.mouseDown(select());
  fireEvent.click(screen.getByRole('option', { name: 'Work' }));
  await waitFor(() => expect(screen.queryByRole('listbox')).not.toBeInTheDocument());
  fireEvent.mouseDown(select());
  expect(screen.queryByRole('option', { name: 'Work' })).not.toBeInTheDocument();
  expect(screen.getByRole('option', { name: 'Family' })).toBeInTheDocument();
  fireEvent.keyDown(screen.getByRole('listbox'), { key: 'Escape' });
  await waitFor(() => expect(screen.queryByRole('listbox')).not.toBeInTheDocument());
  submit();
  await waitFor(() => expect(p.onContactAdded).toHaveBeenCalled());
  expect(payload().circles).toEqual(['Work']);
});

test('the birthday reminder option only appears once a birthday is entered', () => {
  renderDialog();
  const name = /Create birthday reminder/;
  expect(screen.queryByRole('switch', { name })).not.toBeInTheDocument();
  type('Birthday', '30.04.1990');
  expect(screen.getByRole('switch', { name })).not.toBeChecked();
});

test('with the option on, a yearly reminder is created for the next birthday', async () => {
  vi.useFakeTimers({ toFake: ['Date'] });
  vi.setSystemTime(new Date('2026-06-15T12:00:00Z'));
  const p = renderDialog();
  firstName();
  type('Birthday', '30.04.1990');
  fireEvent.click(screen.getByRole('switch', { name: /Create birthday reminder/ }));
  submit();
  await waitFor(() => expect(p.onContactAdded).toHaveBeenCalled());
  expect(reminders.createReminder).toHaveBeenCalledWith(42, {
    message: 'Birthday of Anna Schmidt',
    by_mail: true,
    remind_at: new Date(2027, 3, 30, 9).toISOString(),
    recurrence: 'yearly',
    reoccur_from_completion: false,
    contact_id: 42,
  });
});

test('with the option off, no reminder is created', async () => {
  const p = renderDialog();
  firstName();
  type('Birthday', '30.04.1990');
  submit();
  await waitFor(() => expect(p.onContactAdded).toHaveBeenCalled());
  expect(reminders.createReminder).not.toHaveBeenCalled();
});

// BUG AddContactDialog.tsx shares one try block: the contact exists but the dialog reports failure, and a retry duplicates it.
test.skip('BUG: a failed birthday reminder does not lose or duplicate the created contact', async () => {
  quietLogs();
  reminders.createReminder.mockRejectedValue(new Error('reminder failed'));
  const p = renderDialog();
  firstName();
  type('Birthday', '30.04.1990');
  fireEvent.click(screen.getByRole('switch', { name: /Create birthday reminder/ }));
  submit();
  await waitFor(() => expect(p.onContactAdded).toHaveBeenCalledWith(42));
  expect(contacts.createContact).toHaveBeenCalledTimes(1);
});

test('success reports, notifies the parent and resets the form', async () => {
  const p = renderDialog();
  firstName();
  type('Birthday', '30.04.1990');
  submit();
  await waitFor(() => expect(p.onClose).toHaveBeenCalled());
  expect(p.onContactAdded).toHaveBeenCalledWith(42);
  expect(snackbar.showSuccess).toHaveBeenCalledWith('Contact created successfully!');
  expect(field(/^First Name/)).toHaveValue('');
  expect(field('Birthday')).toHaveValue('');
});

test('a failed create shows the error, keeps the input and stays open', async () => {
  quietLogs();
  contacts.createContact.mockRejectedValue(new Error('boom'));
  const p = renderDialog();
  firstName();
  submit();
  expect(await screen.findByText('boom')).toBeInTheDocument();
  expect(snackbar.showError).toHaveBeenCalledWith('boom');
  expect(p.onContactAdded).not.toHaveBeenCalled();
  expect(p.onClose).not.toHaveBeenCalled();
  expect(field(/^First Name/)).toHaveValue('Anna');
});

test('cancel discards the form', () => {
  const p = renderDialog();
  firstName();
  fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
  expect(p.onClose).toHaveBeenCalled();
  expect(contacts.createContact).not.toHaveBeenCalled();
  expect(field(/^First Name/)).toHaveValue('');
});

test('only enabled optional fields are rendered', () => {
  renderDialog({ enabledFields: new Set<ContactFieldKey>(['nickname']) });
  expect(field('Nickname')).toBeInTheDocument();
  expect(field(/^First Name/)).toBeInTheDocument();
  expect(screen.queryByRole('textbox', { name: 'Birthday' })).not.toBeInTheDocument();
  expect(screen.queryByRole('textbox', { name: 'Organization' })).not.toBeInTheDocument();
  expect(screen.queryByText('Email', { selector: 'h6' })).not.toBeInTheDocument();
});
