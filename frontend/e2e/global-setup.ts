import { chromium, FullConfig, BrowserContext } from '@playwright/test';

const API_BASE_URL = 'http://localhost:8080/api/v1';

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
    email: 'alice@example.com',
    phone: '+1 555-0101',
    birthday: '1990-03-15',
    circles: ['Friends', 'Work'],
  },
  {
    firstname: 'Bob',
    lastname: 'Smith',
    email: 'bob@example.com',
    phone: '+1 555-0102',
    circles: ['Family'],
  },
  {
    firstname: 'Carol',
    lastname: 'Williams',
    email: 'carol@example.com',
    birthday: '1985-07-22',
    circles: ['Friends'],
  },
  {
    firstname: 'David',
    lastname: 'Brown',
    email: 'david@example.com',
    circles: ['Work'],
  },
  {
    firstname: 'Eve',
    lastname: 'Davis',
    email: 'eve@example.com',
    phone: '+1 555-0105',
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
    } else {
      const data = await response.json();
      // User might already exist from previous run
      if (data.error?.code === 'USER_EXISTS' || response.status === 409) {
        console.log('Test user already exists');
      } else {
        console.log('Registration response:', data);
      }
    }
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
    throw new Error(`Login failed: ${loginResponse.status()} - ${body}`);
  }

  console.log('Logged in successfully');

  // Check if contacts already exist
  const existingResponse = await context.request.get(`${API_BASE_URL}/contacts?limit=1`);

  if (existingResponse.ok()) {
    const existing = await existingResponse.json();
    if (existing.total > 0) {
      console.log(`info: ${existing.total} contacts already exist, skipping creation`);
      return;
    }
  }

  console.log('Creating sample contacts...');

  for (const contact of SAMPLE_CONTACTS) {
    try {
      const response = await context.request.post(`${API_BASE_URL}/contacts`, {
        data: contact,
      });

      if (response.ok()) {
        console.log(`  Created contact: ${contact.firstname} ${contact.lastname}`);
      } else {
        const error = await response.json();
        console.log(`  Failed to create ${contact.firstname}: ${error.error?.message}`);
      }
    } catch (error) {
      console.log(`  Error creating ${contact.firstname}:`, error);
    }
  }

  console.log('Sample contacts created');
}

export default globalSetup;
