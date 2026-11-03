import { test, expect } from './fixtures';

// Pages load on demand. After a release, a tab left open can ask for a page file
// that no longer exists, which the app recovers from by reloading once.
test.describe('Lazy-loaded pages', () => {
  test('reloads and shows the page when its file is stale', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByRole('heading', { name: /dashboard/i })).toBeVisible();

    let blocked = 0;
    await page.route('**/static/ContactsPage-*.js', async (route) => {
      if (blocked++ === 0) {
        await route.abort();
      } else {
        await route.continue();
      }
    });

    await page.getByRole('link', { name: /contacts/i }).click();

    await expect(page.getByRole('heading', { name: /contacts/i })).toBeVisible();
    expect(blocked).toBeGreaterThan(1);
  });

  test('does not reload in a loop when the file stays missing', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByRole('heading', { name: /dashboard/i })).toBeVisible();
    await page.route('**/static/ContactsPage-*.js', (route) => route.abort());

    let loads = 0;
    page.on('load', () => loads++);

    await page.getByRole('link', { name: /contacts/i }).click();

    // The single reload lands on an error screen and stays there.
    await expect(page.getByRole('button', { name: /try again|reload|retry/i }).first()).toBeVisible();
    await page.waitForLoadState('networkidle');
    expect(loads).toBeLessThanOrEqual(1);
  });
});
