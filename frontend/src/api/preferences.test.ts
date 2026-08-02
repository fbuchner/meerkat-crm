import { describe, test, expect, vi, afterEach } from 'vitest';
import { getPreferences, createPreference } from './preferences';

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('getPreferences', () => {
  test('requests the entity-scoped endpoint and parses the response', async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        preferences: [
          {
            id: 'p1',
            entity_id: 'alice-uid',
            category: 'food',
            value: 'Vegetarian',
            sensitivity: 'normal',
          },
        ],
        total: 1,
        page: 1,
        limit: 100,
      }),
    });
    vi.stubGlobal('fetch', fetchMock);

    const response = await getPreferences({ entityId: 'alice-uid' });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url] = fetchMock.mock.calls[0];
    expect(url).toContain('/preferences?');
    expect(url).toContain('entity_id=alice-uid');
    expect(response.total).toBe(1);
    expect(response.preferences[0].value).toBe('Vegetarian');
  });
});

describe('createPreference', () => {
  test('POSTs the input and unwraps the created preference', async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        message: 'Preference created successfully',
        preference: { id: 'p1', entity_id: 'alice-uid', category: 'food', value: 'Vegan', sensitivity: 'normal' },
      }),
    });
    vi.stubGlobal('fetch', fetchMock);

    const result = await createPreference({
      entity_id: 'alice-uid',
      category: 'food',
      value: 'Vegan',
      sensitivity: 'normal',
    });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toContain('/preferences');
    expect(init.method).toBe('POST');
    expect(result.id).toBe('p1');
    expect(result.value).toBe('Vegan');
  });
});
