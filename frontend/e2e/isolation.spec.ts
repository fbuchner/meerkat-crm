import { test, expect } from './fixtures';
import { request } from '@playwright/test';
import { API_BASE_URL } from './global-setup';

// A second account used only to prove data isolation (a 409 on re-run is fine).
const USER_B = {
  username: 'e2e_isolation_userb',
  email: 'e2e_isolation_userb@example.com',
  password: 'IsolationPass123!',
};

test.describe('Multi-user isolation', () => {
  test('a user cannot see another user\'s contacts', async ({ page }) => {
    // Sanity: the seeded user (userA, via the shared storageState) can see Alice.
    const ownView = await page.request.get(
      `${API_BASE_URL}/contacts?search=${encodeURIComponent('Alice Johnson')}&limit=10`
    );
    expect(ownView.ok()).toBeTruthy();
    const own = await ownView.json();
    expect(
      (own.contacts || []).some((c: any) => c.firstname === 'Alice' && c.lastname === 'Johnson')
    ).toBeTruthy();

    // userB gets a clean API context with no shared cookies.
    const ctx = await request.newContext();
    try {
      await ctx.post(`${API_BASE_URL}/register`, { data: USER_B }).catch(() => {});

      const login = await ctx.post(`${API_BASE_URL}/login`, {
        data: { identifier: USER_B.username, password: USER_B.password },
      });
      expect(login.ok(), 'userB login should succeed').toBeTruthy();

      // userB must not see any of userA's seeded contacts.
      const search = await ctx.get(
        `${API_BASE_URL}/contacts?search=${encodeURIComponent('Alice Johnson')}&limit=10`
      );
      expect(search.ok()).toBeTruthy();
      const result = await search.json();
      expect(
        (result.contacts || []).some((c: any) => c.firstname === 'Alice' && c.lastname === 'Johnson')
      ).toBeFalsy();
    } finally {
      await ctx.dispose();
    }
  });
});
