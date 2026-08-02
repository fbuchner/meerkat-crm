import { test, expect, vi, afterEach } from 'vitest';
import { render, screen, cleanup, fireEvent } from '@testing-library/react';
import '../i18n/config';
import PreferenceDialog from './PreferenceDialog';

afterEach(cleanup);

function renderDialog(props: Partial<React.ComponentProps<typeof PreferenceDialog>> = {}) {
  const defaults: React.ComponentProps<typeof PreferenceDialog> = {
    open: true,
    onClose: vi.fn(),
    onSave: vi.fn().mockResolvedValue(undefined),
    ...props,
  };
  return render(<PreferenceDialog {...defaults} />);
}

test('create mode shows the category, value, key, and sensitivity fields', () => {
  renderDialog();
  expect(screen.getByLabelText('Category *')).toBeInTheDocument();
  expect(screen.getByLabelText('Value *')).toBeInTheDocument();
  expect(screen.getByLabelText('Key')).toBeInTheDocument();
  expect(screen.getByLabelText('Sensitivity')).toBeInTheDocument();
});

test('requires a value before saving', async () => {
  const onSave = vi.fn().mockResolvedValue(undefined);
  renderDialog({ onSave });

  fireEvent.click(screen.getByRole('button', { name: 'Save' }));

  expect(await screen.findByText('Value is required')).toBeInTheDocument();
  expect(onSave).not.toHaveBeenCalled();
});

test('saves with the selected category, value, and sensitivity', async () => {
  const onSave = vi.fn().mockResolvedValue(undefined);
  renderDialog({ onSave });

  fireEvent.change(screen.getByLabelText('Value *'), { target: { value: 'Photography' } });
  fireEvent.mouseDown(screen.getByLabelText('Category *'));
  fireEvent.click(await screen.findByRole('option', { name: 'Hobby' }));
  fireEvent.click(screen.getByRole('button', { name: 'Save' }));

  await vi.waitFor(() => expect(onSave).toHaveBeenCalled());
  expect(onSave).toHaveBeenCalledWith({
    category: 'hobby',
    key: undefined,
    value: 'Photography',
    sensitivity: 'normal',
  });
});

test('edit mode pre-fills the existing preference', () => {
  renderDialog({
    preference: {
      id: 'p1',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
      entity_id: 'alice-uid',
      category: 'hobby',
      value: 'Vegetarian',
      sensitivity: 'normal',
    },
  });
  expect(screen.getByLabelText('Value *')).toHaveValue('Vegetarian');
});
