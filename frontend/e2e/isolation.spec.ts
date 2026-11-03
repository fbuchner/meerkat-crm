import { test, expect, loginAsNewUser } from './fixtures';
import { API_BASE_URL } from './global-setup';
import type { APIRequestContext } from '@playwright/test';

// A second account used only to prove data isolation.
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

    const userB = await loginAsNewUser(USER_B);
    try {
      // userB must not see any of userA's seeded contacts.
      const search = await userB.get(
        `${API_BASE_URL}/contacts?search=${encodeURIComponent('Alice Johnson')}&limit=10`
      );
      expect(search.ok()).toBeTruthy();
      const result = await search.json();
      expect(
        (result.contacts || []).some((c: any) => c.firstname === 'Alice' && c.lastname === 'Johnson')
      ).toBeFalsy();
    } finally {
      await userB.dispose();
    }
  });

  test('a user cannot read, change or delete another user\'s resources by ID', async ({}, testInfo) => {
    // A dedicated owner, so events from specs running alongside this one can't
    // add webhook deliveries and change the snapshot.
    const stamp = `${Date.now()}_${testInfo.workerIndex}_${testInfo.repeatEachIndex}`;
    const userA = await loginAsNewUser({
      username: `e2e_isolation_owner_${stamp}`,
      email: `e2e_isolation_owner_${stamp}@example.com`,
      password: USER_B.password,
    });
    const created = await createResources(userA, stamp);
    const userB = await loginAsNewUser(USER_B);

    try {
      const before = await snapshot(userA, created);

      for (const probe of crossUserProbes(created)) {
        const response = await userB.fetch(`${API_BASE_URL}${probe.path}`, {
          method: probe.method,
          ...(probe.data ? { data: probe.data } : {}),
          ...(probe.multipart ? { multipart: probe.multipart } : {}),
        });
        expect.soft(response.status(), `${probe.method} ${probe.path}`).toBe(probe.expected);
      }

      // userB's own resources must not be able to reach into userA's data either.
      const ownContact = idOf(
        await postJson(userB, '/contacts', { firstname: 'E2EFixtureOwn', lastname: stamp }),
        'contact'
      );
      const date = new Date().toISOString();
      for (const probe of [
        { method: 'GET', path: `/contacts/${ownContact}/merge-preview?source_id=${created.contact}` },
        { method: 'POST', path: `/contacts/${ownContact}/merge`, data: { source_id: created.contact } },
        { method: 'POST', path: '/activities', data: { title: 'linked', date, contact_ids: [created.contact] } },
        { method: 'POST', path: `/contacts/${ownContact}/relationships`, data: { name: 'x', type: 'Friend', related_contact_id: created.contact } },
        { method: 'POST', path: '/notes', data: { content: 'moved', date, contact_id: created.contact } },
        { method: 'POST', path: '/contacts/duplicates/dismiss', data: { contact_a_id: ownContact, contact_b_id: created.contact } },
      ]) {
        const response = await userB.fetch(`${API_BASE_URL}${probe.path}`, {
          method: probe.method,
          ...(probe.data ? { data: probe.data } : {}),
        });
        expect.soft(response.status(), `${probe.method} ${probe.path}`).toBe(404);
      }

      expect(await snapshot(userA, created)).toEqual(before);
    } finally {
      await userB.dispose();
      await deleteResources(userA, created);
      await userA.dispose();
    }
  });

  test('a non-admin is refused on the admin user routes', async ({ page }) => {
    const me = await (await page.request.get(`${API_BASE_URL}/users/me`)).json();
    const adminId = me.id ?? me.ID;
    expect(adminId).toBeTruthy();
    const userB = await loginAsNewUser(USER_B);

    try {
      for (const probe of [
        { method: 'GET', path: '/admin/users' },
        { method: 'GET', path: `/admin/users/${adminId}` },
        { method: 'PATCH', path: `/admin/users/${adminId}`, data: { is_admin: false } },
        { method: 'DELETE', path: `/admin/users/${adminId}` },
      ]) {
        const response = await userB.fetch(`${API_BASE_URL}${probe.path}`, {
          method: probe.method,
          ...(probe.data ? { data: probe.data } : {}),
        });
        expect.soft(response.status(), `${probe.method} ${probe.path}`).toBe(403);
      }
    } finally {
      await userB.dispose();
    }
  });
});

interface Resources {
  contact: number;
  otherContact: number;
  relationship: number;
  employment: number;
  note: number;
  activity: number;
  reminder: number;
  completion: number;
  webhook: number;
  calendar: number;
  apiToken: number;
}

async function postJson(context: APIRequestContext, path: string, data: unknown): Promise<any> {
  const response = await context.post(`${API_BASE_URL}${path}`, { data });
  expect(response.ok(), `POST ${path}: ${response.status()} ${await response.text()}`).toBeTruthy();
  return response.json();
}

// Fails on a missing ID: a probe of /notes/undefined would pass as a 404.
function idOf(body: any, ...wrappers: string[]): number {
  const wrapped = wrappers.map((key) => body?.[key]).find(Boolean) ?? body;
  const id = wrapped.ID ?? wrapped.id;
  expect(Number.isInteger(id), `no ID in ${JSON.stringify(body)}`).toBeTruthy();
  return id;
}

async function createResources(userA: APIRequestContext, stamp: string): Promise<Resources> {
  const date = new Date().toISOString();
  const contact = idOf(
    await postJson(userA, '/contacts', { firstname: `E2EFixture${stamp}`, lastname: 'Isolated' }),
    'contact'
  );
  const otherContact = idOf(
    await postJson(userA, '/contacts', { firstname: `E2EFixture${stamp}`, lastname: 'Other' }),
    'contact'
  );
  const relationship = idOf(
    await postJson(userA, `/contacts/${contact}/relationships`, { name: 'Rel', type: 'Friend' }),
    'relationship'
  );
  const employment = idOf(
    await postJson(userA, `/contacts/${contact}/employment-history`, { organization: 'Org' }),
    'employment_history',
    'employment'
  );
  const note = idOf(await postJson(userA, `/contacts/${contact}/notes`, { content: 'secret', date }), 'note');
  const activity = idOf(
    await postJson(userA, '/activities', { title: 'secret', date, contact_ids: [contact] }),
    'activity'
  );
  const reminder = idOf(
    await postJson(userA, `/contacts/${contact}/reminders`, {
      message: 'secret',
      remind_at: date,
      recurrence: 'monthly',
      contact_id: contact,
    }),
    'reminder'
  );
  await postJson(userA, `/reminders/${reminder}/complete`, {});
  const completions = await (await userA.get(`${API_BASE_URL}/contacts/${contact}/reminder-completions`)).json();
  const completion = idOf((completions.completions ?? completions)[0]);
  const webhook = idOf(
    await postJson(userA, '/webhooks', {
      name: 'hook',
      url: 'https://203.0.113.1/hook', // reserved address, so a stray delivery stays offline
      events: ['contact.created'],
      is_active: false,
    }),
    'webhook'
  );
  const calendar = idOf(
    await postJson(userA, '/calendars', {
      name: 'cal',
      url: 'https://203.0.113.1/cal.ics', // reserved address, so a stray sync stays offline
      sync_enabled: false,
    }),
    'calendar',
    'subscription'
  );
  const apiToken = idOf(await postJson(userA, '/api-tokens', { name: `e2e-${stamp}` }), 'api_token', 'token_info');
  return {
    contact,
    otherContact,
    relationship,
    employment,
    note,
    activity,
    reminder,
    completion,
    webhook,
    calendar,
    apiToken,
  };
}

async function deleteResources(userA: APIRequestContext, r: Resources): Promise<void> {
  for (const path of [
    `/webhooks/${r.webhook}`,
    `/calendars/${r.calendar}`,
    `/api-tokens/${r.apiToken}`,
    `/activities/${r.activity}`,
    `/contacts/${r.contact}`,
    `/contacts/${r.otherContact}`,
  ]) {
    await userA.delete(`${API_BASE_URL}${path}`).catch(() => {});
  }
}

// What userA can see of everything they own; userB's attempts must not change it.
async function snapshot(userA: APIRequestContext, r: Resources): Promise<Record<string, unknown>> {
  const paths = [
    `/contacts/${r.contact}`,
    `/contacts/${r.otherContact}`,
    `/contacts/${r.contact}/relationships`,
    `/contacts/${r.contact}/employment-history`,
    `/contacts/${r.contact}/reminder-completions`,
    `/notes/${r.note}`,
    `/activities/${r.activity}`,
    `/reminders/${r.reminder}`,
    `/webhooks/${r.webhook}`,
    `/webhooks/${r.webhook}/deliveries`,
    '/calendars',
    '/api-tokens',
  ];
  const result: Record<string, unknown> = {};
  for (const path of paths) {
    const response = await userA.get(`${API_BASE_URL}${path}`);
    expect(response.status(), `userA reading ${path}`).toBe(200);
    result[path] = await response.json();
  }
  return result;
}

interface Probe {
  method: 'GET' | 'POST' | 'PUT' | 'DELETE';
  path: string;
  data?: unknown;
  multipart?: Record<string, { name: string; mimeType: string; buffer: Buffer }>;
  expected: number;
}

function crossUserProbes(r: Resources): Probe[] {
  const date = new Date().toISOString();
  const contact = `/contacts/${r.contact}`;
  const notFound = (method: Probe['method'], path: string, data?: unknown): Probe => ({
    method,
    path,
    data,
    expected: 404,
  });
  const reminderBody = { message: 'hacked', remind_at: date, recurrence: 'once', contact_id: r.contact };
  const webhookBody = { name: 'hacked', url: 'https://example.com/x', events: ['contact.created'], is_active: true };

  return [
    notFound('GET', contact),
    notFound('PUT', contact, { firstname: 'Hacked' }),
    notFound('DELETE', contact),
    notFound('POST', `${contact}/archive`),
    notFound('POST', `${contact}/unarchive`),
    notFound('GET', `${contact}/merge-preview?source_id=${r.otherContact}`),
    notFound('POST', `${contact}/merge`, { source_id: r.otherContact }),
    notFound('GET', `${contact}/vcf`),
    notFound('GET', `${contact}/profile_picture`),
    {
      method: 'POST',
      path: `${contact}/profile_picture`,
      multipart: { photo: { name: 'photo.png', mimeType: 'image/png', buffer: Buffer.from('not an image') } },
      expected: 404,
    },
    notFound('GET', `${contact}/incoming-relationships`),
    notFound('GET', `${contact}/notes`),
    notFound('POST', `${contact}/notes`, { content: 'hacked', date }),
    notFound('GET', `${contact}/activities`),
    notFound('GET', `${contact}/reminders`),
    notFound('POST', `${contact}/reminders`, reminderBody),
    notFound('GET', `${contact}/reminder-completions`),

    notFound('GET', `${contact}/relationships`),
    notFound('POST', `${contact}/relationships`, { name: 'Hacked', type: 'Friend' }),
    notFound('PUT', `${contact}/relationships/${r.relationship}`, { name: 'Hacked', type: 'Friend' }),
    notFound('DELETE', `${contact}/relationships/${r.relationship}`),

    notFound('GET', `${contact}/employment-history`),
    notFound('POST', `${contact}/employment-history`, { organization: 'Hacked' }),
    notFound('PUT', `${contact}/employment-history/${r.employment}`, { organization: 'Hacked' }),
    notFound('DELETE', `${contact}/employment-history/${r.employment}`),
    notFound('POST', `${contact}/employment-history/${r.employment}/end`),

    notFound('GET', `/notes/${r.note}`),
    notFound('PUT', `/notes/${r.note}`, { content: 'hacked', date }),
    notFound('DELETE', `/notes/${r.note}`),

    notFound('GET', `/activities/${r.activity}`),
    notFound('PUT', `/activities/${r.activity}`, { title: 'hacked', date }),
    notFound('DELETE', `/activities/${r.activity}`),

    notFound('GET', `/reminders/${r.reminder}`),
    notFound('PUT', `/reminders/${r.reminder}`, reminderBody),
    notFound('POST', `/reminders/${r.reminder}/complete`),
    notFound('DELETE', `/reminders/${r.reminder}`),
    notFound('DELETE', `/reminder-completions/${r.completion}`),

    notFound('GET', `/webhooks/${r.webhook}`),
    notFound('PUT', `/webhooks/${r.webhook}`, webhookBody),
    notFound('DELETE', `/webhooks/${r.webhook}`),
    notFound('POST', `/webhooks/${r.webhook}/test`),
    notFound('GET', `/webhooks/${r.webhook}/deliveries`),

    notFound('PUT', `/calendars/${r.calendar}`, { name: 'hacked', url: 'https://example.com/x.ics' }),
    notFound('DELETE', `/calendars/${r.calendar}`),
    notFound('POST', `/calendars/${r.calendar}/sync`),

    notFound('DELETE', `/api-tokens/${r.apiToken}`),
  ];
}
