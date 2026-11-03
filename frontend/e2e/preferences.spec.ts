import { test, expect, LOGGED_OUT, signInAsNewUser, uniqueUser, createTestContact } from './fixtures';

// Preferences belong to the user, so each test gets its own rather than
// changing the seeded user under the specs running alongside it.
test.use({ storageState: LOGGED_OUT });

test.describe('Preferences', () => {
  test('the date format preference changes dates and survives a reload', async ({ page }) => {
    await signInAsNewUser(page, uniqueUser('e2e_datefmt'));
    const contact = await createTestContact(page.request, { birthday: '1990-03-15' });

    await page.goto('/settings');
    await page.getByRole('combobox', { name: 'Date Format' }).click();
    await page.getByRole('option', { name: /^US/ }).click();
    await page.goto(`/contacts/${contact.ID}`);
    await expect(page.getByText(/03\/15\/1990/)).toBeVisible();

    await page.goto('/settings');
    await page.getByRole('combobox', { name: 'Date Format' }).click();
    await page.getByRole('option', { name: /^ISO/ }).click();
    await page.goto(`/contacts/${contact.ID}`);
    await expect(page.getByText(/1990-03-15/)).toBeVisible();
    await expect(page.getByText(/03\/15\/1990/)).toHaveCount(0);

    await page.reload();
    await expect(page.getByText(/1990-03-15/)).toBeVisible();
  });

  test('switching language changes the text and survives a reload', async ({ page }) => {
    await signInAsNewUser(page, uniqueUser('e2e_lang'));

    await page.goto('/settings');
    await page.getByRole('combobox', { name: 'Language' }).click();
    await page.getByRole('option', { name: 'Deutsch' }).click();
    await expect(page.getByRole('heading', { name: 'Einstellungen' })).toBeVisible();

    await page.reload();
    await expect(page.getByRole('heading', { name: 'Einstellungen' })).toBeVisible();
  });
});
