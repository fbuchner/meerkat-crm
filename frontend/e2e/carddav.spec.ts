import { test, expect } from './fixtures';
import { APP_ORIGIN, TEST_USER } from './global-setup';

const basicAuth = (password: string) =>
  `Basic ${Buffer.from(`${TEST_USER.username}:${password}`).toString('base64')}`;

test.describe('CardDAV', () => {
  test('well-known discovery redirects to the CardDAV root', async ({ page }) => {
    const response = await page.request.get(`${APP_ORIGIN}/.well-known/carddav`, { maxRedirects: 0 });
    expect(response.status()).toBeGreaterThanOrEqual(301);
    expect(response.status()).toBeLessThanOrEqual(308);
    expect(response.headers()['location']).toContain('/carddav');
  });

  test('PROPFIND needs valid credentials and then lists the principal', async ({ page }) => {
    const propfind = (authorization: string) =>
      page.request.fetch(`${APP_ORIGIN}/carddav/`, {
        method: 'PROPFIND',
        headers: { Authorization: authorization, Depth: '0', 'Content-Type': 'application/xml' },
        data: '<?xml version="1.0"?><d:propfind xmlns:d="DAV:"><d:prop><d:current-user-principal/></d:prop></d:propfind>',
      });

    expect((await propfind(basicAuth('wrong-password'))).status()).toBe(401);

    const response = await propfind(basicAuth(TEST_USER.password));
    expect(response.status()).toBe(207);
    expect(await response.text()).toContain('current-user-principal');
  });
});
