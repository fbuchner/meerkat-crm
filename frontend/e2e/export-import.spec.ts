import { test, expect, loginAsNewUser, uniqueUser } from './fixtures';
import { API_BASE_URL } from './global-setup';

test.describe('Export and import', () => {
  test('a vCard export imports into a clean user with the same data', async () => {
    const owner = await loginAsNewUser(uniqueUser('e2e_export'));
    const clean = await loginAsNewUser(uniqueUser('e2e_import'));

    try {
      const created = await owner.post(`${API_BASE_URL}/contacts`, {
        data: {
          firstname: 'Exported',
          lastname: 'Person',
          emails: [{ type: 'home', value: 'exported@example.com' }],
          phones: [{ type: 'mobile', value: '+1 555-0199' }],
          birthday: '1990-03-15',
          organization: 'Acme',
        },
      });
      expect(created.ok()).toBeTruthy();

      const exported = await owner.get(`${API_BASE_URL}/export/vcf`);
      expect(exported.ok()).toBeTruthy();
      const vcf = await exported.text();
      expect(vcf).toContain('BEGIN:VCARD');

      const upload = await clean.post(`${API_BASE_URL}/contacts/import/vcf/upload`, {
        multipart: { file: { name: 'contacts.vcf', mimeType: 'text/vcard', buffer: Buffer.from(vcf) } },
      });
      expect(upload.ok(), await upload.text()).toBeTruthy();
      const preview = await upload.json();
      expect(preview.rows).toHaveLength(1);

      const confirm = await clean.post(`${API_BASE_URL}/contacts/import/vcf/confirm`, {
        data: { session_id: preview.session_id, actions: [{ row_index: 0, action: 'add' }] },
      });
      expect(confirm.ok(), await confirm.text()).toBeTruthy();
      expect((await confirm.json()).created).toBe(1);

      const list = await (await clean.get(`${API_BASE_URL}/contacts?limit=10`)).json();
      expect(list.contacts).toHaveLength(1);
      const imported = await (await clean.get(`${API_BASE_URL}/contacts/${list.contacts[0].ID}`)).json();
      expect(imported).toMatchObject({
        firstname: 'Exported',
        lastname: 'Person',
        birthday: '1990-03-15',
        organization: 'Acme',
      });
      expect(imported.emails.map((e: { value: string }) => e.value)).toEqual(['exported@example.com']);
      expect(imported.phones.map((p: { value: string }) => p.value)).toEqual(['+1 555-0199']);
    } finally {
      await owner.dispose();
      await clean.dispose();
    }
  });
});
