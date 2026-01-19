import { test, expect } from './fixtures';
import { loginUser, waitForLoading } from './fixtures';

test.describe('Contacts', () => {
  test.beforeEach(async ({ page }) => {
    await loginUser(page);
  });

  test('should display contacts page with filter controls', async ({ page }) => {
    await page.goto('/contacts');
    
    // Should show contacts heading
    await expect(page.getByRole('heading', { name: /contacts/i })).toBeVisible();
    
    // Should show filter and sort controls
    await expect(page.getByLabel(/filter by circle/i)).toBeVisible();
    await expect(page.getByLabel(/sort by/i)).toBeVisible();
    
    // Should show Add Contact button
    await expect(page.getByRole('button', { name: /add/i })).toBeVisible();
  });

  test('should display seeded contacts', async ({ page }) => {
    await page.goto('/contacts');
    await waitForLoading(page);
    
    // Use search to find seeded contacts (they may be below the fold due to accumulated test data)
    const searchInput = page.locator('input[placeholder*="earch"]').first();
    await searchInput.fill('Alice');
    await searchInput.press('Enter');
    await page.waitForTimeout(500);
    await expect(page.getByText('Alice Johnson')).toBeVisible();
    
    // Search for Bob
    await searchInput.clear();
    await searchInput.fill('Bob');
    await searchInput.press('Enter');
    await page.waitForTimeout(500);
    await expect(page.getByText('Bob Smith')).toBeVisible();
  });

  test('should navigate to contact detail', async ({ page }) => {
    await page.goto('/contacts');
    await waitForLoading(page);
    
    // Use search to find Alice (may be below fold due to accumulated test data)
    const searchInput = page.locator('input[placeholder*="earch"]').first();
    await searchInput.fill('Alice');
    await searchInput.press('Enter');
    await page.waitForTimeout(500);
    
    // Click on a contact
    await page.getByText('Alice Johnson').click();
    
    // Should navigate to contact detail
    await expect(page).toHaveURL(/\/contacts\/\d+/);
    
    // Should show contact info
    await expect(page.getByText('alice@example.com')).toBeVisible();
  });

  test('should create a new contact', async ({ page }) => {
    await page.goto('/contacts');
    
    // Click Add Contact button
    await page.getByRole('button', { name: /add/i }).click();
    
    // Should show dialog
    await expect(page.getByRole('dialog')).toBeVisible();
    
    // Fill in minimal contact details
    await page.getByLabel(/first.*name/i).fill('E2ETest');
    await page.getByLabel(/last.*name/i).fill('Contact');
    
    // Submit the form
    await page.getByRole('button', { name: /create/i }).click();
    
    // Should navigate to the new contact's detail page
    await expect(page).toHaveURL(/\/contacts\/\d+/);
    await expect(page.getByText('E2ETest Contact')).toBeVisible();
  });
});
