import { test, expect } from './fixtures';
import { waitForLoading, searchContact, createTestContact, deleteTestContact } from './fixtures';
import { API_BASE_URL } from './global-setup';

// Authenticated via the shared storageState (see playwright.config.ts) — no
// per-test login needed.
test.describe('Contacts', () => {
  test('should display contacts page with filter controls', async ({ page }) => {
    await page.goto('/contacts');

    await expect(page.getByRole('heading', { name: /contacts/i })).toBeVisible();
    await expect(page.getByLabel(/filter by circle/i)).toBeVisible();
    await expect(page.getByLabel(/sort by/i)).toBeVisible();
    await expect(page.getByRole('button', { name: /add/i })).toBeVisible();
  });

  test('should find seeded contacts via search', async ({ page }) => {
    await page.goto('/contacts');
    await waitForLoading(page);

    await searchContact(page, 'Alice');
    await expect(page.getByText('Alice Johnson')).toBeVisible();

    await searchContact(page, 'Bob');
    await expect(page.getByText('Bob Smith')).toBeVisible();
  });

  test('should navigate to contact detail', async ({ page }) => {
    await page.goto('/contacts');
    await waitForLoading(page);

    await searchContact(page, 'Alice');
    await page.getByText('Alice Johnson').click();

    await expect(page).toHaveURL(/\/contacts\/\d+/);
    await expect(page.getByText('alice@example.com')).toBeVisible();
  });

  test('should create a new contact', async ({ page }) => {
    // Unique name per run so repeated runs don't accumulate duplicates.
    const firstname = `E2EFixture${Date.now()}`;
    const lastname = 'Contact';

    await page.goto('/contacts');
    await page.getByRole('button', { name: /add/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible();

    await page.getByLabel(/first.*name/i).fill(firstname);
    await page.getByLabel(/last.*name/i).fill(lastname);
    await page.getByRole('button', { name: /create/i }).click();

    // Lands on the new contact's detail page.
    await expect(page).toHaveURL(/\/contacts\/\d+/);
    await expect(page.getByRole('heading', { name: `${firstname} ${lastname}` })).toBeVisible();

    // Clean up so runs stay idempotent.
    const contactId = page.url().match(/\/contacts\/(\d+)/)?.[1];
    if (contactId) {
      await deleteTestContact(page.request, contactId);
    }
  });

  test('should edit a contact name', async ({ page }) => {
    const contact = await createTestContact(page.request, { lastname: 'Before' });

    try {
      await page.goto(`/contacts/${contact.ID}`);
      await expect(page.getByRole('heading', { name: `${contact.firstname} Before` })).toBeVisible();

      // Enter profile edit mode via the (hover-revealed) pencil next to the
      // name. Use the .edit-icon class — MUI strips icon data-testids from
      // production builds, but className survives. The name pencil is first.
      await page.locator('.edit-icon').first().click();

      const lastName = page.getByLabel('Last Name', { exact: true });
      await lastName.fill('After');
      // Save is the only primary-coloured icon button in the header card.
      await page.locator('.MuiCard-root').first().locator('.MuiIconButton-colorPrimary').click();

      await expect(page.getByRole('heading', { name: `${contact.firstname} After` })).toBeVisible();
    } finally {
      await deleteTestContact(page.request, contact.ID);
    }
  });

  test('should delete a contact from the detail page', async ({ page }) => {
    const contact = await createTestContact(page.request, { lastname: 'ToDelete' });

    await page.goto(`/contacts/${contact.ID}`);
    await expect(page.getByRole('heading', { name: `${contact.firstname} ToDelete` })).toBeVisible();

    // Deletion is confirmed via a native confirm() dialog.
    page.once('dialog', (dialog) => dialog.accept());

    // The delete button only appears inside profile edit mode (entered via the
    // name pencil). The delete button itself has a title, so it's role-addressable.
    await page.locator('.edit-icon').first().click();
    await page.getByRole('button', { name: /delete contact/i }).click();

    // Redirects back to the list, and the contact is gone.
    await expect(page).toHaveURL(/\/contacts$/);

    const lookup = await page.request.get(`${API_BASE_URL}/contacts/${contact.ID}`);
    expect(lookup.status()).toBe(404);
  });

  test('should add multiple pronoun entries with different languages and persist them', async ({ page }) => {
    const contact = await createTestContact(page.request, { lastname: 'Pronouns' });

    try {
      await page.goto(`/contacts/${contact.ID}`);
      await expect(page.getByRole('heading', { name: `${contact.firstname} Pronouns` })).toBeVisible();

      // The field row is the caption's grandparent: caption -> label Box -> row Box (icon, label Box, Edit button).
      const pronounsRow = page.getByText('Pronouns', { exact: true }).locator('xpath=../..');
      await pronounsRow.getByRole('button', { name: 'Edit' }).click();

      // Rows start empty; each "Add" click creates one before it can be filled in.
      await page.getByRole('button', { name: /^add$/i }).last().click();
      await page.getByLabel('Language', { exact: false }).first().fill('en');
      await page.getByLabel('Pronouns', { exact: true }).first().fill('they/them');

      await page.getByRole('button', { name: /^add$/i }).last().click();
      await page.getByLabel('Language', { exact: false }).nth(1).fill('de');
      await page.getByLabel('Pronouns', { exact: true }).nth(1).fill('sie/ihr');

      await page.getByRole('button', { name: /save/i }).last().click();

      await expect(page.getByText('they/them (en)')).toBeVisible();
      await expect(page.getByText('sie/ihr (de)')).toBeVisible();

      await page.reload();
      await expect(page.getByText('they/them (en)')).toBeVisible();
      await expect(page.getByText('sie/ihr (de)')).toBeVisible();
    } finally {
      await deleteTestContact(page.request, contact.ID);
    }
  });

  test('should only allow selecting valid GramGender enum values', async ({ page }) => {
    const contact = await createTestContact(page.request, { lastname: 'GramGender' });

    try {
      // GramGender is opt-in via Settings; enable it directly via the API for this test.
      await page.request.patch(`${API_BASE_URL}/users/enabled-contact-fields`, {
        data: { fields: ['emails', 'phones', 'addresses', 'nickname', 'gender', 'pronouns', 'gram_gender', 'birthday', 'how_we_met', 'food_preference', 'work_information', 'contact_information'] },
      });

      await page.goto(`/contacts/${contact.ID}`);
      await expect(page.getByRole('heading', { name: `${contact.firstname} GramGender` })).toBeVisible();

      const gramGenderRow = page.getByText('Grammatical Gender', { exact: true }).locator('xpath=../..');
      await gramGenderRow.getByRole('button', { name: 'Edit' }).click();

      await page.getByRole('button', { name: /^add$/i }).last().click();
      const valueSelect = page.getByLabel('Grammatical Gender', { exact: true }).last();
      await valueSelect.click();

      // Only the 6 spec-core RFC 9554 values should be offered.
      for (const option of ['Animate', 'Common', 'Feminine', 'Inanimate', 'Masculine', 'Neuter']) {
        await expect(page.getByRole('option', { name: option, exact: true })).toBeVisible();
      }
      await expect(page.getByRole('option', { name: 'Robot' })).toHaveCount(0);

      await page.getByRole('option', { name: 'Feminine' }).click();
      await page.getByRole('button', { name: /save/i }).last().click();

      await expect(page.getByText('Feminine', { exact: true })).toBeVisible();
    } finally {
      await deleteTestContact(page.request, contact.ID);
    }
  });
});
