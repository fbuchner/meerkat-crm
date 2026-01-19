import { test, expect } from '@playwright/test';
import { TEST_USER } from './global-setup';
import { loginUser, logoutUser } from './fixtures';

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
      
      // Fill login form
      await page.getByLabel(/username or email/i).fill(TEST_USER.username);
      await page.getByLabel(/password/i).fill(TEST_USER.password);
      
      // Submit
      await page.getByRole('button', { name: /login/i }).click();
      
      // Should show dashboard after login
      await expect(page.getByRole('heading', { name: /dashboard/i })).toBeVisible({ timeout: 10000 });
    });

    test('should show error with invalid credentials', async ({ page }) => {
      await page.goto('/');
      
      // Fill with wrong credentials
      await page.getByLabel(/username or email/i).fill('wronguser');
      await page.getByLabel(/password/i).fill('wrongpass');
      
      // Submit
      await page.getByRole('button', { name: /login/i }).click();
      
      // Should show error
      await expect(page.getByText(/invalid|incorrect|failed/i)).toBeVisible({ timeout: 5000 });
    });

    test('should show error with empty credentials', async ({ page }) => {
      await page.goto('/');
      
      // Click login without filling anything
      await page.getByRole('button', { name: /login/i }).click();
      
      // Should show validation error or still be on login page
      await expect(page.getByRole('heading', { name: /login/i })).toBeVisible();
    });
  });

  test.describe('Logout', () => {
    test('should logout successfully', async ({ page }) => {
      await loginUser(page);
      
      // Find and click logout button
      await page.getByRole('button', { name: /logout/i }).click();
      
      // Should show login form again
      await expect(page.getByRole('heading', { name: /login/i })).toBeVisible({ timeout: 10000 });
    });
  });

  test.describe('Registration', () => {
    test('should navigate to registration page', async ({ page }) => {
      await page.goto('/');
      
      // Click register link
      await page.getByRole('link', { name: /register/i }).click();
      
      // Should show registration form
      await expect(page.getByRole('heading', { name: /register/i })).toBeVisible();
      await expect(page.getByLabel(/username/i)).toBeVisible();
      await expect(page.getByLabel(/email/i)).toBeVisible();
    });
  });

  test.describe('Protected Routes', () => {
    test('should show login when accessing contacts without auth', async ({ page }) => {
      await page.goto('/contacts');
      // App shows login form at any route when not authenticated
      await expect(page.getByRole('heading', { name: /login/i })).toBeVisible();
    });

    test('should show login when accessing settings without auth', async ({ page }) => {
      await page.goto('/settings');
      await expect(page.getByRole('heading', { name: /login/i })).toBeVisible();
    });
  });
});
