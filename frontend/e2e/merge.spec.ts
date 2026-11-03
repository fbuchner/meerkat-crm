import { test, expect, createTestContact, deleteTestContact, itemWithButton } from './fixtures';
import { API_BASE_URL } from './global-setup';

test.describe('Merge duplicates', () => {
  test('merging combines the data and removes one contact', async ({ page }) => {
    const firstname = `E2EFixtureMerge${Date.now()}`;
    const first = await createTestContact(page.request, { firstname, lastname: 'Twin' });
    const second = await createTestContact(page.request, { firstname, lastname: 'Twin' });
    const noteContent = `E2E merge note ${Date.now()}`;

    try {
      for (const contact of [first, second]) {
        const note = await page.request.post(`${API_BASE_URL}/contacts/${contact.ID}/notes`, {
          data: { content: `${noteContent} ${contact.ID}`, date: new Date().toISOString() },
        });
        expect(note.ok()).toBeTruthy();
      }

      await page.goto('/contacts/duplicates');
      const pair = itemWithButton(page, firstname, 'Merge');
      await pair.getByRole('button', { name: 'Merge', exact: true }).click();
      const dialog = page.getByRole('dialog');
      await dialog.getByRole('button', { name: 'Merge contacts' }).click();
      await expect(dialog).toBeHidden();
      await expect(page.getByText(firstname)).toHaveCount(0);

      const [firstStatus, secondStatus] = await Promise.all(
        [first, second].map(async (c) => (await page.request.get(`${API_BASE_URL}/contacts/${c.ID}`)).status())
      );
      expect([firstStatus, secondStatus].sort()).toEqual([200, 404]);

      const survivor = firstStatus === 200 ? first : second;
      const notes = await (await page.request.get(`${API_BASE_URL}/contacts/${survivor.ID}/notes`)).json();
      const contents = (notes.notes ?? notes).map((n: { content: string }) => n.content);
      expect(contents.sort()).toEqual([`${noteContent} ${first.ID}`, `${noteContent} ${second.ID}`].sort());
    } finally {
      await deleteTestContact(page.request, first.ID);
      await deleteTestContact(page.request, second.ID);
    }
  });
});
