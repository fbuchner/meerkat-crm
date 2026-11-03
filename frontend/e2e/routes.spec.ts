import { test, expect } from './fixtures';

for (const route of ['/', '/contacts', '/contacts/duplicates', '/notes', '/activities', '/settings', '/settings/data', '/network', '/users', '/api-tokens']) {
  test(`route ${route} renders without page or CSP errors`, async ({ page }) => {
    const problems: string[] = [];
    page.on('pageerror', (e) => problems.push('pageerror: ' + e.message));
    page.on('console', (m) => {
      if (m.type() === 'error' && !/status of 4\d\d/.test(m.text())) problems.push('console: ' + m.text());
    });
    await page.goto(route);
    await page.waitForLoadState('networkidle');
    await expect(page.locator('#root')).not.toBeEmpty();
    await expect(page.getByText(/something went wrong/i)).toHaveCount(0);
    expect(problems).toEqual([]);
  });
}
