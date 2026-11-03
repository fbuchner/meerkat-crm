import { Page, Locator, APIRequestContext, expect, request as playwrightRequest } from '@playwright/test';
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
 * Waits for any MUI loading spinners to disappear.
 */
export async function waitForLoading(page: Page): Promise<void> {
  await page.waitForSelector('[role="progressbar"]', { state: 'hidden', timeout: 10000 });
}

/**
 * Submits a search on the contacts list. It doesn't wait for results, so follow
 * it with an auto-waiting assertion on the contact you expect.
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

export interface UserCredentials {
  username: string;
  email: string;
  password: string;
}

/**
 * Registers a user (an existing account is fine, any other failure is not) and
 * returns an API context logged in as them. The caller disposes it.
 */
export async function loginAsNewUser(user: UserCredentials): Promise<APIRequestContext> {
  // Without an explicit storageState this inherits the seeded user's cookies
  // from the config, and a cookie outranks any Authorization header.
  const context = await playwrightRequest.newContext({ storageState: LOGGED_OUT });
  const registered = await context.post(`${API_BASE_URL}/register`, { data: user });
  expect(
    registered.ok() || registered.status() === 409,
    `registering ${user.username} failed: ${registered.status()}`
  ).toBeTruthy();

  const login = await context.post(`${API_BASE_URL}/login`, {
    data: { identifier: user.username, password: user.password },
  });
  expect(login.ok(), `logging in ${user.username} failed: ${login.status()}`).toBeTruthy();
  return context;
}

let uniqueCounter = 0;

/**
 * Credentials nobody has used yet, for specs that change per-user settings or
 * need a clean slate and so can't share the seeded user.
 */
export function uniqueUser(prefix: string): UserCredentials {
  const id = `${prefix}_${Date.now()}_${process.pid}_${uniqueCounter++}`;
  return { username: id, email: `${id}@example.com`, password: 'E2eUniquePass123!' };
}

/** Registers a user, then signs the page in through the UI. */
export async function signInAsNewUser(page: Page, user: UserCredentials): Promise<void> {
  const context = await loginAsNewUser(user);
  await context.dispose();
  await loginUser(page, user);
}

/**
 * The innermost element holding both the text and a button with that exact name,
 * for lists whose items have no landmark or list role of their own.
 */
export function itemWithButton(page: Page, text: string, buttonName: string): Locator {
  return page
    .locator('div, li, tr')
    .filter({ hasText: text })
    .filter({ has: page.getByRole('button', { name: buttonName, exact: true }) })
    .last();
}
