import { test, expect, LOGGED_OUT, signInAsNewUser, uniqueUser } from './fixtures';
import { API_BASE_URL } from './global-setup';

test.use({ storageState: LOGGED_OUT });

test.describe('Webhooks', () => {
  test('creates a webhook, sends a test and shows the delivery', async ({ page }) => {
    await signInAsNewUser(page, uniqueUser('e2e_webhook'));
    const name = `E2E hook ${Date.now()}`;

    await page.goto('/api-tokens');
    await page.getByRole('button', { name: 'Add Webhook' }).click();
    const dialog = page.getByRole('dialog');
    await dialog.getByRole('textbox', { name: 'Name' }).fill(name);
    // Nothing listens here, so the test stays offline; the failure is still recorded.
    await dialog.getByRole('textbox', { name: 'URL' }).fill('http://127.0.0.1:9/hook');
    await dialog.getByRole('combobox', { name: 'Events' }).click();
    await page.getByRole('option').first().click();
    await page.keyboard.press('Escape');
    await dialog.getByRole('button', { name: 'Save' }).click();

    const secretDialog = page.getByRole('dialog').filter({ hasText: 'Webhook Secret' });
    await expect(secretDialog.getByRole('textbox')).not.toHaveValue('');
    await secretDialog.getByRole('button', { name: /done/i }).click();
    await expect(page.getByText(name)).toBeVisible();

    const tested = page.waitForResponse((r) => /\/webhooks\/\d+\/test$/.test(r.url()));
    await page.getByRole('button', { name: 'Test', exact: true }).click();
    expect((await tested).status()).toBe(200);

    const webhooks = await (await page.request.get(`${API_BASE_URL}/webhooks`)).json();
    const webhook = (webhooks.webhooks ?? webhooks)[0];
    const deliveries = await (await page.request.get(`${API_BASE_URL}/webhooks/${webhook.id}/deliveries`)).json();
    expect(deliveries.deliveries).toHaveLength(1);

    // The refused connection has no status code, so the delivery shows as "err".
    await expect(page.getByText('err', { exact: true })).toBeVisible();

    await page.reload();
    await page.getByRole('button', { name: 'Recent Deliveries' }).click();
    await expect(page.getByText('err', { exact: true })).toBeVisible();
  });
});
