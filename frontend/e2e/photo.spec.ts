import { test, expect, createTestContact, deleteTestContact } from './fixtures';
import { API_BASE_URL } from './global-setup';
import { deflateSync } from 'node:zlib';

// A valid solid-colour PNG, large enough for the cropper to work with.
function solidPng(size: number): Buffer {
  const crcTable = Array.from({ length: 256 }, (_, n) => {
    let c = n;
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    return c >>> 0;
  });
  const crc32 = (data: Buffer) => {
    let c = 0xffffffff;
    for (const byte of data) c = crcTable[(c ^ byte) & 0xff] ^ (c >>> 8);
    return (c ^ 0xffffffff) >>> 0;
  };
  const chunk = (type: string, data: Buffer) => {
    const length = Buffer.alloc(4);
    length.writeUInt32BE(data.length);
    const body = Buffer.concat([Buffer.from(type), data]);
    const crc = Buffer.alloc(4);
    crc.writeUInt32BE(crc32(body));
    return Buffer.concat([length, body, crc]);
  };
  const header = Buffer.alloc(13);
  header.writeUInt32BE(size, 0);
  header.writeUInt32BE(size, 4);
  header[8] = 8; // bit depth
  header[9] = 2; // RGB
  const row = Buffer.concat([Buffer.from([0]), Buffer.alloc(size * 3, 0x66)]);
  const pixels = Buffer.concat(Array.from({ length: size }, () => row));
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunk('IHDR', header),
    chunk('IDAT', deflateSync(pixels)),
    chunk('IEND', Buffer.alloc(0)),
  ]);
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
        .setInputFiles({ name: 'photo.png', mimeType: 'image/png', buffer: solidPng(400) });
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
