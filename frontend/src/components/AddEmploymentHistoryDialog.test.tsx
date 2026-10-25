import { test, expect, vi, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import i18n from '../i18n/config';
import AddEmploymentHistoryDialog from './AddEmploymentHistoryDialog';

vi.mock('../context/SnackbarContext', () => ({
  useSnackbar: () => ({ showError: vi.fn(), showSuccess: vi.fn() }),
}));

afterEach(cleanup);

function renderDialog(props: Partial<React.ComponentProps<typeof AddEmploymentHistoryDialog>> = {}) {
  const p = { open: true, onClose: vi.fn(), onSave: vi.fn().mockResolvedValue(undefined), ...props };
  render(
    <I18nextProvider i18n={i18n}>
      <AddEmploymentHistoryDialog {...p} />
    </I18nextProvider>
  );
  return p;
}

const type = (label: string, value: string) =>
  fireEvent.change(screen.getByLabelText(new RegExp(label)), { target: { value } });

test('requires organization', () => {
  const p = renderDialog();
  fireEvent.click(screen.getByText('Save'));
  expect(screen.getByText('Organization is required')).toBeInTheDocument();
  expect(p.onSave).not.toHaveBeenCalled();
});

test('rejects malformed dates', () => {
  const p = renderDialog();
  type('Organization', 'Acme');
  type('Start Date', '15/03/2020');
  fireEvent.click(screen.getByText('Save'));
  expect(screen.getByText('Dates must be YYYY, YYYY-MM, or YYYY-MM-DD')).toBeInTheDocument();
  expect(p.onSave).not.toHaveBeenCalled();
});

test('saves trimmed data, omits blanks, then closes', async () => {
  const p = renderDialog();
  type('Organization', '  Acme ');
  type('Start Date', '2020-03');
  fireEvent.click(screen.getByText('Save'));
  await waitFor(() => expect(p.onClose).toHaveBeenCalled());
  expect(p.onSave).toHaveBeenCalledWith({
    organization: 'Acme',
    department: undefined,
    job_title: undefined,
    role: undefined,
    start_date: '2020-03',
    end_date: undefined,
    notes: undefined,
  });
});

test('edit mode prefills from the entry', () => {
  renderDialog({
    entry: { ID: 1, CreatedAt: '', UpdatedAt: '', contact_id: 1, organization: 'Acme', job_title: 'Dev', start_date: '2019' },
  });
  expect(screen.getByText('Edit Job')).toBeInTheDocument();
  expect(screen.getByDisplayValue('Acme')).toBeInTheDocument();
  expect(screen.getByDisplayValue('Dev')).toBeInTheDocument();
  expect(screen.getByDisplayValue('2019')).toBeInTheDocument();
});

test('stays open and shows the error when save fails', async () => {
  const p = renderDialog({ onSave: vi.fn().mockRejectedValue(new Error('boom')) });
  type('Organization', 'Acme');
  fireEvent.click(screen.getByText('Save'));
  await waitFor(() => expect(screen.getByText('boom')).toBeInTheDocument());
  expect(p.onClose).not.toHaveBeenCalled();
});
