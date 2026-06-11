import { test, expect, LOGGED_OUT, loginUser } from './fixtures';
import { TEST_USER } from './global-setup';

// Auth flows must start from logged-out browser
test.use({ storageState: LOGGED_OUT });

test.describe('Authentication', () => {
  test.describe('Login', () => {
    test('should display login page when not authenticated', async ({ page }) => {
      await page.goto('/');

      // Should show login form (app shows login at root when not authenticated)
      await expect(page.getByRole('heading', { name: /login/i })).toBeVisible();
      await expect(page.getByLabel(/username or email/i)).toBeVisible();
      await expect(page.getByLabel(/password/i)).toBeVisible();
      await expect(page.getByRole('button', { name: /login/i })).toBeVisible();
    });

    test('should login successfully with valid credentials', async ({ page }) => {
      await page.goto('/');

      await page.getByLabel(/username or email/i).fill(TEST_USER.username);
      await page.getByLabel(/password/i).fill(TEST_USER.password);
      await page.getByRole('button', { name: /login/i }).click();

      // Should show dashboard after login
      await expect(page.getByRole('heading', { name: /dashboard/i })).toBeVisible({ timeout: 10000 });
    });

    test('should show error with invalid credentials', async ({ page }) => {
      await page.goto('/');

      await page.getByLabel(/username or email/i).fill('wronguser');
      await page.getByLabel(/password/i).fill('wrongpass');
      await page.getByRole('button', { name: /login/i }).click();

      await expect(page.getByText(/invalid|incorrect|failed/i)).toBeVisible({ timeout: 5000 });
    });

    test('should stay on login page with empty credentials', async ({ page }) => {
      await page.goto('/');

      // Required fields prevent submission; we should remain on the login page.
      await page.getByRole('button', { name: /login/i }).click();
      await expect(page.getByRole('heading', { name: /login/i })).toBeVisible();
    });
  });

  test.describe('Logout', () => {
    test('should logout successfully', async ({ page }) => {
      await loginUser(page);

      await page.getByRole('button', { name: /logout/i }).click();

      // Should show login form again
      await expect(page.getByRole('heading', { name: /login/i })).toBeVisible({ timeout: 10000 });
    });
  });

  test.describe('Registration', () => {
    test('should navigate to registration page', async ({ page }) => {
      await page.goto('/');

      await page.getByRole('link', { name: /register|account/i }).click();

      await expect(page.getByRole('heading', { name: /register/i })).toBeVisible();
      await expect(page.getByLabel(/username/i)).toBeVisible();
      await expect(page.getByLabel(/email/i)).toBeVisible();
    });

    test('should register a new account end-to-end', async ({ page }) => {
      // Unique credentials so the test is repeatable across runs.
      const unique = Date.now();
      const username = `e2e_reg_${unique}`;
      const email = `e2e_reg_${unique}@example.com`;

      await page.goto('/register');

      await page.getByLabel(/username/i).fill(username);
      await page.getByLabel(/email/i).fill(email);
      await page.getByLabel(/password/i).fill('RegPassword123!');
      await page.getByRole('button', { name: /register/i }).click();

      // Success alert appears, then the app redirects to the login page.
      await expect(page.getByRole('heading', { name: /login/i })).toBeVisible({ timeout: 10000 });

      // The freshly registered account can actually log in.
      await page.getByLabel(/username or email/i).fill(username);
      await page.getByLabel(/password/i).fill('RegPassword123!');
      await page.getByRole('button', { name: /login/i }).click();
      await expect(page.getByRole('heading', { name: /dashboard/i })).toBeVisible({ timeout: 10000 });
    });
  });

  test.describe('Protected Routes', () => {
    test('should show login when accessing contacts without auth', async ({ page }) => {
      await page.goto('/contacts');
      await expect(page.getByRole('heading', { name: /login/i })).toBeVisible();
    });

    test('should show login when accessing settings without auth', async ({ page }) => {
      await page.goto('/settings');
      await expect(page.getByRole('heading', { name: /login/i })).toBeVisible();
    });
  });
});
