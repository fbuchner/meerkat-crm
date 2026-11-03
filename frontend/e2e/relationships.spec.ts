import { test, expect } from './fixtures';
import { createTestContact, deleteTestContact } from './fixtures';
import { API_BASE_URL } from './global-setup';

// Relationships live on a tab inside the contact information card. Each test
// uses a throwaway contact and cleans it up afterwards.
test.describe('Relationships', () => {
  test('should add a manual relationship to a contact', async ({ page }) => {
    const contact = await createTestContact(page.request);
    const relName = `E2E Rel ${Date.now()}`;

    try {
      await page.goto(`/contacts/${contact.ID}`);

      // Switch to the Relationships tab and open the add dialog.
      await page.getByRole('tab', { name: /relationships/i }).click();
      await page.getByRole('button', { name: /add relationship/i }).click();

      const dialog = page.getByRole('dialog');
      await expect(dialog).toBeVisible();

      // Manual entry is the default mode.
      await dialog.getByRole('textbox', { name: 'Name', exact: true }).fill(relName);

      // Pick the "Custom type..." option so we don't depend on preset type labels.
      await dialog.getByRole('combobox', { name: 'Type' }).click();
      await page.getByRole('option', { name: /custom type/i }).click();
      await dialog.getByRole('textbox', { name: /enter custom type/i }).fill('Friend');

      await dialog.getByRole('button', { name: /^save$/i }).click();

      await expect(dialog).toBeHidden();
      await expect(page.getByText(relName)).toBeVisible();

      await page.reload();
      await page.getByRole('tab', { name: /relationships/i }).click();
      await expect(page.getByText(relName)).toBeVisible();
    } finally {
      await deleteTestContact(page.request, contact.ID);
    }
  });

  test('should edit and delete a relationship', async ({ page }) => {
    const contact = await createTestContact(page.request);
    const name = `E2E Rel ${Date.now()}`;
    const created = await page.request.post(`${API_BASE_URL}/contacts/${contact.ID}/relationships`, {
      data: { name, type: 'Friend' },
    });
    expect(created.ok()).toBeTruthy();

    try {
      await page.goto(`/contacts/${contact.ID}`);
      await page.getByRole('tab', { name: /relationships/i }).click();
      await page.getByRole('button', { name: 'Edit', exact: true }).click();
      const dialog = page.getByRole('dialog');
      await dialog.getByRole('textbox', { name: 'Name', exact: true }).fill(`${name} edited`);
      await dialog.getByRole('button', { name: /^save$/i }).click();
      await expect(dialog).toBeHidden();

      await page.reload();
      await page.getByRole('tab', { name: /relationships/i }).click();
      await expect(page.getByText(`${name} edited`)).toBeVisible();

      page.once('dialog', (confirm) => confirm.accept());
      await page.getByRole('button', { name: 'Delete', exact: true }).click();
      await expect(page.getByText(name)).toHaveCount(0);

      await page.reload();
      await page.getByRole('tab', { name: /relationships/i }).click();
      await expect(page.getByText(name)).toHaveCount(0);
    } finally {
      await deleteTestContact(page.request, contact.ID);
    }
  });
});
