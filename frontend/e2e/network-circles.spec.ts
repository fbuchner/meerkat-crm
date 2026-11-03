import { test, expect, LOGGED_OUT, signInAsNewUser, uniqueUser, createTestContact } from './fixtures';

test.describe('Contact circles', () => {
  test('the circle filter limits the contact list', async ({ page }) => {
    await page.goto('/contacts');
    await page.getByRole('combobox', { name: /filter by circle/i }).click();
    await page.getByRole('option', { name: 'Family', exact: true }).click();

    // Seeded: Bob is in Family, Carol is only in Friends.
    await expect(page.getByText('Bob Smith')).toBeVisible();
    await expect(page.getByText('Carol Williams')).toHaveCount(0);
  });
});

test.describe('Network graph', () => {
  test.use({ storageState: LOGGED_OUT });

  test('shows the graph for a user with contacts, and a message without', async ({ page }) => {
    await signInAsNewUser(page, uniqueUser('e2e_network'));
    await page.goto('/network');
    await expect(page.getByRole('alert')).toBeVisible();
    await expect(page.locator('canvas')).toHaveCount(0);

    await createTestContact(page.request, { firstname: 'Graphed', lastname: 'Person', circles: ['Friends'] });
    await page.reload();
    await expect(page.getByRole('heading', { name: 'Contact Network' })).toBeVisible();
    await expect(page.locator('canvas').first()).toBeVisible();
  });
});
