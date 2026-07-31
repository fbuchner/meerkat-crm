// Contact merge/dedupe API calls -- ticket N1
// (docs/fork-plan/tickets/01-N1-contact-merge.md). Mirrors backend/models/
// contact_merge.go's DTOs by hand -- no dynamic schema endpoint exists
// anywhere in this codebase, so this must be kept in sync manually if the
// backend shape changes.
import { apiFetch, API_BASE_URL, getAuthHeaders, parseErrorResponse } from './client';
import { ContactValue, ContactAddress } from './contacts';

export interface ContactMergeFieldConflict {
  field: string;
  label: string;
  keeper_value: string;
  loser_value: string;
}

export interface ContactMergeResolution {
  emails: ContactValue[];
  phones: ContactValue[];
  addresses: ContactAddress[];
  urls: ContactValue[];
  impps: ContactValue[];
  circles: string[];
  custom_fields: Record<string, string>;
  resolved_scalars: Record<string, string>;
  conflicts: ContactMergeFieldConflict[];
  field_value_conflicts: ContactMergeFieldConflict[];
}

export interface ContactMergeAssociationCounts {
  notes: number;
  activities: number;
  reminders: number;
  reminder_completions: number;
  relationship_edges: number;
  household_memberships: number;
  circle_memberships: number;
  tags: number;
  life_events: number;
  life_event_references: number;
  field_values: number;
  contact_sync_links: number;
}

export interface ContactMergePreviewResponse {
  keep_id: number;
  merge_id: number;
  resolution: ContactMergeResolution;
  association_counts: ContactMergeAssociationCounts;
}

// previewContactMerge computes the merge outcome without writing anything --
// call this first, let the user resolve resolution.conflicts +
// resolution.field_value_conflicts, then call commitContactMerge with those
// choices.
export async function previewContactMerge(keepId: number, mergeId: number): Promise<ContactMergePreviewResponse> {
  const response = await apiFetch(`${API_BASE_URL}/contacts/merge/preview`, {
    method: 'POST', headers: getAuthHeaders(),
    body: JSON.stringify({ keep_id: keepId, merge_id: mergeId }),
  });
  if (!response.ok) throw await parseErrorResponse(response);
  return response.json();
}

// commitContactMerge merges mergeId into keepId. resolutions must cover
// every field in a freshly-fetched preview's conflicts AND
// field_value_conflicts (keyed by ContactMergeFieldConflict.field in both
// cases) -- the backend recomputes conflicts fresh at commit time and
// rejects with the specific missing field(s) if any is absent, so a stale
// preview can never silently be trusted.
export async function commitContactMerge(
  keepId: number,
  mergeId: number,
  resolutions: Record<string, string>
): Promise<{ message: string; contact: unknown }> {
  const response = await apiFetch(`${API_BASE_URL}/contacts/merge`, {
    method: 'POST', headers: getAuthHeaders(),
    body: JSON.stringify({ keep_id: keepId, merge_id: mergeId, resolutions }),
  });
  if (!response.ok) throw await parseErrorResponse(response);
  return response.json();
}
