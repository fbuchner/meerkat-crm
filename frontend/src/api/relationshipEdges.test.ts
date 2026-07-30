import { describe, test, expect } from 'vitest';
import {
  RELATIONSHIP_EDGE_TYPE_TOKENS,
  RELATIONSHIP_EDGE_TYPES,
  RelationshipEdge,
  RelationshipEdgeType,
  getEffectiveType,
  getDisplayLabel,
  toBackendType,
  getOtherPartyId,
} from './relationshipEdges';

// §3d WP3 (docs/fork-plan/95-backlog-and-priorities.md): the direction-
// resolution logic below is the highest-risk code in this WP -- getting it
// backwards would silently create/display incorrectly-labeled relationships.
// Every asymmetric pair is asserted explicitly in both directions; every
// symmetric token is asserted to read identically regardless of viewed side.

function makeEdge(overrides: Partial<RelationshipEdge> = {}): RelationshipEdge {
  return {
    id: 'edge-1',
    source_id: 'alice',
    target_id: 'bob',
    type: 'parent_of',
    directional: true,
    source: 'user-confirmed',
    confidence: 1.0,
    status: 'confirmed',
    sensitivity: 'normal',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

describe('getEffectiveType / getDisplayLabel', () => {
  test('parent_of/child_of: viewed=target reads type directly, viewed=source reads the inverse', () => {
    // Alice is Bob's parent (source=alice, type=parent_of, target=bob).
    const edge = makeEdge({ source_id: 'alice', target_id: 'bob', type: 'parent_of' });
    // Viewing Bob's page: the other party (Alice) is his parent.
    expect(getEffectiveType(edge, 'bob')).toBe('parent_of');
    expect(getDisplayLabel(edge, 'bob')).toBe('relationships.types.parent_of');
    // Viewing Alice's page: the other party (Bob) is her child.
    expect(getEffectiveType(edge, 'alice')).toBe('child_of');
    expect(getDisplayLabel(edge, 'alice')).toBe('relationships.types.child_of');
  });

  test('mentor_of/mentee_of: both directions', () => {
    // Alice mentors Bob (source=alice, type=mentee_of would mean alice is
    // the mentee -- construct explicitly: type describes source's role, so
    // "alice is bob's mentor" is type: mentor_of, source: alice, target: bob).
    const edge = makeEdge({ source_id: 'alice', target_id: 'bob', type: 'mentor_of' });
    // Viewing Bob's page: the other party (Alice) is his mentor.
    expect(getEffectiveType(edge, 'bob')).toBe('mentor_of');
    // Viewing Alice's page: the other party (Bob) is her mentee.
    expect(getEffectiveType(edge, 'alice')).toBe('mentee_of');
  });

  test('owns/owned_by: both directions, curated labels not derived from token name', () => {
    // Alice owns Bob (a pet) -- source=alice, type=owns, target=bob.
    const edge = makeEdge({ source_id: 'alice', target_id: 'bob', type: 'owns' });
    // Viewing Bob's (the pet's) page: the other party (Alice) is the owner.
    expect(getEffectiveType(edge, 'bob')).toBe('owns');
    expect(getDisplayLabel(edge, 'bob')).toBe('relationships.types.owns');
    // Viewing Alice's page: the other party (Bob) is her pet.
    expect(getEffectiveType(edge, 'alice')).toBe('owned_by');
    expect(getDisplayLabel(edge, 'alice')).toBe('relationships.types.owned_by');
  });

  const symmetricTokens: RelationshipEdgeType[] = [
    'spouse_of', 'sibling_of', 'friend_of', 'roommate_of', 'partner_of',
    'co_parent_of', 'gets_along_with', 'conflicts_with', 'related_to',
  ];
  test.each(symmetricTokens)('%s is symmetric: label identical regardless of viewed side', (token) => {
    const edge = makeEdge({ source_id: 'alice', target_id: 'bob', type: token });
    expect(getEffectiveType(edge, 'alice')).toBe(token);
    expect(getEffectiveType(edge, 'bob')).toBe(token);
    expect(RELATIONSHIP_EDGE_TYPES[token].symmetric).toBe(true);
  });

  test('unregistered type falls back to related_to without throwing', () => {
    const edge = makeEdge({ type: 'some_future_token_this_frontend_mirror_does_not_know_about' });
    expect(() => getEffectiveType(edge, 'bob')).not.toThrow();
    expect(getEffectiveType(edge, 'bob')).toBe('related_to');
    // Viewed=source with an unregistered type: metaFor's own fallback makes
    // this related_to's inverse, which is itself related_to.
    expect(getEffectiveType(edge, 'alice')).toBe('related_to');
  });

  test('every registered token has a valid inverse entry (self-consistency)', () => {
    for (const token of RELATIONSHIP_EDGE_TYPE_TOKENS) {
      const meta = RELATIONSHIP_EDGE_TYPES[token];
      expect(RELATIONSHIP_EDGE_TYPES[meta.inverse]).toBeDefined();
      // Symmetric tokens must be their own inverse.
      if (meta.symmetric) {
        expect(meta.inverse).toBe(token);
      }
    }
  });
});

describe('toBackendType', () => {
  test('identity when viewed is not source (create-mode path)', () => {
    for (const token of RELATIONSHIP_EDGE_TYPE_TOKENS) {
      expect(toBackendType(token, false)).toBe(token);
    }
  });

  test('round-trips against getEffectiveType for every token, both directions', () => {
    for (const token of RELATIONSHIP_EDGE_TYPE_TOKENS) {
      const edge = makeEdge({ source_id: 'alice', target_id: 'bob', type: token });

      const effectiveAtTarget = getEffectiveType(edge, 'bob'); // viewed is target
      expect(toBackendType(effectiveAtTarget, false)).toBe(token);

      const effectiveAtSource = getEffectiveType(edge, 'alice'); // viewed is source
      expect(toBackendType(effectiveAtSource, true)).toBe(token);
    }
  });
});

describe('getOtherPartyId', () => {
  test('returns target when viewed is source, and vice versa', () => {
    const edge = makeEdge({ source_id: 'alice', target_id: 'bob' });
    expect(getOtherPartyId(edge, 'alice')).toBe('bob');
    expect(getOtherPartyId(edge, 'bob')).toBe('alice');
  });
});
