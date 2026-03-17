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
  
  fullyParallel: true,
  
  // Fail the build on CI if you accidentally left test.only in the source code
  forbidOnly: !!process.env.CI,
  
  // Retry on CI only
  retries: process.env.CI ? 1 : 0,
  
  // Use single worker to avoid parallel login issues and rate limiting
  workers: 1,
  
  // Reporter to use
  reporter: [
    ['html', { open: 'never' }],
    ['list']
  ],
  
  // Shared settings for all projects
  use: {
    baseURL: 'http://localhost:7300',
    
    // Collect trace when retrying the failed test
    trace: 'on-first-retry',
    
    screenshot: 'only-on-failure',
    video: 'on-first-retry',
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  // Global setup for test data seeding
  globalSetup: './e2e/global-setup.ts',

  timeout: 30000,
  expect: {
    timeout: 5000,
  },
});
