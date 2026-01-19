import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright configuration for Meerkat CRM E2E tests.
 * 
 * Run tests with: npx playwright test
 * Run with UI:    npx playwright test --ui
 * Run specific:   npx playwright test auth.spec.ts
 */
export default defineConfig({
  testDir: './e2e',
  
  // Run tests in files in parallel
  fullyParallel: true,
  
  // Fail the build on CI if you accidentally left test.only in the source code
  forbidOnly: !!process.env.CI,
  
  // Retry on CI only
  retries: process.env.CI ? 2 : 0,
  
  // Use single worker to avoid parallel login issues and rate limiting
  workers: 1,
  
  // Reporter to use
  reporter: [
    ['html', { open: 'never' }],
    ['list']
  ],
  
  // Shared settings for all projects
  use: {
    // Base URL for navigation
    baseURL: 'http://localhost:3000',
    
    // Collect trace when retrying the failed test
    trace: 'on-first-retry',
    
    // Take screenshot on failure
    screenshot: 'only-on-failure',
    
    // Record video on failure
    video: 'on-first-retry',
  },

  // Configure projects for major browsers
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  // Global setup for test data seeding
  globalSetup: './e2e/global-setup.ts',

  // Timeout settings
  timeout: 30000,
  expect: {
    timeout: 5000,
  },
});
