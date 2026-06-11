import { chromium, FullConfig, BrowserContext } from '@playwright/test';

export const API_BASE_URL = 'http://localhost:8080/api/v1';

// Test user credentials
export const TEST_USER = {
  username: 'testuser',
  email: 'testuser@example.com',
  password: 'TestPassword123!',
};

// Sample contacts for testing
const SAMPLE_CONTACTS = [
  {
    firstname: 'Alice',
    lastname: 'Johnson',
    emails: [{ type: 'home', value: 'alice@example.com' }],
    phones: [{ type: 'mobile', value: '+1 555-0101' }],
    birthday: '1990-03-15',
    circles: ['Friends', 'Work'],
  },
  {
    firstname: 'Bob',
    lastname: 'Smith',
    emails: [{ type: 'home', value: 'bob@example.com' }],
    phones: [{ type: 'mobile', value: '+1 555-0102' }],
    circles: ['Family'],
  },
  {
    firstname: 'Carol',
    lastname: 'Williams',
    emails: [{ type: 'home', value: 'carol@example.com' }],
    birthday: '1985-07-22',
    circles: ['Friends'],
  },
  {
    firstname: 'David',
    lastname: 'Brown',
    emails: [{ type: 'home', value: 'david@example.com' }],
    circles: ['Work'],
  },
  {
    firstname: 'Eve',
    lastname: 'Davis',
    emails: [{ type: 'home', value: 'eve@example.com' }],
    phones: [{ type: 'mobile', value: '+1 555-0105' }],
  },
];

async function globalSetup(config: FullConfig) {
  console.log('Setting up test environment...');

  // Wait for backend to be ready
  await waitForBackend();

  // Register test user (ignore if already exists)
  await registerTestUser();

  // Use a browser context to login and make authenticated API calls
  // (login now uses httpOnly cookies instead of returning a token)
  const browser = await chromium.launch();
  const context = await browser.newContext({ baseURL: 'http://localhost:8080' });

  try {
    await loginAndCreateContacts(context);
  } finally {
    await browser.close();
  }

  console.log('Test environment ready!');
}

async function waitForBackend(maxRetries = 30): Promise<void> {
  console.log('Waiting for backend to be ready...');

  for (let i = 0; i < maxRetries; i++) {
    try {
      const response = await fetch('http://localhost:8080/health');
      if (response.ok) {
        console.log('Backend is ready');
        return;
      }
    } catch {
      // Backend not ready yet
    }
    await new Promise(resolve => setTimeout(resolve, 1000));
  }

  throw new Error('Backend did not become ready in time');
}

async function registerTestUser(): Promise<void> {
  console.log('Registering test user...');

  try {
    const response = await fetch(`${API_BASE_URL}/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        username: TEST_USER.username,
        email: TEST_USER.email,
        password: TEST_USER.password,
      }),
    });

    if (response.ok) {
      console.log('Test user registered');
      return;
    }

    const data = await response.json().catch(() => ({}));

    // User already exists from a previous run — fine.
    if (data.error?.code === 'USER_EXISTS' || response.status === 409) {
      console.log('Test user already exists');
      return;
    }

    // E2E suite registers the seeded user and a second user
    if (data.error?.code === 'registration_disabled') {
      console.warn(
        '\n⚠️  Registration is DISABLED on this backend (DISABLE_REGISTRATION=true).\n' +
        '   The E2E suite needs it enabled to seed users and to exercise the\n' +
        '   registration/isolation specs. Set DISABLE_REGISTRATION=false, or run\n' +
        '   against the docker-compose.test.yml stack which already does.\n'
      );
      return;
    }

    console.log('Registration response:', data);
  } catch (error) {
    console.log('Registration error (user may already exist):', error);
  }
}

async function loginAndCreateContacts(context: BrowserContext): Promise<void> {
  console.log('Logging in test user...');

  // Login via API — cookies are set automatically on the browser context
  const loginResponse = await context.request.post(`${API_BASE_URL}/login`, {
    data: {
      identifier: TEST_USER.username,
      password: TEST_USER.password,
    },
  });

  if (!loginResponse.ok()) {
    const body = await loginResponse.text();
    throw new Error(
      `Login as the seeded test user failed: ${loginResponse.status()} - ${body}\n` +
      'On a fresh backend this usually means registration is disabled, so the test ' +
      'user was never created. Set DISABLE_REGISTRATION=false (the ' +
      'docker-compose.test.yml stack already does) and retry.'
    );
  }

  console.log('Logged in successfully');

  // Remove data left behind by previous runs so the suite starts clean
  await cleanupLeftoverTestData(context);

  // Ensure each sample contact exists exactly once (idempotent upsert by name).
  console.log('Ensuring sample contacts exist...');

  for (const contact of SAMPLE_CONTACTS) {
    try {
      const search = `${contact.firstname} ${contact.lastname}`;
      const lookup = await context.request.get(
        `${API_BASE_URL}/contacts?search=${encodeURIComponent(search)}&limit=1`
      );
      if (lookup.ok()) {
        const found = await lookup.json();
        if ((found.contacts || []).some(
          (c: any) => c.firstname === contact.firstname && c.lastname === contact.lastname
        )) {
          console.log(`  Exists: ${search}`);
          continue;
        }
      }

      const response = await context.request.post(`${API_BASE_URL}/contacts`, {
        data: contact,
      });

      if (response.ok()) {
        console.log(`  Created contact: ${search}`);
      } else {
        const error = await response.json();
        console.log(`  Failed to create ${contact.firstname}: ${error.error?.message}`);
      }
    } catch (error) {
      console.log(`  Error creating ${contact.firstname}:`, error);
    }
  }

  console.log('Sample contacts ready');
}

// Deletes contacts created by previous E2E runs. 
export const E2E_CONTACT_PREFIX = 'E2EFixture';

async function cleanupLeftoverTestData(context: BrowserContext): Promise<void> {
  try {
    const response = await context.request.get(
      `${API_BASE_URL}/contacts?search=${encodeURIComponent(E2E_CONTACT_PREFIX)}&limit=200`
    );
    if (!response.ok()) return;

    const data = await response.json();
    const stale = (data.contacts || []).filter((c: any) =>
      (c.firstname || '').startsWith(E2E_CONTACT_PREFIX)
    );

    for (const contact of stale) {
      await context.request.delete(`${API_BASE_URL}/contacts/${contact.ID}`).catch(() => {});
    }

    if (stale.length > 0) {
      console.log(`Cleaned up ${stale.length} leftover E2E contact(s)`);
    }
  } catch (error) {
    console.log('Cleanup of leftover test data failed (non-fatal):', error);
  }
}

export default globalSetup;
