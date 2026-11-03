import { test, expect, vi, beforeEach, afterEach } from 'vitest';
import { configure, render, screen, fireEvent, waitFor, cleanup, within } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import i18n from '../i18n/config';
import ImportContactsDialog from './ImportContactsDialog';
import * as api from '../api/import';
import type { ColumnMapping, ImportPreviewResponse, ImportRowPreview, ImportUploadResponse } from '../api/import';

vi.mock('../api/import', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../api/import')>()),
  uploadCSVForImport: vi.fn(),
  uploadVCFForImport: vi.fn(),
  getImportPreview: vi.fn(),
  confirmImport: vi.fn(),
  confirmVCFImport: vi.fn(),
}));

const snackbar = vi.hoisted(() => ({ showSuccess: vi.fn(), showError: vi.fn() }));
vi.mock('../context/SnackbarContext', () => ({ useSnackbar: () => snackbar }));

// MUI role queries on these large dialogs are slow; a busy machine must not turn that into a failure.
vi.setConfig({ testTimeout: 30000 });
configure({ asyncUtilTimeout: 10000 });

const mocked = vi.mocked(api);
const MB = 1024 * 1024;

function file(name: string, size = 10) {
  const f = new File(['x'], name);
  Object.defineProperty(f, 'size', { value: size });
  return f;
}

const mapping = (csv_column: string, contact_field: string, group = 0): ColumnMapping => ({ csv_column, contact_field, group });

function upload(mappings: ColumnMapping[]): ImportUploadResponse {
  return {
    session_id: 'csv-session',
    headers: mappings.map((m) => m.csv_column),
    suggested_mappings: mappings,
    row_count: 1,
    sample_data: [mappings.map((m) => `sample of ${m.csv_column}`)],
  };
}

const row = (row_index: number, firstname: string, extra: Partial<ImportRowPreview> = {}): ImportRowPreview => ({
  row_index,
  parsed_contact: { firstname, lastname: 'Doe', email: `${firstname}@example.com` },
  validation_errors: [],
  duplicate_match: null,
  suggested_action: 'add',
  ...extra,
});

const duplicate = (existing_deleted = false) => ({
  existing_contact_id: 1,
  existing_firstname: 'Stored',
  existing_lastname: 'Doe',
  existing_email: '',
  existing_phone: '',
  match_reason: 'email' as const,
  existing_deleted,
});

const preview = (rows: ImportRowPreview[], session_id = 'csv-session'): ImportPreviewResponse => ({
  session_id,
  rows,
  total_rows: rows.length,
  valid_rows: rows.length,
  duplicate_count: 0,
  error_count: 0,
});

const result = (extra = {}) => ({ total_processed: 1, created: 1, updated: 0, skipped: 0, errors: [], ...extra });

function renderDialog() {
  const p = { open: true, onClose: vi.fn(), onImportComplete: vi.fn() };
  render(
    <I18nextProvider i18n={i18n}>
      <ImportContactsDialog {...p} />
    </I18nextProvider>
  );
  return p;
}

const selectFile = (f: File) =>
  fireEvent.change(document.querySelector('input[type="file"]') as HTMLInputElement, { target: { files: [f] } });
const click = (name: string) => fireEvent.click(screen.getByRole('button', { name }));
const rowOf = (name: RegExp) => screen.getByRole('row', { name });

async function choose(combobox: HTMLElement, name: string) {
  fireEvent.mouseDown(combobox);
  fireEvent.click(screen.getByRole('option', { name }));
  await waitFor(() => expect(screen.queryByRole('listbox')).not.toBeInTheDocument());
}

async function toMapping(mappings: ColumnMapping[]) {
  mocked.uploadCSVForImport.mockResolvedValue(upload(mappings));
  selectFile(file('people.csv'));
  await screen.findByRole('columnheader', { name: 'CSV Column' });
}

async function toPreview(rows: ImportRowPreview[]) {
  await toMapping([mapping('Name', 'firstname')]);
  mocked.getImportPreview.mockResolvedValue(preview(rows));
  click('Continue');
  await screen.findByRole('columnheader', { name: 'Action' });
}

beforeEach(() => {
  vi.resetAllMocks();
  localStorage.clear();
});

afterEach(cleanup);

test('a file that is neither CSV nor VCF is rejected', () => {
  renderDialog();
  selectFile(file('people.xlsx'));
  expect(screen.getByRole('alert')).toHaveTextContent(/valid CSV/);
  expect(mocked.uploadCSVForImport).not.toHaveBeenCalled();
  expect(mocked.uploadVCFForImport).not.toHaveBeenCalled();
});
test('the invalid file message names both accepted types', () => {
  renderDialog();
  selectFile(file('people.xlsx'));
  expect(screen.getByRole('alert')).toHaveTextContent(/VCF/);
});

test('CSV files above 5 MB are refused before uploading', () => {
  renderDialog();
  selectFile(file('people.csv', 5 * MB + 1));
  expect(screen.getByRole('alert')).toHaveTextContent(/too large/);
  expect(mocked.uploadCSVForImport).not.toHaveBeenCalled();
});

test('VCF files above 10 MB are refused before uploading', () => {
  renderDialog();
  selectFile(file('people.vcf', 10 * MB + 1));
  expect(screen.getByRole('alert')).toHaveTextContent(/too large/);
  expect(mocked.uploadVCFForImport).not.toHaveBeenCalled();
});
test('the size error states the 10 MB limit for VCF files', () => {
  renderDialog();
  selectFile(file('people.vcf', 10 * MB + 1));
  expect(screen.getByRole('alert')).toHaveTextContent(/10\s?MB/);
});

test('a 6 MB VCF is within its limit and uploads', async () => {
  mocked.uploadVCFForImport.mockResolvedValue(preview([row(0, 'Anna')], 'vcf-session'));
  renderDialog();
  selectFile(file('people.vcf', 6 * MB));
  await screen.findByRole('columnheader', { name: 'Action' });
  expect(mocked.uploadVCFForImport).toHaveBeenCalledTimes(1);
});

test('the file extension check ignores case', async () => {
  mocked.uploadCSVForImport.mockResolvedValue(upload([mapping('Name', 'firstname')]));
  renderDialog();
  selectFile(file('PEOPLE.CSV'));
  await screen.findByRole('columnheader', { name: 'CSV Column' });
  expect(mocked.uploadCSVForImport).toHaveBeenCalledTimes(1);
});

test('a dropped file is uploaded like a selected one', async () => {
  mocked.uploadCSVForImport.mockResolvedValue(upload([mapping('Name', 'firstname')]));
  renderDialog();
  const dropZone = (document.querySelector('input[type="file"]') as HTMLElement).parentElement!;
  fireEvent.drop(dropZone, { dataTransfer: { files: [file('people.csv')] } });
  await screen.findByRole('columnheader', { name: 'CSV Column' });
});

test('a failed upload shows the error and stays on the upload step', async () => {
  mocked.uploadCSVForImport.mockRejectedValue(new Error('bad csv'));
  renderDialog();
  selectFile(file('people.csv'));
  expect(await screen.findByRole('alert')).toHaveTextContent('bad csv');
  expect(document.querySelector('input[type="file"]')).toBeInTheDocument();
});

test('the mapping step shows each column with its sample and suggested field', async () => {
  renderDialog();
  await toMapping([mapping('Name', 'firstname'), mapping('Mail', 'email')]);
  const name = within(rowOf(/^Name/));
  expect(name.getByText('sample of Name')).toBeInTheDocument();
  expect(name.getByRole('combobox')).toHaveTextContent('First Name');
  expect(within(rowOf(/^Mail/)).getByText('sample of Mail')).toBeInTheDocument();
});

test('continuing with every column ignored is refused', async () => {
  renderDialog();
  await toMapping([mapping('Name', ''), mapping('Mail', '')]);
  click('Continue');
  expect(screen.getByRole('alert')).toHaveTextContent('Please map at least one column');
  expect(mocked.getImportPreview).not.toHaveBeenCalled();
});

test('changed mappings are what the preview is requested with', async () => {
  mocked.getImportPreview.mockResolvedValue(preview([row(0, 'Anna')]));
  renderDialog();
  await toMapping([mapping('Name', 'firstname'), mapping('Mail', '')]);
  await choose(within(rowOf(/^Mail/)).getByRole('combobox'), 'Last Name');
  click('Continue');
  await screen.findByRole('columnheader', { name: 'Action' });
  expect(mocked.getImportPreview).toHaveBeenCalledWith('csv-session', [
    mapping('Name', 'firstname'),
    mapping('Mail', 'lastname'),
  ]);
});

test('a preview failure shows the error and stays on the mapping step', async () => {
  mocked.getImportPreview.mockRejectedValue(new Error('no preview'));
  renderDialog();
  await toMapping([mapping('Name', 'firstname')]);
  click('Continue');
  expect(await screen.findByRole('alert')).toHaveTextContent('no preview');
  expect(screen.getByRole('columnheader', { name: 'CSV Column' })).toBeInTheDocument();
});

test('two columns on one single-value field warn; the repeatable email field does not', async () => {
  renderDialog();
  await toMapping([
    mapping('First', 'firstname'),
    mapping('Given', 'firstname'),
    mapping('Mail 1', 'email'),
    mapping('Mail 2', 'email'),
  ]);
  expect(screen.getByRole('alert')).toHaveTextContent(/only the last column will be used: First Name$/);
  expect(screen.getAllByText('Mapped more than once')).toHaveLength(2);
  expect(within(rowOf(/^Mail 1/)).queryByText('Mapped more than once')).not.toBeInTheDocument();
});

test('the same part field in different groups is not a conflict', async () => {
  renderDialog();
  await toMapping([mapping('City 1', 'address_city', 0), mapping('City 2', 'address_city', 1)]);
  expect(screen.queryByRole('alert')).not.toBeInTheDocument();
});

test('resolving a mapping conflict removes the warning', async () => {
  renderDialog();
  await toMapping([mapping('First', 'firstname'), mapping('Given', 'firstname')]);
  expect(screen.getByRole('alert')).toBeInTheDocument();
  await choose(within(rowOf(/^Given/)).getByRole('combobox'), 'Last Name');
  expect(screen.queryByRole('alert')).not.toBeInTheDocument();
});

test('the preview summarises the suggested actions and follows changes', async () => {
  renderDialog();
  await toPreview([
    row(0, 'Anna'),
    row(1, 'Ben', { suggested_action: 'update', duplicate_match: duplicate() }),
    row(2, 'Cara', { suggested_action: 'skip' }),
  ]);
  expect(screen.getByText('1 to create')).toBeInTheDocument();
  expect(screen.getByText('1 to update')).toBeInTheDocument();
  expect(screen.getByText('1 to skip')).toBeInTheDocument();

  await choose(within(rowOf(/Cara/)).getByRole('combobox'), 'Add as New');
  expect(screen.getByText('2 to create')).toBeInTheDocument();
  expect(screen.getByText('0 to skip')).toBeInTheDocument();
});

test('update is only offered for a duplicate, and worded as restore for a deleted one', async () => {
  renderDialog();
  await toPreview([
    row(0, 'Anna'),
    row(1, 'Ben', { suggested_action: 'skip', duplicate_match: duplicate() }),
    row(2, 'Cara', { suggested_action: 'skip', duplicate_match: duplicate(true) }),
  ]);

  const options = async (name: RegExp) => {
    fireEvent.mouseDown(within(rowOf(name)).getByRole('combobox'));
    const names = within(screen.getByRole('listbox')).getAllByRole('option').map((o) => o.textContent);
    fireEvent.keyDown(screen.getByRole('listbox'), { key: 'Escape' });
    await waitFor(() => expect(screen.queryByRole('listbox')).not.toBeInTheDocument());
    return names;
  };

  expect(await options(/Anna/)).toEqual(['Skip', 'Add as New']);
  expect(await options(/Ben/)).toEqual(['Skip', 'Add as New', 'Update Existing']);
  expect(await options(/Cara/)).toEqual(['Skip', 'Add as New', 'Restore Deleted Contact']);
  expect(within(rowOf(/Ben/)).getByText('Duplicate')).toBeInTheDocument();
  expect(within(rowOf(/Cara/)).getByText('Deleted contact')).toBeInTheDocument();
});

test('a row with validation errors is counted, flagged and cannot be changed', async () => {
  renderDialog();
  await toPreview([row(0, 'Anna', { validation_errors: ['bad email'], suggested_action: 'skip' })]);
  expect(screen.getByText('1 with errors')).toBeInTheDocument();
  const anna = within(rowOf(/Anna/));
  expect(anna.getByText('Error')).toBeInTheDocument();
  expect(anna.getByRole('combobox')).toHaveAttribute('aria-disabled', 'true');
});

test('missing parsed values show a dash; rows are numbered from one', async () => {
  renderDialog();
  await toPreview([{ ...row(4, ''), parsed_contact: {} }]);
  const cells = within(screen.getAllByRole('row')[1]).getAllByRole('cell');
  expect(cells.slice(0, 4).map((c) => c.textContent)).toEqual(['5', '-', '-', '-']);
});

test('confirming a CSV import sends every row action and shows the result', async () => {
  mocked.confirmImport.mockResolvedValue(result({ created: 1, updated: 1, skipped: 1 }));
  const p = renderDialog();
  await toPreview([
    row(0, 'Anna'),
    row(1, 'Ben', { suggested_action: 'update', duplicate_match: duplicate() }),
    row(2, 'Cara', { suggested_action: 'skip' }),
  ]);
  click('Import');
  expect(await screen.findByText('Import Complete')).toBeInTheDocument();
  expect(mocked.confirmImport).toHaveBeenCalledWith('csv-session', [
    { row_index: 0, action: 'add' },
    { row_index: 1, action: 'update' },
    { row_index: 2, action: 'skip' },
  ]);
  expect(mocked.confirmVCFImport).not.toHaveBeenCalled();
  expect(screen.getByText('1 contacts created')).toBeInTheDocument();
  expect(screen.getByText('1 contacts updated')).toBeInTheDocument();
  expect(screen.getByText('1 rows skipped')).toBeInTheDocument();
  expect(p.onImportComplete).toHaveBeenCalledTimes(1);
  expect(snackbar.showSuccess).toHaveBeenCalledWith('Import completed: 1 created, 1 updated');
});

test('a changed row action is what gets confirmed', async () => {
  mocked.confirmImport.mockResolvedValue(result({ created: 0, skipped: 1 }));
  renderDialog();
  await toPreview([row(0, 'Anna')]);
  await choose(within(rowOf(/Anna/)).getByRole('combobox'), 'Skip');
  click('Import');
  await screen.findByText('Import Complete');
  expect(mocked.confirmImport).toHaveBeenCalledWith('csv-session', [{ row_index: 0, action: 'skip' }]);
});

test('an import that changed nothing does not refresh the parent, and lists its errors', async () => {
  mocked.confirmImport.mockResolvedValue(result({ created: 0, skipped: 1, errors: ['row 1 failed'] }));
  const p = renderDialog();
  await toPreview([row(0, 'Anna')]);
  click('Import');
  expect(await screen.findByText('row 1 failed')).toBeInTheDocument();
  expect(p.onImportComplete).not.toHaveBeenCalled();
  expect(snackbar.showSuccess).not.toHaveBeenCalled();
});

test('a failed confirm shows the error and stays on the preview', async () => {
  mocked.confirmImport.mockRejectedValue(new Error('import down'));
  const p = renderDialog();
  await toPreview([row(0, 'Anna')]);
  click('Import');
  expect(await screen.findByRole('alert')).toHaveTextContent('import down');
  expect(screen.getByRole('columnheader', { name: 'Action' })).toBeInTheDocument();
  expect(p.onImportComplete).not.toHaveBeenCalled();
});

test('a VCF skips the mapping step and confirms through the VCF endpoint', async () => {
  mocked.uploadVCFForImport.mockResolvedValue(preview([row(0, 'Anna')], 'vcf-session'));
  mocked.confirmVCFImport.mockResolvedValue(result());
  renderDialog();
  selectFile(file('people.vcf'));
  await screen.findByRole('columnheader', { name: 'Action' });
  expect(screen.queryByText('Map Columns')).not.toBeInTheDocument();
  expect(mocked.uploadCSVForImport).not.toHaveBeenCalled();
  click('Import');
  await screen.findByText('Import Complete');
  expect(mocked.confirmVCFImport).toHaveBeenCalledWith('vcf-session', [{ row_index: 0, action: 'add' }]);
  expect(mocked.confirmImport).not.toHaveBeenCalled();
});

test('back from a VCF preview returns to the upload step', async () => {
  mocked.uploadVCFForImport.mockResolvedValue(preview([row(0, 'Anna')], 'vcf-session'));
  renderDialog();
  selectFile(file('people.vcf'));
  await screen.findByRole('columnheader', { name: 'Action' });
  click('Back');
  expect(document.querySelector('input[type="file"]')).toBeInTheDocument();
  expect(screen.queryByRole('columnheader', { name: 'Action' })).not.toBeInTheDocument();
});

test('back from a CSV preview returns to the mapping step with the mappings intact', async () => {
  renderDialog();
  await toPreview([row(0, 'Anna')]);
  click('Back');
  expect(within(rowOf(/^Name/)).getByRole('combobox')).toHaveTextContent('First Name');
});

test('cancelling mid-way closes and starts over at the upload step', async () => {
  const p = renderDialog();
  await toPreview([row(0, 'Anna')]);
  click('Cancel');
  expect(p.onClose).toHaveBeenCalled();
  expect(document.querySelector('input[type="file"]')).toBeInTheDocument();
  expect(mocked.confirmImport).not.toHaveBeenCalled();
});

test('done closes the dialog after the result', async () => {
  mocked.confirmImport.mockResolvedValue(result());
  const p = renderDialog();
  await toPreview([row(0, 'Anna')]);
  click('Import');
  await screen.findByText('Import Complete');
  click('Done');
  expect(p.onClose).toHaveBeenCalled();
});
