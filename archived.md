# Contact Archiving Feature Specification

This document specifies the product behavior for archiving contacts in Meerkat CRM.

## Overview

Archiving allows users to hide contacts they no longer actively maintain relationships with, without permanently deleting them. Archived contacts retain their history but are de-emphasized throughout the application.

## Data Model

Add a boolean `archived` field to the Contact model:
- Default value: `false`
- When `true`, the contact is considered archived

## Behavior by Feature

### Contacts List Page (`/contacts`)

- **Default view**: Only shows active (non-archived) contacts
- **Filter toggle**: Add a toggle/checkbox "Show archived" to include archived contacts in the list
- **Visual distinction**: Archived contacts displayed with visual indication (e.g., grayed out, badge, or muted styling)
- **Sorting**: When showing archived, they appear mixed with active contacts according to current sort order

### Global Search (Full Results Page)

- **Scope**: Searches through ALL contacts, including archived
- **Ordering**: Archived contacts appear **last** in the search results, after all active contacts
- **Visual distinction**: Archived contacts marked with visual indication

### Instant Search (Header Autocomplete)

- **Scope**: Only suggests **active** (non-archived) contacts
- **Rationale**: Quick navigation should prioritize active relationships

### Dashboard

- **Random contacts**: Excludes archived contacts
- **Upcoming birthdays**: Excludes archived contacts
- **Reminders widget**: N/A (reminders deleted on archive)

### Network Graph

- **Scope**: Excludes archived contacts entirely
- **Rationale**: Network visualization should represent active relationship network

### Reminders

- **On archive**: All reminders associated with the contact are **deleted**
- **Rationale**: User chose "delete" - archiving means ending active reminder tracking for that contact

### Activities

- **Behavior**: Activities remain intact and associated with the archived contact
- **Display**: Activities involving archived contacts still appear in activity views
- **Rationale**: Activities often involve multiple contacts; historical record should be preserved

### Relationships

- **Behavior**: Relationship links to/from archived contacts remain intact
- **Display**: When viewing an active contact, relationships to archived contacts are still visible

### Export

- **Behavior**: Always includes archived contacts in exports (CSV, VCF, JSON)
- **No filter option needed**: Export represents full data backup

### Notes

- **No changes needed**: Notes page only displays notes not associated with any contact

## User Actions

### Archive a Contact

- **Location**: Contact detail page (add archive button/action)
- **Confirmation**: Show confirmation dialog warning that reminders will be deleted
- **Result**: Sets `archived = true`, deletes associated reminders

### Unarchive a Contact

- **Location**: Contact detail page (when viewing archived contact)
- **Result**: Sets `archived = false`
- **Note**: Deleted reminders are not restored

### Bulk Archiving

- **Not implemented** in initial version
- Single contact archiving only

## Visual Design Guidelines

Archived contacts should be visually distinct but not jarring:
- Option A: Reduced opacity (e.g., 60% opacity)
- Option B: "Archived" chip/badge next to name
- Option C: Muted text color + italic
- Recommendation: Use a subtle "Archived" chip for clarity

## API Changes

### Contact Endpoints

| Endpoint | Change |
|----------|--------|
| `GET /contacts` | Add `?archived=true/false` query param (default: false). Add `?include_archived=true` to include both. |
| `GET /contacts/:id` | No change (returns contact regardless of archived status) |
| `PUT /contacts/:id` | Allow setting `archived` field |
| `GET /contacts/search` | Add `?include_archived=true` param, return archived contacts last |

### New Endpoint (Optional)

Consider dedicated archive/unarchive endpoints for clarity:
- `POST /contacts/:id/archive`
- `POST /contacts/:id/unarchive`

## Implementation Phases

### Phase 1: Backend
1. Add `archived` boolean field to Contact model + migration
2. Update contact list endpoint to filter by archived status
3. Update search endpoint to support include_archived and ordering
4. Handle reminder deletion on archive

### Phase 2: Frontend - Contacts Page
1. Add "Show archived" filter toggle
2. Update contact cards with archived visual indication
3. Add archive/unarchive action to contact detail page
4. Add confirmation dialog for archiving (warns about reminder deletion)

### Phase 3: Frontend - Search
1. Update header autocomplete to exclude archived contacts
2. Update search results page to show archived contacts last with visual indication

### Phase 4: Frontend - Other Views
1. Update dashboard queries to exclude archived contacts
2. Update network graph to exclude archived contacts

## Out of Scope

The following are explicitly not included in this feature:
- Archive reason/notes
- Automatic archive suggestions
- Bulk archiving
- Archive date tracking
- Archived contacts in notes page (already not applicable)

## Testing Considerations

- Verify archived contacts excluded from dashboard
- Verify archived contacts excluded from network graph
- Verify header autocomplete excludes archived
- Verify full search includes archived (shown last)
- Verify reminders deleted on archive
- Verify activities preserved after archiving
- Verify export includes archived contacts
- Verify unarchive works correctly
