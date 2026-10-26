import type { Page } from '@playwright/test';
import { test, expect } from './fixtures';
import { createTestContact, deleteTestContact } from './fixtures';

// Employment history lives on its own tab inside the contact information card.
// Each test uses a throwaway contact and cleans it up afterwards.
test.describe('Employment history', () => {
  async function openTab(page: Page, contactId: number) {
    await page.goto(`/contacts/${contactId}`);
    await page.getByRole('tab', { name: /employment history/i }).click();
  }

  async function addJob(
    page: Page,
    fields: { organization: string; jobTitle?: string; start?: string; end?: string }
  ) {
    await page.getByRole('button', { name: /add job/i }).click();
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();
    await dialog.getByRole('textbox', { name: /organization/i }).fill(fields.organization);
    if (fields.jobTitle) await dialog.getByRole('textbox', { name: /job title/i }).fill(fields.jobTitle);
    if (fields.start) await dialog.getByRole('textbox', { name: /start date/i }).fill(fields.start);
    if (fields.end) await dialog.getByRole('textbox', { name: /end date/i }).fill(fields.end);
    await dialog.getByRole('button', { name: /^save$/i }).click();
    return dialog;
  }

  test('adds a current job and shows it with the Current chip', async ({ page }) => {
    const contact = await createTestContact(page.request);
    const org = `E2E Org ${Date.now()}`;

    try {
      await openTab(page, contact.ID);
      await expect(page.getByText(/no employment history yet/i)).toBeVisible();

      const dialog = await addJob(page, { organization: org, jobTitle: 'Engineer', start: '2020-01' });

      await expect(dialog).toBeHidden();
      await expect(page.getByText(org)).toBeVisible();
      await expect(page.getByText('Current', { exact: true })).toBeVisible();
    } finally {
      await deleteTestContact(page.request, contact.ID);
    }
  });

  test('rejects an empty organization and a malformed date', async ({ page }) => {
    const contact = await createTestContact(page.request);

    try {
      await openTab(page, contact.ID);

      await page.getByRole('button', { name: /add job/i }).click();
      const dialog = page.getByRole('dialog');
      await dialog.getByRole('button', { name: /^save$/i }).click();
      await expect(dialog.getByText(/organization is required/i)).toBeVisible();

      await dialog.getByRole('textbox', { name: /organization/i }).fill('Acme');
      await dialog.getByRole('textbox', { name: /start date/i }).fill('01/2020');
      await dialog.getByRole('button', { name: /^save$/i }).click();
      await expect(dialog.getByText(/dates must be yyyy/i)).toBeVisible();
      await expect(dialog).toBeVisible();
    } finally {
      await deleteTestContact(page.request, contact.ID);
    }
  });

  test('edits an entry', async ({ page }) => {
    const contact = await createTestContact(page.request);
    const org = `E2E Edit ${Date.now()}`;

    try {
      await openTab(page, contact.ID);
      await addJob(page, { organization: org, start: '2018' });
      await expect(page.getByText(org)).toBeVisible();

      await page.getByRole('button', { name: /^edit$/i }).click();
      const dialog = page.getByRole('dialog');
      await dialog.getByRole('textbox', { name: /organization/i }).fill(`${org} Renamed`);
      await dialog.getByRole('button', { name: /^save$/i }).click();

      await expect(dialog).toBeHidden();
      await expect(page.getByText(`${org} Renamed`)).toBeVisible();
    } finally {
      await deleteTestContact(page.request, contact.ID);
    }
  });

  test('ends a current job', async ({ page }) => {
    const contact = await createTestContact(page.request);
    const org = `E2E End ${Date.now()}`;

    try {
      await openTab(page, contact.ID);
      await addJob(page, { organization: org, start: '2019' });
      await expect(page.getByText('Current', { exact: true })).toBeVisible();

      // The list uses window.confirm; Playwright dismisses dialogs by default.
      page.once('dialog', (d) => d.accept());
      await page.getByRole('button', { name: /end this job/i }).click();

      await expect(page.getByText('Current', { exact: true })).toBeHidden();
      await expect(page.getByRole('button', { name: /end this job/i })).toBeHidden();
    } finally {
      await deleteTestContact(page.request, contact.ID);
    }
  });

  test('deletes an entry', async ({ page }) => {
    const contact = await createTestContact(page.request);
    const org = `E2E Del ${Date.now()}`;

    try {
      await openTab(page, contact.ID);
      await addJob(page, { organization: org, start: '2015', end: '2017' });
      await expect(page.getByText(org)).toBeVisible();

      page.once('dialog', (d) => d.accept());
      await page.getByRole('button', { name: /^delete$/i }).click();

      await expect(page.getByText(org)).toBeHidden();
      await expect(page.getByText(/no employment history yet/i)).toBeVisible();
    } finally {
      await deleteTestContact(page.request, contact.ID);
    }
  });

  test('keeps entries after reload', async ({ page }) => {
    const contact = await createTestContact(page.request);
    const org = `E2E Persist ${Date.now()}`;

    try {
      await openTab(page, contact.ID);
      await addJob(page, { organization: org, start: '2021-06-15' });
      await expect(page.getByText(org)).toBeVisible();

      await openTab(page, contact.ID);
      await expect(page.getByText(org)).toBeVisible();
    } finally {
      await deleteTestContact(page.request, contact.ID);
    }
  });
});
