import { describe, test, expect } from 'vitest';
import { resolveNetworkFilters, DEFAULT_NETWORK_FILTERS } from './networkFilters';

describe('resolveNetworkFilters', () => {
  test('returns defaults when nothing is stored', () => {
    expect(resolveNetworkFilters(null)).toEqual(DEFAULT_NETWORK_FILTERS);
    expect(resolveNetworkFilters(undefined)).toEqual(DEFAULT_NETWORK_FILTERS);
    expect(resolveNetworkFilters('')).toEqual(DEFAULT_NETWORK_FILTERS);
  });

  test('restores a full stored filter set', () => {
    const stored = JSON.stringify({
      selectedCircle: 'Family',
      showRelationships: true,
      showActivities: false,
      showCircles: true,
    });
    expect(resolveNetworkFilters(stored)).toEqual({
      selectedCircle: 'Family',
      showRelationships: true,
      showActivities: false,
      showCircles: true,
    });
  });

  test('falls back per field when the stored object is partial', () => {
    const stored = JSON.stringify({ showCircles: true });
    expect(resolveNetworkFilters(stored)).toEqual({
      ...DEFAULT_NETWORK_FILTERS,
      showCircles: true,
    });
  });

  test('returns defaults for corrupt JSON', () => {
    expect(resolveNetworkFilters('{not json')).toEqual(DEFAULT_NETWORK_FILTERS);
    expect(resolveNetworkFilters('null')).toEqual(DEFAULT_NETWORK_FILTERS);
  });

  test('does not share state between calls', () => {
    const first = resolveNetworkFilters(null);
    first.showCircles = true;
    expect(resolveNetworkFilters(null).showCircles).toBe(false);
  });
});
