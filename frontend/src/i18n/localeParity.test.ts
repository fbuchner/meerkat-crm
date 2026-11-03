import { describe, expect, test } from 'vitest';

const locales = import.meta.glob<Record<string, unknown>>('./locales/*.json', {
  eager: true,
  import: 'default',
});

function flattenKeys(value: unknown, prefix = ''): string[] {
  if (value === null || typeof value !== 'object') return [prefix];
  return Object.entries(value).flatMap(([key, child]) =>
    flattenKeys(child, prefix ? `${prefix}.${key}` : key)
  );
}

const englishKeys = flattenKeys(locales['./locales/en.json']).sort();
const otherLocales = Object.keys(locales).filter((path) => !path.endsWith('/en.json'));

describe('locale key parity', () => {
  test('finds every translated locale', () => {
    expect(otherLocales.sort()).toEqual([
      './locales/de.json',
      './locales/es.json',
      './locales/fr.json',
      './locales/it.json',
      './locales/zh.json',
    ]);
  });

  for (const path of otherLocales) {
    test(`${path} has the same keys as en.json`, () => {
      expect(flattenKeys(locales[path]).sort()).toEqual(englishKeys);
    });
  }
});
