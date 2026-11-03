import { test, expect } from './fixtures';
import { createTestContact, deleteTestContact } from './fixtures';
import { API_BASE_URL } from './global-setup';

// Notes and activities both render in the contact's Timeline tab. Each test
// uses a throwaway contact and cleans it up (which cascades the note/activity).
test.describe('Timeline', () => {
  test('should add a note to a contact', async ({ page }) => {
    const contact = await createTestContact(page.request);
    const noteContent = `E2E note ${Date.now()}`;

    try {
      await page.goto(`/contacts/${contact.ID}`);

      // Timeline is the default tab.
      await page.getByRole('button', { name: /add note/i }).click();
      const dialog = page.getByRole('dialog');
      await expect(dialog).toBeVisible();

      await dialog.getByLabel(/content/i).fill(noteContent);
      await dialog.getByRole('button', { name: /^save$/i }).click();

      await expect(dialog).toBeHidden();
      await expect(page.getByText(noteContent)).toBeVisible();

      await page.reload();
      await expect(page.getByText(noteContent)).toBeVisible();
    } finally {
      await deleteTestContact(page.request, contact.ID);
    }
  });

  test('should add an activity to a contact', async ({ page }) => {
    const contact = await createTestContact(page.request);
    const activityTitle = `E2E activity ${Date.now()}`;

    try {
      await page.goto(`/contacts/${contact.ID}`);

      await page.getByRole('button', { name: /add activity/i }).click();
      const dialog = page.getByRole('dialog');
      await expect(dialog).toBeVisible();

      await dialog.getByRole('textbox', { name: 'Title', exact: true }).fill(activityTitle);
      await dialog.getByRole('button', { name: /^save$/i }).click();

      await expect(dialog).toBeHidden();
      await expect(page.getByText(activityTitle)).toBeVisible();

      await page.reload();
      await expect(page.getByText(activityTitle)).toBeVisible();
    } finally {
      await deleteTestContact(page.request, contact.ID);
    }
  });

  test('should edit and delete a note from the timeline', async ({ page }) => {
    const contact = await createTestContact(page.request);
    const content = `E2E timeline note ${Date.now()}`;
    const created = await page.request.post(`${API_BASE_URL}/contacts/${contact.ID}/notes`, {
      data: { content, date: new Date().toISOString() },
    });
    expect(created.ok(), `create failed: ${created.status()} ${await created.text()}`).toBeTruthy();

    try {
      await page.goto(`/contacts/${contact.ID}`);
      await page.getByRole('listitem').filter({ hasText: content }).getByRole('button', { name: 'Edit' }).click({ force: true });
      const dialog = page.getByRole('dialog');
      await dialog.getByLabel(/content/i).fill(`${content} edited`);
      await dialog.getByRole('button', { name: /^save$/i }).click();
      await expect(dialog).toBeHidden();

      await page.reload();
      await expect(page.getByText(`${content} edited`)).toBeVisible();

      await page.getByRole('listitem').filter({ hasText: content }).getByRole('button', { name: 'Edit' }).click({ force: true });
      page.once('dialog', (confirm) => confirm.accept());
      await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click();
      await expect(page.getByText(content)).toHaveCount(0);

      await page.reload();
      await expect(page.getByRole('tab', { name: /reminders/i })).toBeVisible();
      await expect(page.getByText(content)).toHaveCount(0);
    } finally {
      await deleteTestContact(page.request, contact.ID);
    }
  });

  test('should edit and delete an activity from the timeline', async ({ page }) => {
    const contact = await createTestContact(page.request);
    const title = `E2E timeline activity ${Date.now()}`;
    const created = await page.request.post(`${API_BASE_URL}/activities`, {
      data: { title, date: new Date().toISOString(), contact_ids: [contact.ID] },
    });
    expect(created.ok(), `create failed: ${created.status()} ${await created.text()}`).toBeTruthy();

    try {
      await page.goto(`/contacts/${contact.ID}`);
      await page.getByRole('listitem').filter({ hasText: title }).getByRole('button', { name: 'Edit' }).click({ force: true });
      const dialog = page.getByRole('dialog');
      await dialog.getByRole('textbox', { name: 'Title', exact: true }).fill(`${title} edited`);
      await dialog.getByRole('button', { name: /^save$/i }).click();
      await expect(dialog).toBeHidden();

      await page.reload();
      await expect(page.getByText(`${title} edited`)).toBeVisible();

      await page.getByRole('listitem').filter({ hasText: title }).getByRole('button', { name: 'Edit' }).click({ force: true });
      page.once('dialog', (confirm) => confirm.accept());
      await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click();
      await expect(page.getByText(title)).toHaveCount(0);

      await page.reload();
      await expect(page.getByRole('tab', { name: /reminders/i })).toBeVisible();
      await expect(page.getByText(title)).toHaveCount(0);
    } finally {
      await deleteTestContact(page.request, contact.ID);
    }
  });
});
