import { test, expect } from './fixtures';
import { loginUser, waitForLoading } from './fixtures';

test.describe('Reminders', () => {
  test.beforeEach(async ({ page }) => {
    await loginUser(page);
  });

  test('should create a reminder from contact detail page', async ({ page }) => {
    // Go to contacts and select one (use search to find Alice reliably)
    await page.goto('/contacts');
    await waitForLoading(page);
    
    // Search for Alice to ensure she's visible
    const searchInput = page.locator('input[placeholder*="earch"]').first();
    await searchInput.fill('Alice');
    await searchInput.press('Enter');
    await page.waitForTimeout(500);
    
    // Click on Alice
    await page.getByText('Alice Johnson').click();
    await expect(page).toHaveURL(/\/contacts\/\d+/);
    
    // Click on Reminders tab (must click to switch from Timeline)
    const remindersTab = page.getByRole('tab', { name: /reminders/i });
    await expect(remindersTab).toBeVisible({ timeout: 5000 });
    await remindersTab.click();
    
    // Wait for tab content to load
    await page.waitForTimeout(500);
    
    // Click Add Reminder button
    await page.getByRole('button', { name: /add.*reminder/i }).click();
    
    // Should show reminder dialog
    await expect(page.getByRole('dialog')).toBeVisible();
    
    // Fill in reminder details
    const messageInput = page.getByRole('textbox', { name: /message/i });
    await messageInput.fill('E2E Test Reminder');
    
    // Save reminder
    await page.getByRole('button', { name: /save|create/i }).click();
    
    // Wait for dialog to close
    await page.waitForTimeout(1000);
    
    // Should show the reminder in the list
    await expect(page.locator('p').filter({ hasText: 'E2E Test Reminder' }).first()).toBeVisible({ timeout: 5000 });
  });

  test('should show reminder form fields', async ({ page }) => {
    // Go to contacts and select one (use search to find Bob reliably)
    await page.goto('/contacts');
    await waitForLoading(page);
    
    // Search for Bob to ensure he's visible
    const searchInput = page.locator('input[placeholder*="earch"]').first();
    await searchInput.fill('Bob');
    await searchInput.press('Enter');
    await page.waitForTimeout(500);
    
    await page.getByText('Bob Smith').click();
    await expect(page).toHaveURL(/\/contacts\/\d+/);
    
    // Click on Reminders tab (must click to switch from Timeline)
    const remindersTab = page.getByRole('tab', { name: /reminders/i });
    await expect(remindersTab).toBeVisible({ timeout: 5000 });
    await remindersTab.click();
    
    // Wait for tab content to load
    await page.waitForTimeout(500);
    
    // Open reminder dialog
    await page.getByRole('button', { name: /add.*reminder/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible();
    
    // Check that form fields are present
    await expect(page.getByRole('textbox', { name: /message/i })).toBeVisible();
    await expect(page.getByLabel(/date/i).first()).toBeVisible();
    
    // Close dialog
    await page.keyboard.press('Escape');
  });
});
