import { request } from '@playwright/test';
import { test, expect, LOGGED_OUT, loginAsNewUser, loginUser, uniqueUser } from './fixtures';

// docker-compose.test.yml sends mail to Mailpit, which exposes it over HTTP.
const MAILPIT_URL = process.env.MAILPIT_URL ?? 'http://localhost:8025';

test.use({ storageState: LOGGED_OUT });

async function readResetToken(email: string): Promise<string> {
  const mailpit = await request.newContext();
  try {
    let messageId = '';
    await expect
      .poll(
        async () => {
          const search = await mailpit.get(`${MAILPIT_URL}/api/v1/search`, {
            params: { query: `to:"${email}"` },
          });
          messageId = (await search.json()).messages?.[0]?.ID ?? '';
          return messageId;
        },
        { timeout: 15000 }
      )
      .not.toBe('');
    const message = await (await mailpit.get(`${MAILPIT_URL}/api/v1/message/${messageId}`)).json();
    // The token sits on its own line, and base64url can start or end with a dash.
    const token = String(message.Text).match(/^\s*([A-Za-z0-9_-]{20,})\s*$/m)?.[1];
    expect(token, `no token in mail: ${message.Text}`).toBeTruthy();
    return token as string;
  } finally {
    await mailpit.dispose();
  }
}

test.describe('Password reset', () => {
  test('a mailed token sets a new password', async ({ page }) => {
    const user = uniqueUser('e2e_reset');
    const context = await loginAsNewUser(user);
    await context.dispose();
    const newPassword = 'E2eResetPass456!';

    await page.goto('/');
    await page.getByRole('button', { name: /forgot/i }).click();
    const dialog = page.getByRole('dialog');
    await dialog.getByRole('textbox', { name: /^Email/ }).fill(user.email);
    await dialog.getByRole('button', { name: 'Send reset email' }).click();

    await dialog.getByRole('textbox', { name: /^Reset token/ }).fill(await readResetToken(user.email));
    await dialog.getByRole('textbox', { name: /^New password/ }).fill(newPassword);
    await dialog.getByRole('textbox', { name: /^Confirm new password/ }).fill(newPassword);
    await dialog.getByRole('button', { name: 'Reset password' }).click();
    await expect(dialog.getByRole('alert').filter({ hasText: /password reset successful/i })).toBeVisible();
    await dialog.getByRole('button', { name: 'Return to login' }).click();

    await loginUser(page, { ...user, password: newPassword });
  });
});
