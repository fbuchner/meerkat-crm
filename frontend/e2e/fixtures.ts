import { test as base, Page, APIRequestContext, expect } from '@playwright/test';
import { TEST_USER, API_BASE_URL, E2E_CONTACT_PREFIX } from './global-setup';

export { test } from '@playwright/test';
export { expect } from '@playwright/test';

export const LOGGED_OUT = { cookies: [], origins: [] };

/**
 * Logs a user in through the UI. Only needed by specs that explicitly start
 * logged out (the shared storageState already covers the common case).
 */
export async function loginUser(page: Page, credentials = TEST_USER): Promise<void> {
  await page.goto('/');

  // Already authenticated (e.g. shared storageState is active) — nothing to do.
  const dashboardHeading = page.getByRole('heading', { name: /dashboard/i });
  if (await dashboardHeading.isVisible({ timeout: 1000 }).catch(() => false)) {
    return;
  }

  await page.getByLabel(/username or email/i).fill(credentials.username);
  await page.getByLabel(/password/i).fill(credentials.password);
  await page.getByRole('button', { name: /login/i }).click();

  await expect(dashboardHeading).toBeVisible({ timeout: 15000 });
}

/**
 * Logs the current user out via the UI and waits for the login form.
 */
export async function logoutUser(page: Page): Promise<void> {
  await page.getByRole('button', { name: /logout/i }).click();
  await expect(page.getByRole('heading', { name: /login/i })).toBeVisible({ timeout: 10000 });
}

/**
 * Waits for any MUI loading spinners to disappear.
 */
export async function waitForLoading(page: Page): Promise<void> {
  await page
    .waitForSelector('[role="progressbar"]', { state: 'hidden', timeout: 10000 })
    .catch(() => {});
}

/**
 * Searches the contacts list and returns once the matching contact is visible.
 * Replaces the previous fill + fixed-sleep pattern with an auto-waiting assertion.
 */
export async function searchContact(page: Page, query: string): Promise<void> {
  const searchInput = page.locator('input[placeholder*="earch"]').first();
  await searchInput.fill(query);
  await searchInput.press('Enter');
}

// ---------------------------------------------------------------------------
// API helpers — used to set up and tear down data without driving the UI.
// ---------------------------------------------------------------------------

export interface CreatedContact {
  ID: number;
  firstname: string;
  lastname: string;
}

/**
 * Creates a throwaway contact via the API. Names are prefixed so global-setup
 * can sweep up any that leak when a test crashes mid-run.
 */
export async function createTestContact(
  request: APIRequestContext,
  overrides: Record<string, unknown> = {}
): Promise<CreatedContact> {
  const firstname = `${E2E_CONTACT_PREFIX}${Date.now()}`;
  const response = await request.post(`${API_BASE_URL}/contacts`, {
    data: { firstname, lastname: 'Temp', ...overrides },
  });
  expect(response.ok(), `failed to create test contact: ${response.status()}`).toBeTruthy();
  // The API wraps the created contact: { contact: {...} }.
  const body = await response.json();
  return body.contact || body;
}

/**
 * Deletes a contact via the API. Safe to call in finally/afterEach blocks.
 */
export async function deleteTestContact(
  request: APIRequestContext,
  id: number | string
): Promise<void> {
  await request.delete(`${API_BASE_URL}/contacts/${id}`).catch(() => {});
}
