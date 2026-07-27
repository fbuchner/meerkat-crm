package main

import (
	"context"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

// TestOpenAPISpecValidates loads and validates backend/openapi.yaml (WP-71
// item 6, docs/fork-plan/50-integration-and-rebrand.md): the spec must be
// well-formed OpenAPI 3.0 and internally consistent (every $ref resolves,
// every schema/path is structurally valid). This is a Go-based check
// (github.com/getkin/kin-openapi, added as a test-only dependency) rather
// than a Node/swagger-cli install, per this WP's own tooling note (no local
// Node toolchain is assumed available; Go/Docker is this repo's confirmed
// toolchain — docs/fork-plan/70-environment.md).
func TestOpenAPISpecValidates(t *testing.T) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false

	doc, err := loader.LoadFromFile("openapi.yaml")
	if err != nil {
		t.Fatalf("failed to load openapi.yaml: %v", err)
	}

	if err := doc.Validate(context.Background()); err != nil {
		t.Fatalf("openapi.yaml failed validation: %v", err)
	}

	// Spot-check the schemas/paths this WP added actually exist, so a future
	// edit that silently deletes a path/schema (rather than just drifting
	// its fields) is caught here too.
	wantSchemas := []string{
		"ContactSummary", "ContactSummaryWithRelations", "ContactRecordInput",
		"ContactRecordResponse", "Card", "CRMEnvelope", "Passthrough",
		"ColumnMapping", "ImportUploadResponse", "ImportPreviewRequest",
		"DuplicateMatch", "ImportRowPreview", "ImportPreviewResponse",
		"RowImportAction", "ImportConfirmRequest", "ImportResult",
		"ContactSubscriptionInput", "ContactSubscriptionResponse",
	}
	for _, name := range wantSchemas {
		if _, ok := doc.Components.Schemas[name]; !ok {
			t.Errorf("openapi.yaml is missing expected schema %q", name)
		}
	}

	wantPaths := []string{
		"/contacts", "/contacts/{id}", "/export/vcf", "/export/jscontact",
		// WP-73b import + contact-subscription endpoints (added alongside
		// this test's own extension, so a future path deletion is caught
		// here rather than silently drifting like the original 4-path spec
		// did for months before this WP).
		"/contacts/import/upload", "/contacts/import/vcf/upload",
		"/contacts/import/jscontact/upload", "/contacts/import/preview",
		"/contacts/import/confirm", "/contacts/import/vcf/confirm",
		"/contact-subscriptions", "/contact-subscriptions/{id}",
		"/contact-subscriptions/{id}/sync",
	}
	for _, p := range wantPaths {
		if doc.Paths.Find(p) == nil {
			t.Errorf("openapi.yaml is missing expected path %q", p)
		}
	}

	// NOTE: this spec (and this test's wantPaths) still only covers the
	// nested-Card contact API (WP-71) and the import/contact-subscription
	// surface (WP-73b) — most of the pre-existing REST API (activities,
	// calendars, notes, reminders, users, webhooks, ...) was never
	// documented here and is out of scope for this pass. A repo-wide
	// route-vs-spec audit is a separate, larger effort (tracked as Phase 3
	// pre-existing-surface work), not something to assert here.

	// GET /contacts must document the query mechanics Gap 2 requires be
	// preserved (page/limit/search/sort/order/include_archived/archived/
	// circle/includes) and must NOT document fields= (Gap 3: removed, not
	// just undocumented-but-still-there).
	contactsGet := doc.Paths.Find("/contacts").Get
	wantParams := []string{"page", "limit", "search", "sort", "order", "include_archived", "archived", "circle", "includes"}
	gotParams := map[string]bool{}
	for _, p := range contactsGet.Parameters {
		gotParams[p.Value.Name] = true
	}
	for _, name := range wantParams {
		if !gotParams[name] {
			t.Errorf("GET /contacts is missing documented query param %q", name)
		}
	}
	if gotParams["fields"] {
		t.Error("GET /contacts still documents fields=, which WP-71 Gap 3 removed")
	}

	// POST/PUT /contacts must reference ContactRecordInput as their request
	// body (the nested shape), not a flat DTO.
	postBody := doc.Paths.Find("/contacts").Post.RequestBody.Value.Content.Get("application/json").Schema
	if postBody.Ref == "" || postBody.Ref != "#/components/schemas/ContactRecordInput" {
		t.Errorf("POST /contacts request body ref = %q, want #/components/schemas/ContactRecordInput", postBody.Ref)
	}
}
