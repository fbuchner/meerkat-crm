import { test, expect } from './fixtures';
import { createTestContact, deleteTestContact } from './fixtures';

// §3d WP3 (docs/fork-plan/95-backlog-and-priorities.md): replaces
// relationships.spec.ts now that the Relationships tab talks to
// RelationshipEdge instead of the legacy models.Relationship. Accept/Reject
// (the suggestion-review flow) is intentionally not covered here -- nothing
// in this app can produce a `status: suggested` edge yet
// (services/household_service.go's GenerateHouseholdSuggestions has no HTTP
// trigger anywhere), so there's no real data this e2e run could exercise
// that path against. See RelationshipEdgeList.test.tsx for that coverage
// instead, via a mocked suggested-status edge.
test.describe('RelationshipEdges', () => {
  test('add a manual (thin-contact) relationship to a contact', async ({ page }) => {
    const contact = await createTestContact(page.request);
    const relName = `E2E Rel ${Date.now()}`;

    try {
      await page.goto(`/contacts/${contact.ID}`);

      await page.getByRole('tab', { name: /relationships/i }).click();
      await page.getByRole('button', { name: /add relationship/i }).click();

      const dialog = page.getByRole('dialog');
      await expect(dialog).toBeVisible();

      // Manual entry is the default mode.
      await dialog.getByRole('textbox', { name: /^name/i }).fill(relName);

      // The Relationship Type Select is the first combobox in this dialog
      // (Gender is second, manual-mode only; Sensitivity is third).
      await dialog.getByRole('combobox').first().click();
      await page.getByRole('option', { name: 'Friend' }).click();

      await dialog.getByRole('button', { name: /^save$/i }).click();

      await expect(dialog).toBeHidden();
      await expect(page.getByText(relName)).toBeVisible();
      await expect(page.getByText('Friend')).toBeVisible();
    } finally {
      await deleteTestContact(page.request, contact.ID);
    }
  });

  test('add a linked-contact relationship, visible with the inverse label from the other contact\'s page', async ({ page }) => {
    const alice = await createTestContact(page.request, { firstname: 'E2E-Alice', lastname: `Rel${Date.now()}` });
    const bob = await createTestContact(page.request, { firstname: 'E2E-Bob', lastname: `Rel${Date.now()}` });

    try {
      await page.goto(`/contacts/${alice.ID}`);
      await page.getByRole('tab', { name: /relationships/i }).click();
      await page.getByRole('button', { name: /add relationship/i }).click();

      const dialog = page.getByRole('dialog');
      await expect(dialog).toBeVisible();

      // Switch to linked-contact mode and select Bob.
      await dialog.getByRole('radio', { name: /link to existing contact/i }).click();
      const autocomplete = dialog.getByRole('combobox', { name: /select contact/i });
      await autocomplete.fill(bob.lastname);
      await page.getByRole('option', { name: new RegExp(bob.lastname) }).click();

      // In linked mode the DOM order is: Autocomplete (combobox #0), then
      // the Relationship Type Select (combobox #1) -- no Gender field shown.
      await dialog.getByRole('combobox').nth(1).click();
      await page.getByRole('option', { name: 'Parent' }).click();

      await dialog.getByRole('button', { name: /^save$/i }).click();
      await expect(dialog).toBeHidden();

      // From Alice's page: Bob is her Parent.
      await expect(page.getByText('E2E-Bob')).toBeVisible();
      await expect(page.getByText('Parent')).toBeVisible();

      // From Bob's page: Alice is his Child (the inverse label).
      await page.goto(`/contacts/${bob.ID}`);
      await page.getByRole('tab', { name: /relationships/i }).click();
      await expect(page.getByText('E2E-Alice')).toBeVisible();
      await expect(page.getByText('Child')).toBeVisible();
    } finally {
      await deleteTestContact(page.request, alice.ID);
      await deleteTestContact(page.request, bob.ID);
    }
  });

  test('edit a relationship\'s type', async ({ page }) => {
    const contact = await createTestContact(page.request);
    const relName = `E2E Edit ${Date.now()}`;

    try {
      await page.goto(`/contacts/${contact.ID}`);
      await page.getByRole('tab', { name: /relationships/i }).click();
      await page.getByRole('button', { name: /add relationship/i }).click();

      const createDialog = page.getByRole('dialog');
      await createDialog.getByRole('textbox', { name: /^name/i }).fill(relName);
      await createDialog.getByRole('combobox').first().click();
      await page.getByRole('option', { name: 'Friend' }).click();
      await createDialog.getByRole('button', { name: /^save$/i }).click();
      await expect(createDialog).toBeHidden();

      // Hover the card to reveal the edit action, then edit the type.
      const card = page.locator('text=' + relName).locator('..').locator('..');
      await card.hover();
      await card.getByLabel('Edit').click();

      const editDialog = page.getByRole('dialog');
      await expect(editDialog).toBeVisible();
      await editDialog.getByRole('combobox').first().click();
      await page.getByRole('option', { name: 'Sibling' }).click();
      await editDialog.getByRole('button', { name: /^save$/i }).click();
      await expect(editDialog).toBeHidden();

      await expect(page.getByText('Sibling')).toBeVisible();
    } finally {
      await deleteTestContact(page.request, contact.ID);
    }
  });

  test('delete a relationship', async ({ page }) => {
    const contact = await createTestContact(page.request);
    const relName = `E2E Delete ${Date.now()}`;

    try {
      await page.goto(`/contacts/${contact.ID}`);
      await page.getByRole('tab', { name: /relationships/i }).click();
      await page.getByRole('button', { name: /add relationship/i }).click();

      const dialog = page.getByRole('dialog');
      await dialog.getByRole('textbox', { name: /^name/i }).fill(relName);
      await dialog.getByRole('combobox').first().click();
      await page.getByRole('option', { name: 'Friend' }).click();
      await dialog.getByRole('button', { name: /^save$/i }).click();
      await expect(dialog).toBeHidden();
      await expect(page.getByText(relName)).toBeVisible();

      page.on('dialog', (d) => d.accept());
      const card = page.locator('text=' + relName).locator('..').locator('..');
      await card.hover();
      await card.getByLabel('Delete').click();

      await expect(page.getByText(relName)).toBeHidden();
    } finally {
      await deleteTestContact(page.request, contact.ID);
    }
  });
});
