import { expect, test } from 'vitest';
import { CONTACT_DETAIL_COLUMNS } from './ContactDetailPage';
import contactController from '../../backend/controllers/contact_controller.go?raw';

// Columns the backend may serve that the detail page deliberately never reads.
const BACKEND_ONLY = ['photo_thumbnail'];

function parseBackendColumns(): string[] {
  const body = contactController.match(/var contactSelectableFields = \[\]string\{([\s\S]*?)\n\}/)?.[1];
  if (!body) throw new Error('contactSelectableFields not found in contact_controller.go');
  return [...body.matchAll(/"([^"]+)"/g)].map((match) => match[1]);
}

// A column missing from the frontend list reads back as a zero value, overwrites
// local state, and the next save persists the wipe (see commit 348a46f).
test('the detail page and the backend agree on selectable contact columns', () => {
  const backend = parseBackendColumns().filter((column) => !BACKEND_ONLY.includes(column));

  expect([...CONTACT_DETAIL_COLUMNS].sort()).toEqual(backend.sort());
});
