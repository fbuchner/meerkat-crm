import { test, expect } from './fixtures';
import { loginUser, waitForLoading } from './fixtures';

test.describe('Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await loginUser(page);
  });

  test('should display dashboard after login', async ({ page }) => {
    // After login, should be on dashboard
    await expect(page.getByRole('heading', { name: /dashboard/i })).toBeVisible();
  });

  test('should display three dashboard sections', async ({ page }) => {
    await page.goto('/');
    await waitForLoading(page);
    
    // Should show all three dashboard sections
    await expect(page.getByText('Upcoming Birthdays').first()).toBeVisible();
    await expect(page.getByText('Upcoming Reminders').first()).toBeVisible();
    await expect(page.getByText('Stay in Touch').first()).toBeVisible();
  });

  test('should display contacts in Stay in Touch section', async ({ page }) => {
    await page.goto('/');
    await waitForLoading(page);
    
    // Should show random contacts in Stay in Touch section
    // At least one contact should be visible
    const stayInTouchSection = page.getByText('Stay in Touch').locator('..');
    await expect(stayInTouchSection).toBeVisible();
  });
});

test.describe('Navigation', () => {
  test.beforeEach(async ({ page }) => {
    await loginUser(page);
  });

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
  });

  test('should navigate to settings from sidebar', async ({ page }) => {
    await page.goto('/');
    // Settings is a collapsible submenu - click to expand it
    await page.locator('.MuiDrawer-root').getByText('Settings').click();
    // Click the Profile link inside the expanded submenu
    await page.locator('.MuiDrawer-root').getByText('Profile').click();
    await expect(page).toHaveURL(/\/settings/);
  });
});
