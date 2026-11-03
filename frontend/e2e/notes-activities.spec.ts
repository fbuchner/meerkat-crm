import { test, expect, LOGGED_OUT, signInAsNewUser, uniqueUser } from './fixtures';

// The standalone pages hold notes with no contact and activities across contacts.
// These lists paginate, so each test gets its own user: items that specs running
// alongside add to the seeded user could push the one under test off the page.
test.use({ storageState: LOGGED_OUT });

test.describe('Standalone notes page', () => {
  test('creates, edits and deletes a note', async ({ page }) => {
    const content = `E2E standalone note ${Date.now()}`;
    const edited = `${content} edited`;

    await signInAsNewUser(page, uniqueUser('e2e_notes'));
    await page.goto('/notes');
    await page.getByRole('button', { name: 'Add Note' }).click();
    let dialog = page.getByRole('dialog');
    await dialog.getByLabel(/content/i).fill(content);
    await dialog.getByRole('button', { name: /^save$/i }).click();
    await expect(dialog).toBeHidden();
    await expect(page.getByText(content)).toBeVisible();

    await page.reload();
    const item = page.getByRole('listitem').filter({ hasText: content });
    await item.getByRole('button', { name: 'Edit' }).click({ force: true });
    dialog = page.getByRole('dialog');
    await dialog.getByLabel(/content/i).fill(edited);
    await dialog.getByRole('button', { name: /^save$/i }).click();
    await expect(dialog).toBeHidden();
    await expect(page.getByText(edited)).toBeVisible();

    await page.reload();
    await page.getByRole('listitem').filter({ hasText: edited }).getByRole('button', { name: 'Edit' }).click({ force: true });
    page.once('dialog', (confirm) => confirm.accept());
    await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click();
    await expect(page.getByText(edited)).toHaveCount(0);

    await page.reload();
    await expect(page.getByRole('heading', { name: 'Notes' })).toBeVisible();
    await expect(page.getByText(edited)).toHaveCount(0);
  });
});

test.describe('Standalone activities page', () => {
  test('creates, edits and deletes an activity', async ({ page }) => {
    const title = `E2E standalone activity ${Date.now()}`;
    const edited = `${title} edited`;

    await signInAsNewUser(page, uniqueUser('e2e_activities'));
    await page.goto('/activities');
    await page.getByRole('button', { name: 'Add Activity' }).click();
    let dialog = page.getByRole('dialog');
    await dialog.getByRole('textbox', { name: 'Title', exact: true }).fill(title);
    await dialog.getByRole('button', { name: /^save$/i }).click();
    await expect(dialog).toBeHidden();
    await expect(page.getByText(title)).toBeVisible();

    await page.reload();
    await page.getByRole('listitem').filter({ hasText: title }).getByRole('button', { name: 'Edit' }).click({ force: true });
    dialog = page.getByRole('dialog');
    await dialog.getByRole('textbox', { name: 'Title', exact: true }).fill(edited);
    await dialog.getByRole('button', { name: /^save$/i }).click();
    await expect(dialog).toBeHidden();
    await expect(page.getByText(edited)).toBeVisible();

    await page.reload();
    await page.getByRole('listitem').filter({ hasText: edited }).getByRole('button', { name: 'Edit' }).click({ force: true });
    page.once('dialog', (confirm) => confirm.accept());
    await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click();
    await expect(page.getByText(edited)).toHaveCount(0);

    await page.reload();
    await expect(page.getByRole('heading', { name: 'Activities' })).toBeVisible();
    await expect(page.getByText(edited)).toHaveCount(0);
  });
});
