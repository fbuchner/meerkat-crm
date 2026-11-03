import { test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, cleanup, within } from '@testing-library/react';
import { MemoryRouter, useLocation } from 'react-router-dom';
import { I18nextProvider } from 'react-i18next';
import i18n from '../i18n/config';
import { DateFormatProvider } from '../DateFormatProvider';
import ContactTimeline from './ContactTimeline';
import type { Note } from '../api/notes';
import type { Activity } from '../api/activities';
import type { ReminderCompletion } from '../api/reminders';

const date = '2026-03-04T00:00:00Z';
const note: Note = { ID: 1, content: 'Met for coffee', date, CreatedAt: '', UpdatedAt: '' };
const activity: Activity = {
  ID: 2,
  title: 'Lunch',
  description: 'Pizza',
  location: 'Berlin',
  date,
  CreatedAt: '',
  UpdatedAt: '',
  contacts: [
    { ID: 8, firstname: 'Anna', lastname: 'Schmidt', nickname: 'Annie' },
    { ID: 9, firstname: 'Ben', lastname: 'Meyer' },
  ],
};
const completion: ReminderCompletion = { ID: 3, contact_id: 1, message: 'Called Anna', completed_at: date };

type Items = React.ComponentProps<typeof ContactTimeline>['timelineItems'];
const noteItem = { type: 'note', data: note, date } as const;
const activityItem = { type: 'activity', data: activity, date } as const;
const completionItem = { type: 'completion', data: completion, date } as const;

const CurrentPath = () => <div data-testid="path">{useLocation().pathname}</div>;

function renderTimeline(timelineItems: Items) {
  const p = { onEditItem: vi.fn(), onDeleteCompletion: vi.fn() };
  render(
    <I18nextProvider i18n={i18n}>
      <DateFormatProvider>
        <MemoryRouter>
          <ContactTimeline timelineItems={timelineItems} {...p} />
          <CurrentPath />
        </MemoryRouter>
      </DateFormatProvider>
    </I18nextProvider>
  );
  return p;
}

beforeEach(() => localStorage.clear());
afterEach(cleanup);

test('with nothing to show it says so', () => {
  renderTimeline([]);
  expect(screen.getByText('No notes or activities yet')).toBeInTheDocument();
  expect(screen.queryByRole('listitem')).not.toBeInTheDocument();
});

test('renders one entry per item, in the given order', () => {
  renderTimeline([noteItem, activityItem, completionItem]);
  const items = screen.getAllByRole('listitem');
  expect(items).toHaveLength(3);
  expect(items[0]).toHaveTextContent('Met for coffee');
  expect(items[1]).toHaveTextContent('Lunch');
  expect(items[2]).toHaveTextContent('Called Anna');
});

test('a note shows its label and content', () => {
  renderTimeline([noteItem]);
  const item = within(screen.getByRole('listitem'));
  expect(item.getByText('Note')).toBeInTheDocument();
  expect(item.getByText('Met for coffee')).toBeInTheDocument();
});

test('an activity shows title, description, location and participants', () => {
  renderTimeline([activityItem]);
  const item = within(screen.getByRole('listitem'));
  expect(item.getByText('Lunch')).toBeInTheDocument();
  expect(item.getByText('Pizza')).toBeInTheDocument();
  expect(item.getByText(/Berlin/)).toBeInTheDocument();
  expect(item.getByRole('button', { name: 'Anna "Annie" Schmidt' })).toBeInTheDocument();
  expect(item.getByRole('button', { name: 'Ben Meyer' })).toBeInTheDocument();
});

test('an activity without details falls back and omits the empty sections', () => {
  renderTimeline([{ type: 'activity', data: { ...activity, description: '', location: '', contacts: [] }, date }]);
  const item = screen.getByRole('listitem');
  expect(item).toHaveTextContent('No description');
  expect(item).not.toHaveTextContent('📍');
  expect(item).not.toHaveTextContent('👥');
});

test('a completed reminder shows its label and message', () => {
  renderTimeline([completionItem]);
  const item = within(screen.getByRole('listitem'));
  expect(item.getByText('Reminder Completed')).toBeInTheDocument();
  expect(item.getByText('Called Anna')).toBeInTheDocument();
});

test('clicking a participant opens that contact', () => {
  renderTimeline([activityItem]);
  fireEvent.click(screen.getByRole('button', { name: 'Ben Meyer' }));
  expect(screen.getByTestId('path')).toHaveTextContent('/contacts/9');
});

// The delete icon has no accessible name (see the skipped test below), so find the actions by position.
const lastButton = (item: HTMLElement) => within(item).getAllByRole('button').at(-1)!;

test('the action on a note or activity opens it for editing', () => {
  const p = renderTimeline([noteItem, activityItem]);
  const [first, second] = screen.getAllByRole('listitem');
  fireEvent.click(lastButton(first));
  expect(p.onEditItem).toHaveBeenLastCalledWith('note', note);
  fireEvent.click(lastButton(second));
  expect(p.onEditItem).toHaveBeenLastCalledWith('activity', activity);
  expect(p.onDeleteCompletion).not.toHaveBeenCalled();
});

test('the action on a completed reminder deletes that completion', () => {
  const p = renderTimeline([completionItem]);
  fireEvent.click(lastButton(screen.getByRole('listitem')));
  expect(p.onDeleteCompletion).toHaveBeenCalledWith(3);
  expect(p.onEditItem).not.toHaveBeenCalled();
});
test('edit and delete actions have accessible names', () => {
  renderTimeline([noteItem, completionItem]);
  expect(screen.getByRole('button', { name: /edit/i })).toBeInTheDocument();
  expect(screen.getByRole('button', { name: /delete/i })).toBeInTheDocument();
});

test('an unparseable date is shown as stored rather than as "Invalid Date"', () => {
  renderTimeline([{ ...noteItem, date: 'sometime' }]);
  const item = screen.getByRole('listitem');
  expect(item).toHaveTextContent('sometime');
  expect(item).not.toHaveTextContent('Invalid Date');
});

test('a missing date does not crash or print "Invalid Date"', () => {
  renderTimeline([{ ...noteItem, date: '' }]);
  const item = screen.getByRole('listitem');
  expect(item).toHaveTextContent('Met for coffee');
  expect(item).not.toHaveTextContent('Invalid Date');
});
test('the missing-date fallback is translated', async () => {
  await i18n.changeLanguage('de');
  try {
    renderTimeline([{ ...noteItem, date: '' }]);
    expect(screen.getByRole('listitem')).not.toHaveTextContent('N/A');
  } finally {
    await i18n.changeLanguage('en');
  }
});
