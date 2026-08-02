import { test, expect, vi, afterEach } from 'vitest';
import { render, screen, cleanup, fireEvent } from '@testing-library/react';
import '../i18n/config';
import PreferenceList from './PreferenceList';
import { Preference } from '../api/preferences';

afterEach(cleanup);

function preference(overrides: Partial<Preference> = {}): Preference {
  return {
    id: 'p1',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    entity_id: 'alice-uid',
    category: 'food',
    value: 'Vegetarian',
    sensitivity: 'normal',
    ...overrides,
  };
}

test('renders preferences with category chips and values', () => {
  render(
    <PreferenceList
      preferences={[
        preference(),
        preference({ id: 'p2', category: 'hobby', value: 'Photography' }),
      ]}
      onEdit={vi.fn()}
      onDelete={vi.fn()}
    />
  );

  expect(screen.getByText('Vegetarian')).toBeInTheDocument();
  expect(screen.getByText('Photography')).toBeInTheDocument();
  expect(screen.getByText('Food')).toBeInTheDocument();
  expect(screen.getByText('Hobby')).toBeInTheDocument();
});

test('shows the sensitivity chip only for non-normal preferences', () => {
  render(
    <PreferenceList
      preferences={[
        preference(),
        preference({ id: 'p2', category: 'hobby', value: 'Secret project', sensitivity: 'secret' }),
      ]}
      onEdit={vi.fn()}
      onDelete={vi.fn()}
    />
  );

  expect(screen.getByText('Secret')).toBeInTheDocument();
  // Only one non-normal chip across the two entries.
  expect(screen.getAllByText('Secret')).toHaveLength(1);
});

test('edit and delete call their handlers', () => {
  const onEdit = vi.fn();
  const onDelete = vi.fn();
  const pref = preference();

  render(<PreferenceList preferences={[pref]} onEdit={onEdit} onDelete={onDelete} />);

  fireEvent.click(screen.getByLabelText('Edit'));
  expect(onEdit).toHaveBeenCalledWith(pref);

  fireEvent.click(screen.getByLabelText('Delete'));
  expect(onDelete).toHaveBeenCalledWith('p1');
});

test('shows the empty state', () => {
  render(<PreferenceList preferences={[]} onEdit={vi.fn()} onDelete={vi.fn()} />);
  expect(screen.getByText('No preferences yet')).toBeInTheDocument();
});
