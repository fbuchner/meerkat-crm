import { test, expect } from './fixtures';
import { createTestContact, deleteTestContact } from './fixtures';

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

      // The MUI Selects have no accessible name (no labelId wiring), and the
      // type select is the first of the two (type, then gender). Pick the
      // "Custom type..." option so we don't depend on preset type labels.
      await dialog.getByRole('combobox').first().click();
      await page.getByRole('option', { name: /custom type/i }).click();
      await dialog.getByRole('textbox', { name: /enter custom type/i }).fill('Friend');

      await dialog.getByRole('button', { name: /^save$/i }).click();

      await expect(dialog).toBeHidden();
      await expect(page.getByText(relName)).toBeVisible();
    } finally {
      await deleteTestContact(page.request, contact.ID);
    }
  });
});
