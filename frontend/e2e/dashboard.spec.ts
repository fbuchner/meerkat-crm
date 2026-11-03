import { test, expect } from './fixtures';
import { waitForLoading } from './fixtures';

// Authenticated via the shared storageState (see playwright.config.ts).
test.describe('Dashboard', () => {
  test('should display dashboard', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByRole('heading', { name: /dashboard/i })).toBeVisible();
  });

  test('should display three dashboard sections', async ({ page }) => {
    await page.goto('/');
    await waitForLoading(page);

    await expect(page.getByText('Upcoming Birthdays').first()).toBeVisible();
    await expect(page.getByText('Upcoming Reminders').first()).toBeVisible();
    await expect(page.getByText('Stay in Touch').first()).toBeVisible();
  });
});

test.describe('Navigation', () => {
  test('should navigate to contacts from sidebar', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('link', { name: /contacts/i }).click();
    await expect(page).toHaveURL(/\/contacts/);
    await expect(page.getByRole('heading', { name: /contacts/i })).toBeVisible();
  });

  test('should navigate to activities from sidebar', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('link', { name: /activities/i }).click();
    await expect(page).toHaveURL(/\/activities/);
    await expect(page.getByRole('heading', { name: /activities/i })).toBeVisible();
  });

  test('should navigate to settings from sidebar', async ({ page }) => {
    await page.goto('/');
    // Settings is a collapsible submenu — expand it, then open Profile.
    await page.getByRole('button', { name: 'Settings', exact: true }).click();
    await page.getByRole('link', { name: 'Profile', exact: true }).click();
    await expect(page).toHaveURL(/\/settings/);
    await expect(page.getByRole('heading', { name: /settings/i }).first()).toBeVisible();
  });
});
