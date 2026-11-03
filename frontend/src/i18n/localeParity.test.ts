import { readdirSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, test } from 'vitest';

const LOCALES_DIR = resolve(__dirname, 'locales');

function flattenKeys(value: unknown, prefix = ''): string[] {
  if (value === null || typeof value !== 'object') return [prefix];
  return Object.entries(value).flatMap(([key, child]) =>
    flattenKeys(child, prefix ? `${prefix}.${key}` : key)
  );
}

function readKeys(file: string): string[] {
  return flattenKeys(JSON.parse(readFileSync(resolve(LOCALES_DIR, file), 'utf8'))).sort();
}

const otherLocales = readdirSync(LOCALES_DIR).filter(
  (file) => file.endsWith('.json') && file !== 'en.json'
);

describe('locale key parity', () => {
  test('finds every translated locale', () => {
    expect(otherLocales.sort()).toEqual(['de.json', 'es.json', 'fr.json', 'it.json', 'zh.json']);
  });

  for (const file of otherLocales) {
    test(`${file} has the same keys as en.json`, () => {
      expect(readKeys(file)).toEqual(readKeys('en.json'));
    });
  }
});
