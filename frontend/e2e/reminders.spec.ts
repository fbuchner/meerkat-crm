import { test, expect } from './fixtures';
import { createTestContact, deleteTestContact, itemWithButton } from './fixtures';
import { API_BASE_URL } from './global-setup';

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

      await page.reload();
      await page.getByRole('tab', { name: /reminders/i }).click();
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

  test('should edit and delete a reminder', async ({ page }) => {
    const contact = await createTestContact(page.request);
    const message = `E2E reminder ${Date.now()}`;
    const created = await page.request.post(`${API_BASE_URL}/contacts/${contact.ID}/reminders`, {
      data: { message, remind_at: new Date().toISOString(), recurrence: 'once', contact_id: contact.ID },
    });
    expect(created.ok()).toBeTruthy();

    try {
      await page.goto(`/contacts/${contact.ID}`);
      await page.getByRole('tab', { name: /reminders/i }).click();
      await itemWithButton(page, message, 'Edit').getByRole('button', { name: 'Edit', exact: true }).click({ force: true });
      const dialog = page.getByRole('dialog');
      await dialog.getByRole('textbox', { name: /message/i }).fill(`${message} edited`);
      await dialog.getByRole('button', { name: /save|update/i }).click();
      await expect(dialog).toBeHidden();

      await page.reload();
      await page.getByRole('tab', { name: /reminders/i }).click();
      await expect(page.getByText(`${message} edited`)).toBeVisible();

      await itemWithButton(page, message, 'Delete').getByRole('button', { name: 'Delete', exact: true }).click({ force: true });
      await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click();
      await expect(page.getByText(message)).toHaveCount(0);

      await page.reload();
      await page.getByRole('tab', { name: /reminders/i }).click();
      await expect(page.getByText(message)).toHaveCount(0);
    } finally {
      await deleteTestContact(page.request, contact.ID);
    }
  });

  test('completing a monthly reminder moves it to the next occurrence', async ({ page }) => {
    const contact = await createTestContact(page.request);
    const message = `E2E recurring ${Date.now()}`;
    const remindAt = new Date('2026-01-15T09:00:00Z');
    const created = await page.request.post(`${API_BASE_URL}/contacts/${contact.ID}/reminders`, {
      data: {
        message,
        remind_at: remindAt.toISOString(),
        recurrence: 'monthly',
        reoccur_from_completion: true,
        contact_id: contact.ID,
      },
    });
    expect(created.ok(), await created.text()).toBeTruthy();
    const body = await created.json();
    const reminder = body.reminder ?? body;
    const scheduled = new Date(reminder.remind_at).getTime();

    try {
      await page.goto(`/contacts/${contact.ID}`);
      await page.getByRole('tab', { name: /reminders/i }).click();
      await page.getByRole('button', { name: 'Complete', exact: true }).click();

      // Still listed, now scheduled later, and the completion is on record.
      await expect
        .poll(async () => {
          const body = await (await page.request.get(`${API_BASE_URL}/reminders/${reminder.ID}`)).json();
          return new Date((body.reminder ?? body).remind_at).getTime();
        })
        .toBeGreaterThan(scheduled);

      const completions = await (
        await page.request.get(`${API_BASE_URL}/contacts/${contact.ID}/reminder-completions`)
      ).json();
      expect(completions.completions ?? completions).toHaveLength(1);

      await page.reload();
      await page.getByRole('tab', { name: /reminders/i }).click();
      await expect(page.getByText(message).first()).toBeVisible();
    } finally {
      await deleteTestContact(page.request, contact.ID);
    }
  });
});
