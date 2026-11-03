import { test, expect, createTestContact, deleteTestContact } from './fixtures';
import { API_BASE_URL } from './global-setup';
import type { Page } from '@playwright/test';

// A real image large enough for the cropper to work with.
async function solidPng(page: Page): Promise<Buffer> {
  const dataUrl = await page.evaluate(() => {
    const canvas = document.createElement('canvas');
    canvas.width = canvas.height = 400;
    const context = canvas.getContext('2d') as CanvasRenderingContext2D;
    context.fillStyle = '#666';
    context.fillRect(0, 0, 400, 400);
    return canvas.toDataURL('image/png');
  });
  return Buffer.from(dataUrl.split(',')[1], 'base64');
}

test.describe('Profile photo', () => {
  test('an uploaded photo is still shown after a reload', async ({ page }) => {
    const contact = await createTestContact(page.request);

    try {
      await page.goto(`/contacts/${contact.ID}`);
      await page.getByRole('button', { name: 'Upload Profile Picture' }).click({ force: true });
      const dialog = page.getByRole('dialog');
      await dialog
        .locator('input[type="file"]')
        .setInputFiles({ name: 'photo.png', mimeType: 'image/png', buffer: await solidPng(page) });
      const uploaded = page.waitForResponse(
        (r) => r.request().method() === 'POST' && r.url().endsWith(`/contacts/${contact.ID}/profile_picture`)
      );
      await dialog.getByRole('button', { name: 'Save', exact: true }).click();
      expect((await uploaded).ok()).toBeTruthy();
      await expect(dialog).toBeHidden();

      await page.reload();
      const avatar = page.getByRole('img', { name: contact.firstname, exact: false }).first();
      await expect(avatar).toBeVisible();
      await expect.poll(() => avatar.evaluate((img: HTMLImageElement) => img.naturalWidth)).toBeGreaterThan(0);

      const stored = await page.request.get(`${API_BASE_URL}/contacts/${contact.ID}/profile_picture`);
      expect(stored.status()).toBe(200);
      expect(stored.headers()['content-type']).toMatch(/^image\//);
    } finally {
      await deleteTestContact(page.request, contact.ID);
    }
  });
});
