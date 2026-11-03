import { request } from '@playwright/test';
import { test, expect, LOGGED_OUT } from './fixtures';
import { API_BASE_URL } from './global-setup';

test.describe('API tokens', () => {
  test('a created token works until it is revoked', async ({ page }) => {
    const name = `E2E token ${Date.now()}`;

    await page.goto('/api-tokens');
    await page.getByRole('button', { name: 'Create Token' }).click();
    const createDialog = page.getByRole('dialog');
    await createDialog.getByLabel('Token Name').fill(name);
    await createDialog.getByRole('button', { name: 'Create', exact: true }).click();

    const createdDialog = page.getByRole('dialog').filter({ hasNot: page.getByLabel('Token Name') });
    const tokenField = createdDialog.getByRole('textbox');
    await expect(tokenField).toHaveValue(/^meerkat_/);
    const token = await tokenField.inputValue();
    await createdDialog.getByRole('button', { name: /done/i }).click();

    // A clean context with no cookies: only the token can authenticate it.
    const client = await request.newContext({
      storageState: LOGGED_OUT,
      extraHTTPHeaders: { Authorization: `Bearer ${token}` },
    });
    try {
      expect((await client.get(`${API_BASE_URL}/contacts?limit=1`)).status()).toBe(200);

      const row = page.getByRole('row').filter({ hasText: name });
      await row.getByRole('button', { name: 'Revoke Token' }).click();
      await page.getByRole('dialog').getByRole('button', { name: 'Revoke', exact: true }).click();
      await expect(row.getByText('Revoked')).toBeVisible();

      expect((await client.get(`${API_BASE_URL}/contacts?limit=1`)).status()).toBe(401);
    } finally {
      await client.dispose();
    }
  });
});
