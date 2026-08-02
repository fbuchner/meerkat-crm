import { test, expect, vi, afterEach } from 'vitest';
import { render, screen, cleanup, fireEvent } from '@testing-library/react';
import '../i18n/config';
import HouseholdDialog from './HouseholdDialog';

afterEach(cleanup);

function renderDialog(props: Partial<React.ComponentProps<typeof HouseholdDialog>> = {}) {
  const defaults: React.ComponentProps<typeof HouseholdDialog> = {
    open: true,
    onClose: vi.fn(),
    onSave: vi.fn().mockResolvedValue(undefined),
    ...props,
  };
  return render(<HouseholdDialog {...defaults} />);
}

test('create mode shows the name and type fields', () => {
  renderDialog();
  expect(screen.getByLabelText('Name *')).toBeInTheDocument();
  expect(screen.getByLabelText('Type *')).toBeInTheDocument();
});

test('requires a name before saving', async () => {
  const onSave = vi.fn().mockResolvedValue(undefined);
  renderDialog({ onSave });

  fireEvent.click(screen.getByRole('button', { name: 'Save' }));

  expect(await screen.findByText('Name is required')).toBeInTheDocument();
  expect(onSave).not.toHaveBeenCalled();
});

test('saves with the entered name and selected type', async () => {
  const onSave = vi.fn().mockResolvedValue(undefined);
  renderDialog({ onSave });

  fireEvent.change(screen.getByLabelText('Name *'), { target: { value: 'Smith Family' } });
  fireEvent.mouseDown(screen.getByLabelText('Type *'));
  fireEvent.click(await screen.findByText('Roommates'));
  fireEvent.click(screen.getByRole('button', { name: 'Save' }));

  await vi.waitFor(() => expect(onSave).toHaveBeenCalled());
  expect(onSave).toHaveBeenCalledWith({ name: 'Smith Family', type: 'roommates' });
});

test('edit mode pre-fills the existing household values', () => {
  renderDialog({
    household: { id: 'h1', created_at: '', updated_at: '', name: 'Apt 4B', type: 'roommates' },
  });
  expect(screen.getByLabelText('Name *')).toHaveValue('Apt 4B');
});
