import { test, expect, vi, beforeEach, afterEach } from 'vitest';
import { configure, render, screen, fireEvent, waitFor, cleanup } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import i18n from '../i18n/config';
import { DateFormat, DateFormatProvider } from '../DateFormatProvider';
import ReminderDialog from './ReminderDialog';
import type { Reminder } from '../api/reminders';

// MUI role queries on these large dialogs are slow; a busy machine must not turn that into a failure.
vi.setConfig({ testTimeout: 30000 });
configure({ asyncUtilTimeout: 10000 });

const existing: Reminder = {
  ID: 3,
  message: 'Call Anna',
  by_mail: false,
  remind_at: '2026-09-01T00:00:00Z',
  recurrence: 'monthly',
  reoccur_from_completion: false,
  completed: false,
  email_sent: false,
  contact_id: 7,
};

function renderDialog(
  props: Partial<React.ComponentProps<typeof ReminderDialog>> = {},
  format: DateFormat = 'eu'
) {
  localStorage.setItem('dateFormat', format);
  const p = { open: true, onClose: vi.fn(), onSave: vi.fn().mockResolvedValue(undefined), contactId: 7, ...props };
  render(
    <I18nextProvider i18n={i18n}>
      <DateFormatProvider>
        <ReminderDialog {...p} />
      </DateFormatProvider>
    </I18nextProvider>
  );
  return p;
}

const messageField = () => screen.getByRole('textbox', { name: /^Message/ });
const dateField = () => screen.getByRole('textbox', { name: /^Date/ });
const typeInto = (field: HTMLElement, value: string) => fireEvent.change(field, { target: { value } });
const save = () => fireEvent.click(screen.getByRole('button', { name: 'Save' }));

async function chooseRecurrence(name: string) {
  fireEvent.mouseDown(screen.getByRole('combobox', { name: /Recurrence/ }));
  fireEvent.click(screen.getByRole('option', { name }));
  await waitFor(() => expect(screen.queryByRole('listbox')).not.toBeInTheDocument());
}

beforeEach(() => {
  localStorage.clear();
  vi.useFakeTimers({ toFake: ['Date'] });
  vi.setSystemTime(new Date('2026-06-15T12:00:00Z'));
});

afterEach(() => {
  cleanup();
  vi.useRealTimers();
  vi.unstubAllEnvs();
});

test('a blank message is rejected', () => {
  const p = renderDialog();
  typeInto(messageField(), '   ');
  save();
  expect(screen.getByRole('alert')).toHaveTextContent('Message is required');
  expect(p.onSave).not.toHaveBeenCalled();
});

test('a missing date is rejected', () => {
  const p = renderDialog();
  typeInto(messageField(), 'Call');
  typeInto(dateField(), '');
  save();
  expect(screen.getByRole('alert')).toHaveTextContent('Date is required');
  expect(p.onSave).not.toHaveBeenCalled();
});

test.each<[DateFormat, string]>([
  ['eu', '12.2026'],
  ['eu', '12/25/2026'],
  ['us', '25/12/2026'],
  ['iso', '2026-13-01'],
])('in %s format the date %s is rejected', (format, value) => {
  const p = renderDialog({}, format);
  typeInto(messageField(), 'Call');
  typeInto(dateField(), value);
  save();
  expect(screen.getByRole('alert')).toHaveTextContent('Invalid date format');
  expect(p.onSave).not.toHaveBeenCalled();
});
test('a date that does not exist in the calendar is rejected', () => {
  const p = renderDialog();
  typeInto(messageField(), 'Call');
  typeInto(dateField(), '31.02.2026');
  save();
  expect(screen.getByRole('alert')).toHaveTextContent('Invalid date format');
  expect(p.onSave).not.toHaveBeenCalled();
});

test.each<[DateFormat, string]>([
  ['eu', '25.12.2026'],
  ['us', '12/25/2026'],
  ['iso', '2026-12-25'],
])('in %s format %s is saved as the same day, with a trimmed message', async (format, value) => {
  const p = renderDialog({}, format);
  typeInto(messageField(), '  Call Anna ');
  typeInto(dateField(), value);
  save();
  await waitFor(() => expect(p.onClose).toHaveBeenCalled());
  expect(p.onSave).toHaveBeenCalledWith({
    message: 'Call Anna',
    by_mail: true,
    remind_at: '2026-12-25T00:00:00.000Z',
    recurrence: 'once',
    reoccur_from_completion: true,
    contact_id: 7,
  });
});

test.each([
  ['Once', '15.06.2026'],
  ['Weekly', '22.06.2026'],
  ['Monthly', '15.07.2026'],
  ['Quarterly', '15.09.2026'],
  ['Every 6 months', '15.12.2026'],
  ['Yearly', '15.06.2027'],
])('choosing %s suggests %s for a new reminder', async (label, expected) => {
  renderDialog();
  await chooseRecurrence(label);
  expect(dateField()).toHaveValue(expected);
});

test('the suggested date follows the date format preference', () => {
  renderDialog({}, 'us');
  expect(dateField()).toHaveValue('06/15/2026');
  expect(dateField()).toHaveAttribute('placeholder', 'MM/DD/YYYY');
});

test('typing digits inserts the separators of the preferred format', () => {
  renderDialog({}, 'eu');
  typeInto(dateField(), '');
  typeInto(dateField(), '25122026');
  expect(dateField()).toHaveValue('25.12.2026');
});
test('a monthly suggestion from 31 January stays in February', async () => {
  vi.setSystemTime(new Date('2026-01-31T12:00:00Z'));
  renderDialog();
  await chooseRecurrence('Monthly');
  expect(dateField()).toHaveValue('28.02.2026');
});
test('the suggested date is the local calendar day', () => {
  vi.stubEnv('TZ', 'Europe/Berlin');
  vi.setSystemTime(new Date('2026-02-28T23:30:00Z')); // 00:30 on 1 March in Berlin
  renderDialog();
  expect(dateField()).toHaveValue('01.03.2026');
});

test('the options for rescheduling only appear for recurring reminders', async () => {
  const name = /Reschedule from completion date/;
  renderDialog();
  expect(screen.queryByRole('checkbox', { name })).not.toBeInTheDocument();
  await chooseRecurrence('Weekly');
  expect(screen.getByRole('checkbox', { name })).toBeChecked();
});

test('recurrence and both checkboxes reach the saved data', async () => {
  const p = renderDialog();
  typeInto(messageField(), 'Call');
  await chooseRecurrence('Weekly');
  fireEvent.click(screen.getByRole('checkbox', { name: /Reschedule from completion date/ }));
  fireEvent.click(screen.getByRole('checkbox', { name: /Send email notification/ }));
  save();
  await waitFor(() => expect(p.onClose).toHaveBeenCalled());
  expect(p.onSave).toHaveBeenCalledWith(
    expect.objectContaining({
      recurrence: 'weekly',
      by_mail: false,
      reoccur_from_completion: false,
      remind_at: '2026-06-22T00:00:00.000Z',
    })
  );
});

test('initial values prefill a new reminder', () => {
  renderDialog({ initialValues: { message: 'Stay in touch', recurrence: 'quarterly' } });
  expect(messageField()).toHaveValue('Stay in touch');
  expect(screen.getByRole('combobox', { name: /Recurrence/ })).toHaveTextContent('Quarterly');
  expect(dateField()).toHaveValue('15.09.2026');
});

test('editing prefills from the reminder and keeps its date when recurrence changes', async () => {
  renderDialog({ reminder: existing });
  expect(screen.getByRole('heading', { name: 'Edit Reminder' })).toBeInTheDocument();
  expect(messageField()).toHaveValue('Call Anna');
  expect(dateField()).toHaveValue('01.09.2026');
  expect(screen.getByRole('checkbox', { name: /Send email notification/ })).not.toBeChecked();
  await chooseRecurrence('Yearly');
  expect(dateField()).toHaveValue('01.09.2026');
});

test('a failed save shows the error and stays open for another try', async () => {
  const p = renderDialog({ onSave: vi.fn().mockRejectedValue(new Error('boom')) });
  typeInto(messageField(), 'Call');
  save();
  expect(await screen.findByRole('alert')).toHaveTextContent('boom');
  expect(p.onClose).not.toHaveBeenCalled();
  expect(screen.getByRole('button', { name: 'Save' })).toBeEnabled();
});

test('the message counter reports the length against the limit', () => {
  renderDialog();
  typeInto(messageField(), 'Hello');
  expect(screen.getByText('5/500')).toBeInTheDocument();
  expect(messageField()).toHaveAttribute('maxlength', '500');
});
