// RelationshipEdge API calls -- §3d WP3 (docs/fork-plan/95-backlog-and-priorities.md),
// replaces api/relationships.ts's legacy models.Relationship-backed calls.
import { apiFetch, API_BASE_URL, getAuthHeaders, parseErrorResponse } from './client';

// Mirrors backend/models/relationship_type_registry.go's relationTypeRegistry
// exactly (token -> Inverse/Symmetric). No dynamic "list valid types" endpoint
// exists anywhere in this codebase -- every other enum (Gender, DateFormat,
// ...) is a hardcoded frontend mirror of the backend's `oneof`/custom
// validators, and this follows the same convention. MUST be kept in sync by
// hand with relationship_type_registry.go if that file changes.
export type RelationshipEdgeType =
  | 'parent_of' | 'child_of' | 'spouse_of' | 'sibling_of' | 'friend_of'
  | 'roommate_of' | 'partner_of' | 'co_parent_of' | 'mentor_of' | 'mentee_of'
  | 'owned_by' | 'owns' | 'gets_along_with' | 'conflicts_with' | 'related_to';

export type RelationshipEdgeSource = 'user-confirmed' | 'household-inferred' | 'imported' | 'ai-suggested';
export type RelationshipEdgeStatus = 'confirmed' | 'suggested';
export type RelationshipEdgeSensitivity = 'normal' | 'private' | 'secret';

interface RelationTypeMeta {
  inverse: RelationshipEdgeType;
  symmetric: boolean;
}

export const RELATIONSHIP_EDGE_TYPES: Record<RelationshipEdgeType, RelationTypeMeta> = {
  parent_of:       { inverse: 'child_of',        symmetric: false },
  child_of:        { inverse: 'parent_of',       symmetric: false },
  spouse_of:       { inverse: 'spouse_of',       symmetric: true },
  sibling_of:      { inverse: 'sibling_of',      symmetric: true },
  friend_of:       { inverse: 'friend_of',       symmetric: true },
  roommate_of:     { inverse: 'roommate_of',     symmetric: true },
  partner_of:      { inverse: 'partner_of',      symmetric: true },
  co_parent_of:    { inverse: 'co_parent_of',    symmetric: true },
  mentor_of:       { inverse: 'mentee_of',       symmetric: false },
  mentee_of:       { inverse: 'mentor_of',       symmetric: false },
  owned_by:        { inverse: 'owns',            symmetric: false },
  owns:            { inverse: 'owned_by',        symmetric: false },
  gets_along_with: { inverse: 'gets_along_with', symmetric: true },
  conflicts_with:  { inverse: 'conflicts_with',  symmetric: true },
  related_to:      { inverse: 'related_to',      symmetric: true },
};

export const RELATIONSHIP_EDGE_TYPE_TOKENS = Object.keys(RELATIONSHIP_EDGE_TYPES) as RelationshipEdgeType[];

export interface RelationshipEdge {
  id: string;
  source_id: string;
  target_id: string;
  // Loosely typed on read (defensive against registry drift/future tokens
  // this frontend mirror hasn't been updated for yet) -- narrowed to
  // RelationshipEdgeType only where this app writes it.
  type: string;
  directional: boolean;
  metadata?: Record<string, unknown>;
  source: RelationshipEdgeSource;
  confidence: number;
  status: RelationshipEdgeStatus;
  sensitivity: RelationshipEdgeSensitivity;
  created_at: string;
  updated_at: string;
}

export interface ThinContactInput {
  name: string;
  gender?: string;
  birthday?: string;
}

export interface RelationshipEdgeInput {
  source_id?: string;
  source_thin?: ThinContactInput;
  target_id?: string;
  target_thin?: ThinContactInput;
  type: RelationshipEdgeType;
  sensitivity?: RelationshipEdgeSensitivity;
  metadata?: Record<string, unknown>;
}

export interface RelationshipEdgesResponse {
  relationship_edges: RelationshipEdge[];
  total: number;
  page: number;
  limit: number;
}

export interface GetRelationshipEdgesParams {
  contactId: string; // Contact.VCardUID
  status?: RelationshipEdgeStatus;
  type?: RelationshipEdgeType;
  page?: number;
  limit?: number;
}

export async function getRelationshipEdges(params: GetRelationshipEdgesParams): Promise<RelationshipEdgesResponse> {
  const { contactId, status, type, page = 1, limit = 100 } = params;
  const queryParams = new URLSearchParams({ contact_id: contactId, page: page.toString(), limit: limit.toString() });
  if (status) queryParams.append('status', status);
  if (type) queryParams.append('type', type);

  const response = await apiFetch(`${API_BASE_URL}/relationship-edges?${queryParams.toString()}`, { headers: getAuthHeaders() });
  if (!response.ok) throw await parseErrorResponse(response);
  return response.json();
}

// NOTE the response shape asymmetry vs update/accept below -- create is the
// only one wrapped in {relationship_edge: ...}.
export async function createRelationshipEdge(input: RelationshipEdgeInput): Promise<RelationshipEdge> {
  const response = await apiFetch(`${API_BASE_URL}/relationship-edges`, {
    method: 'POST', headers: getAuthHeaders(), body: JSON.stringify(input),
  });
  if (!response.ok) throw await parseErrorResponse(response);
  const result = await response.json();
  return result.relationship_edge;
}

// PUT is full-replace; callers editing an existing edge MUST pass the edge's
// own already-resolved source_id/target_id, never *_thin -- resolveRelationshipEndpoint
// (backend) always inserts a new Contact for a non-nil *_thin input, even on
// update, so resubmitting manual-entry fields would silently orphan a new
// thin Contact on every edit.
export async function updateRelationshipEdge(id: string, input: RelationshipEdgeInput): Promise<RelationshipEdge> {
  const response = await apiFetch(`${API_BASE_URL}/relationship-edges/${id}`, {
    method: 'PUT', headers: getAuthHeaders(), body: JSON.stringify(input),
  });
  if (!response.ok) throw await parseErrorResponse(response);
  return response.json(); // raw edge, NOT wrapped -- unlike create
}

export async function deleteRelationshipEdge(id: string): Promise<void> {
  const response = await apiFetch(`${API_BASE_URL}/relationship-edges/${id}`, {
    method: 'DELETE', headers: getAuthHeaders(),
  });
  if (!response.ok) throw await parseErrorResponse(response);
}

// Promotes a suggested edge to confirmed. 409 if already confirmed (surfaces
// via parseErrorResponse/ApiError like any other backend error).
export async function acceptRelationshipEdge(id: string): Promise<RelationshipEdge> {
  const response = await apiFetch(`${API_BASE_URL}/relationship-edges/${id}/accept`, {
    method: 'PATCH', headers: getAuthHeaders(),
  });
  if (!response.ok) throw await parseErrorResponse(response);
  return response.json(); // raw edge, NOT wrapped
}

// Rejecting a suggestion has no dedicated endpoint -- DELETE doubles as
// reject. Exposed as a distinct name so callers/hooks read intent clearly;
// it is literally deleteRelationshipEdge under the hood.
export const rejectRelationshipEdge = deleteRelationshipEdge;

// ---------------------------------------------------------------------------
// Direction resolution -- the highest-risk logic in this file.
//
// `type` describes SourceID's role relative to TargetID (verified against
// backend/models/relationship_type_registry.go and cmd/backfill-relationship-
// edges/migrate.go's own doc comments: e.g. type: "parent_of" means "Source
// is Target's parent"). So when creating from a contact's detail page, the
// viewed contact is always TargetID, and a dropdown label like "Mother"
// (describing the OTHER party) maps directly onto `type`. When *displaying*
// an edge on a contact's page, the inverse applies depending on which side
// the viewed contact occupies -- see getEffectiveType below.
// ---------------------------------------------------------------------------

// Falls back to 'related_to' defensively -- protects against a future/stale
// `edge.type` value not present in RELATIONSHIP_EDGE_TYPES (registry drift)
// crashing the relationships tab instead of degrading to the generic label.
function metaFor(type: string): RelationTypeMeta {
  return RELATIONSHIP_EDGE_TYPES[type as RelationshipEdgeType] ?? RELATIONSHIP_EDGE_TYPES.related_to;
}

// The relation token as it reads from viewedContactUid's perspective ("what
// IS the other party to me"):
//   viewedContactUid === edge.target_id -> the OTHER party is source, and
//     `type` already describes source directly -- no inversion.
//   viewedContactUid === edge.source_id -> the OTHER party is target, whose
//     role is the INVERSE of `type` (e.g. type=parent_of, viewed=source
//     means viewed is the parent, so the OTHER party -- target -- is my
//     child -> inverse token child_of).
export function getEffectiveType(edge: RelationshipEdge, viewedContactUid: string): RelationshipEdgeType {
  const isViewedSource = edge.source_id === viewedContactUid;
  if (isViewedSource) {
    return metaFor(edge.type).inverse;
  }
  // Viewed is target -> `type` already describes the other party (source)
  // directly. Falls back to 'related_to' if edge.type isn't a token this
  // frontend mirror knows about (registry drift).
  return (RELATIONSHIP_EDGE_TYPES[edge.type as RelationshipEdgeType] ? edge.type : 'related_to') as RelationshipEdgeType;
}

// i18n KEY (not a translated string) for the label to show for this edge as
// viewed from viewedContactUid -- caller passes this through t().
export function getDisplayLabel(edge: RelationshipEdge, viewedContactUid: string): string {
  return `relationships.types.${getEffectiveType(edge, viewedContactUid)}`;
}

// Converts a user's type-dropdown pick (always framed as "what is the OTHER
// party to me", matching getDisplayLabel/getEffectiveType) back into the
// backend's `type` value (always "what is SourceID to TargetID"), given
// which side viewedContactUid occupies. Create mode: viewedContactUid is
// always TargetID, so this is the identity function there. Only matters in
// edit mode, when viewedContactUid may be SourceID (editing an edge
// originally created from the OTHER party's page).
export function toBackendType(dropdownToken: RelationshipEdgeType, viewedIsSource: boolean): RelationshipEdgeType {
  return viewedIsSource ? RELATIONSHIP_EDGE_TYPES[dropdownToken].inverse : dropdownToken;
}

// The other endpoint's VCardUID -- whichever of source_id/target_id isn't
// viewedContactUid.
export function getOtherPartyId(edge: RelationshipEdge, viewedContactUid: string): string {
  return edge.source_id === viewedContactUid ? edge.target_id : edge.source_id;
}
