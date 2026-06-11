import { test, expect } from './fixtures';
import { createTestContact, deleteTestContact } from './fixtures';

// Authenticated via the shared storageState (see playwright.config.ts).
// Each test runs against its own throwaway contact
test.describe('Reminders', () => {
  test('should create a reminder from a contact detail page', async ({ page }) => {
    const contact = await createTestContact(page.request);

    try {
      await page.goto(`/contacts/${contact.ID}`);

      // Switch from the Timeline tab to the Reminders tab.
      await page.getByRole('tab', { name: /reminders/i }).click();

      await page.getByRole('button', { name: /add.*reminder/i }).click();
      await expect(page.getByRole('dialog')).toBeVisible();

      await page.getByRole('textbox', { name: /message/i }).fill('E2E Test Reminder');
      await page.getByRole('button', { name: /save|create/i }).click();

      // Dialog closes and the reminder shows up in the list.
      await expect(page.getByRole('dialog')).toBeHidden();
      await expect(page.getByText('E2E Test Reminder')).toBeVisible();
    } finally {
      await deleteTestContact(page.request, contact.ID);
    }
  });

  test('should show reminder form fields', async ({ page }) => {
    const contact = await createTestContact(page.request);

    try {
      await page.goto(`/contacts/${contact.ID}`);
      await page.getByRole('tab', { name: /reminders/i }).click();

      await page.getByRole('button', { name: /add.*reminder/i }).click();
      await expect(page.getByRole('dialog')).toBeVisible();

      await expect(page.getByRole('textbox', { name: /message/i })).toBeVisible();
      await expect(page.getByLabel(/date/i).first()).toBeVisible();

      await page.keyboard.press('Escape');
      await expect(page.getByRole('dialog')).toBeHidden();
    } finally {
      await deleteTestContact(page.request, contact.ID);
    }
  });
});
