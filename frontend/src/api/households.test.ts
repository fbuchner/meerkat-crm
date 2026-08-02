import { describe, test, expect, vi, afterEach } from 'vitest';
import { suggestHouseholdRelationships } from './households';
import { getRelationshipEdges } from './relationshipEdges';
import { RelationshipEdge } from './relationshipEdges';

afterEach(() => {
  vi.unstubAllGlobals();
});

// T1 (docs/fork-plan/tickets/09-T1-households.md): the trigger endpoint and
// the review-inbox query must agree on the shape of a generated suggestion —
// the trigger's output is exactly what feeds RelationshipEdgeList's suggested
// section on each member's contact page, for the first time against real data.

const generatedSuggestion: RelationshipEdge = {
  id: 'edge-1',
  source_id: 'alice-uid',
  target_id: 'bob-uid',
  type: 'spouse_of',
  directional: false,
  source: 'household-inferred',
  confidence: 0.8,
  status: 'suggested',
  sensitivity: 'normal',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
};

describe('suggestHouseholdRelationships', () => {
  test('POSTs to the trigger endpoint and returns the generated edges', async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        message: 'Relationship suggestions generated',
        household_id: 'h1',
        suggested_edges: [generatedSuggestion],
        total: 1,
      }),
    });
    vi.stubGlobal('fetch', fetchMock);

    const result = await suggestHouseholdRelationships('h1');

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toContain('/households/h1/suggest-relationships');
    expect(init.method).toBe('POST');
    expect(result.total).toBe(1);
    expect(result.suggested_edges[0]).toEqual(generatedSuggestion);
    expect(result.suggested_edges[0].status).toBe('suggested');
  });
});

describe('review loop', () => {
  test('a generated suggestion surfaces through the status=suggested inbox query that feeds RelationshipEdgeList', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          relationship_edges: [generatedSuggestion],
          total: 1,
          page: 1,
          limit: 100,
        }),
      })
    );

    const response = await getRelationshipEdges({ contactId: 'alice-uid', status: 'suggested' });

    expect(response.relationship_edges[0]).toEqual(generatedSuggestion);
    expect(response.relationship_edges[0].status).toBe('suggested');
    expect(response.relationship_edges[0].source).toBe('household-inferred');
  });
});
