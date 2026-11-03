// CLAUDE.md requires every date shown to a user to honour the date-format preference.
import { test, expect, vi, beforeEach, afterEach, describe } from 'vitest';
import { render, screen, fireEvent, cleanup, act, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { I18nextProvider } from 'react-i18next';
import i18n from '../i18n/config';
import { DateFormat, DateFormatProvider, useDateFormat } from '../DateFormatProvider';
import ContactTimeline from './ContactTimeline';
import ReminderList from './ReminderList';
import type { Note } from '../api/notes';
import type { Reminder } from '../api/reminders';

// 4 March: day and month differ, so a swapped order cannot pass.
const date = '2026-03-04T00:00:00Z';
const expected: [DateFormat, string][] = [
  ['eu', '04.03.2026'],
  ['us', '03/04/2026'],
  ['iso', '2026-03-04'],
];
const allFormatted = expected.map(([, text]) => text);

const note: Note = { ID: 1, content: 'Coffee', date, CreatedAt: '', UpdatedAt: '' };
const reminder: Reminder = {
  ID: 2,
  message: 'Call',
  by_mail: false,
  remind_at: date,
  recurrence: 'once',
  reoccur_from_completion: false,
  completed: false,
  email_sent: false,
  contact_id: 1,
};

let setFormat: (format: DateFormat) => void;
function FormatSetter() {
  setFormat = useDateFormat().setDateFormat;
  return null;
}

function renderWith(format: DateFormat, ui: React.ReactNode) {
  localStorage.setItem('dateFormat', format);
  return render(
    <I18nextProvider i18n={i18n}>
      <DateFormatProvider>
        <FormatSetter />
        <MemoryRouter>{ui}</MemoryRouter>
      </DateFormatProvider>
    </I18nextProvider>
  );
}

const timeline = (
  <ContactTimeline timelineItems={[{ type: 'note', data: note, date }]} onEditItem={vi.fn()} />
);
const reminderList = (
  <ReminderList reminders={[reminder]} onComplete={vi.fn()} onEdit={vi.fn()} onDelete={vi.fn()} />
);

function expectOnly(text: string, container: HTMLElement) {
  expect(within(container).getByText(text)).toBeInTheDocument();
  for (const other of allFormatted.filter((t) => t !== text)) {
    expect(within(container).queryByText(other)).not.toBeInTheDocument();
  }
}

beforeEach(() => localStorage.clear());
afterEach(cleanup);

describe.each([
  ['ContactTimeline', timeline],
  ['ReminderList', reminderList],
])('%s', (_name, ui) => {
  test.each(expected)('with the %s preference shows %s', (format, text) => {
    const { container } = renderWith(format, ui);
    expectOnly(text, container);
  });

  test('the rendered date differs between preferences', () => {
    const texts = expected.map(([format]) => {
      const { container, unmount } = renderWith(format, ui);
      const text = container.textContent;
      unmount();
      return text;
    });
    expect(new Set(texts).size).toBe(expected.length);
  });

  test('changing the preference updates the date without a reload', () => {
    const { container } = renderWith('eu', ui);
    expectOnly('04.03.2026', container);
    act(() => setFormat('us'));
    expectOnly('03/04/2026', container);
  });

  test('without a stored preference it falls back to the default format', () => {
    const { container } = render(
      <I18nextProvider i18n={i18n}>
        <DateFormatProvider>
          <MemoryRouter>{ui}</MemoryRouter>
        </DateFormatProvider>
      </I18nextProvider>
    );
    expectOnly('04.03.2026', container);
  });
});

test('ReminderList without reminders shows its empty state', () => {
  renderWith('us', <ReminderList reminders={[]} onComplete={vi.fn()} onEdit={vi.fn()} onDelete={vi.fn()} />);
  expect(screen.getByText('No reminders yet')).toBeInTheDocument();
});

test('ReminderList asks for confirmation before deleting', async () => {
  const onDelete = vi.fn().mockResolvedValue(undefined);
  renderWith('iso', <ReminderList reminders={[reminder]} onComplete={vi.fn()} onEdit={vi.fn()} onDelete={onDelete} />);
  expect(screen.getByText('2026-03-04')).toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: 'Delete' }));
  expect(onDelete).not.toHaveBeenCalled();
  await act(async () => {
    fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Delete' }));
  });
  expect(onDelete).toHaveBeenCalledWith(2);
});

const sources = import.meta.glob<string>(['../**/*.{ts,tsx}', '!../**/*.test.{ts,tsx}'], {
  eager: true,
  query: '?raw',
  import: 'default',
});

test('no source formats a date with the browser locale instead of the preference', () => {
  expect(Object.keys(sources).length).toBeGreaterThan(50);
  const offenders = Object.entries(sources)
    .filter(([, source]) => /\.toLocale(Date|Time)?String\(/.test(source))
    .map(([path]) => path);
  expect(offenders).toEqual([]);
});
