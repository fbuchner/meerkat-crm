import { test as base, Page, expect } from '@playwright/test';
import { TEST_USER } from './global-setup';

// Extended test fixtures with authentication helpers
export const test = base.extend<{
  authenticatedPage: Page;
}>({
  // Provides a page that's already logged in
  authenticatedPage: async ({ page }, use) => {
    await loginUser(page);
    await use(page);
  },
});

export { expect } from '@playwright/test';

/**
 * Helper function to login a user with retry logic for rate limiting
 */
export async function loginUser(page: Page, credentials = TEST_USER): Promise<void> {
  const maxRetries = 3;
  
  for (let attempt = 1; attempt <= maxRetries; attempt++) {
    await page.goto('/');
    
    // Check if already logged in (dashboard visible)
    const dashboardHeading = page.getByRole('heading', { name: /dashboard/i });
    if (await dashboardHeading.isVisible({ timeout: 1000 }).catch(() => false)) {
      return; // Already logged in
    }
    
    // Wait for login form to appear
    await page.waitForSelector('form', { timeout: 10000 });
    
    // Fill login form
    const identifierInput = page.getByLabel(/username or email/i);
    const passwordInput = page.getByLabel(/password/i);
    
    await identifierInput.fill(credentials.username);
    await passwordInput.fill(credentials.password);
    
    // Submit
    await page.getByRole('button', { name: /login/i }).click();
    
    // Wait for either success or failure
    const result = await Promise.race([
      dashboardHeading.waitFor({ timeout: 10000 }).then(() => 'success'),
      page.getByText(/login failed|rate limit|too many/i).waitFor({ timeout: 10000 }).then(() => 'failed'),
    ]).catch(() => 'timeout');
    
    if (result === 'success') {
      return;
    }
    
    // If rate limited or failed, wait before retry
    if (attempt < maxRetries) {
      await page.waitForTimeout(2000 * attempt); // Exponential backoff
    }
  }
  
  // Final check - should be logged in
  await expect(page.getByRole('heading', { name: /dashboard/i })).toBeVisible({ timeout: 10000 });
}

/**
 * Helper function to logout a user
 */
export async function logoutUser(page: Page): Promise<void> {
  await page.getByRole('button', { name: /logout/i }).click();
  // Wait for login form to appear (app shows login at same URL)
  await page.waitForSelector('form', { timeout: 10000 });
  await expect(page.getByRole('heading', { name: /login/i })).toBeVisible();
}

/**
 * Helper to wait for loading states to complete
 */
export async function waitForLoading(page: Page): Promise<void> {
  // Wait for any loading spinners to disappear
  await page.waitForSelector('[role="progressbar"]', { state: 'hidden', timeout: 10000 }).catch(() => {});
}

/**
 * Helper to navigate to a specific page while logged in
 */
export async function navigateTo(page: Page, path: string): Promise<void> {
  await page.goto(path);
  await waitForLoading(page);
}
