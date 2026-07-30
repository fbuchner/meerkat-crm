package controllers

import (
	"errors"
	apperrors "mycorrhizal/errors"
	"mycorrhizal/middleware"
	"mycorrhizal/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// resolveRelationshipEndpoint resolves one RelationshipEdge endpoint (source
// or target) to a Contact.VCardUID: either verifying ownership of an
// existing contact (id != ""), or creating a new "thin" Contact from thin
// (mirroring cmd/backfill-relationship-edges/migrate.go's field mapping --
// Firstname=thin.Name verbatim, Gender/Birthday passed through). Returns a
// *apperrors.AppError rather than aborting directly (like verifyOwnedContact
// does, life_event_controller.go) because this runs inside a
// db.Transaction closure and must not abort mid-transaction -- the caller
// aborts exactly once, after the transaction resolves, preserving the
// specific NotFound/InvalidInput status rather than flattening every
// failure into a generic 500.
func resolveRelationshipEndpoint(tx *gorm.DB, userID uint, fieldName, id string, thin *models.ThinContactInput) (string, *apperrors.AppError) {
	switch {
	case id != "" && thin != nil:
		return "", apperrors.ErrInvalidInput(fieldName, "specify either an existing contact id or a new contact, not both")
	case id != "":
		var contact models.Contact
		if err := tx.Where("vcard_uid = ? AND user_id = ?", id, userID).First(&contact).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "", apperrors.ErrNotFound("Contact").WithDetails(fieldName, id)
			}
			return "", apperrors.ErrDatabase("Failed to retrieve contact").WithError(err)
		}
		return contact.VCardUID, nil
	case thin != nil:
		contact := models.Contact{UserID: userID, Firstname: thin.Name, Gender: thin.Gender, Birthday: thin.Birthday}
		if err := tx.Create(&contact).Error; err != nil {
			return "", apperrors.ErrDatabase("Failed to create related contact").WithError(err)
		}
		return contact.VCardUID, nil
	default:
		return "", apperrors.ErrInvalidInput(fieldName, "must specify either an existing contact id or a new contact")
	}
}

// applyRelationshipEdgeInput resolves both endpoints (creating thin contacts
// as needed) and writes Type/Directional/Metadata/Sensitivity onto edge,
// inside a transaction so a thin-contact insert and the edge write/update
// commit or roll back together. Used by both CreateRelationshipEdge and
// UpdateRelationshipEdge. edge.UserID/Status/Source/Confidence must already
// be set by the caller before calling this (create: fresh values; update:
// the pre-loaded row's existing values, left untouched).
func applyRelationshipEdgeInput(db *gorm.DB, userID uint, edge *models.RelationshipEdge, input *models.RelationshipEdgeInput) error {
	sensitivity := input.Sensitivity
	if sensitivity == "" {
		sensitivity = models.RelationshipSensitivityNormal
	}

	return db.Transaction(func(tx *gorm.DB) error {
		sourceID, aerr := resolveRelationshipEndpoint(tx, userID, "source_id", input.SourceID, input.SourceThin)
		if aerr != nil {
			return aerr
		}
		targetID, aerr := resolveRelationshipEndpoint(tx, userID, "target_id", input.TargetID, input.TargetThin)
		if aerr != nil {
			return aerr
		}
		if sourceID == targetID {
			return apperrors.ErrInvalidInput("target_id", "source and target must be different contacts")
		}

		edge.SourceID = sourceID
		edge.TargetID = targetID
		edge.Type = input.Type
		edge.Directional = !models.IsSymmetricRelationType(input.Type)
		edge.Metadata = input.Metadata
		edge.Sensitivity = sensitivity

		if edge.ID == "" {
			return tx.Create(edge).Error
		}
		return tx.Save(edge).Error
	})
}

// abortRelationshipEdgeError unwraps a possibly-*apperrors.AppError coming
// out of applyRelationshipEdgeInput's transaction and aborts with the
// specific status (404/400) when present, falling back to a generic 500
// database error otherwise -- unlike a pure cascade-delete, this
// transaction's failures legitimately include 404 (endpoint contact not
// found) and 400 (bad source/target combo) cases that must not be
// flattened into a 500.
func abortRelationshipEdgeError(c *gin.Context, err error, fallback string) {
	if appErr, ok := err.(*apperrors.AppError); ok {
		apperrors.AbortWithError(c, appErr)
		return
	}
	apperrors.AbortWithError(c, apperrors.ErrDatabase(fallback).WithError(err))
}

// CreateRelationshipEdge creates a new RelationshipEdge for the authenticated
// user. Source/Confidence/Status are server-derived, never client-supplied:
// a user-created edge is always Source: user-confirmed, Confidence: 1.0,
// Status: confirmed (see RelationshipEdgeInput's doc comment).
func CreateRelationshipEdge(c *gin.Context) {
	input, verr := middleware.GetValidated[models.RelationshipEdgeInput](c)
	if verr != nil {
		apperrors.AbortWithError(c, verr)
		return
	}

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	db := c.MustGet("db").(*gorm.DB)

	edge := models.RelationshipEdge{
		UserID:     userID,
		Source:     models.RelationshipSourceUserConfirmed,
		Confidence: 1.0,
		Status:     models.RelationshipStatusConfirmed,
	}
	if err := applyRelationshipEdgeInput(db, userID, &edge, input); err != nil {
		abortRelationshipEdgeError(c, err, "Failed to save relationship edge")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"relationship_edge": edge})
}

// GetRelationshipEdge returns one RelationshipEdge.
func GetRelationshipEdge(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var edge models.RelationshipEdge
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&edge).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Relationship edge").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve relationship edge").WithError(err))
		}
		return
	}

	c.JSON(http.StatusOK, edge)
}

// ListRelationshipEdges returns the authenticated user's RelationshipEdges,
// paginated, optionally filtered by ?contact_id=<Contact.VCardUID> (matches
// either SourceID or TargetID -- the "both directions in one call" ask that
// replaces the legacy GetRelationships/GetIncomingRelationships split),
// ?status=confirmed|suggested (the suggestion-review-inbox query -- omit
// contact_id to page through every pending suggestion across all contacts),
// and ?type=<registered token>.
func ListRelationshipEdges(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	pagination := GetPaginationParams(c)
	contactID := c.Query("contact_id")
	status := c.Query("status")
	relType := c.Query("type")

	baseQuery := db.Model(&models.RelationshipEdge{}).Where("user_id = ?", userID)
	if contactID != "" {
		// Parens are load-bearing: an unparenthesized OR chained after the
		// user_id clause would let it bind looser than intended and leak
		// other users' edges. Mirrors the exact safe pattern already used
		// for this same OR (contact_controller.go's DeleteContact,
		// "(source_id = ? OR target_id = ?) AND user_id = ?").
		baseQuery = baseQuery.Where("(source_id = ? OR target_id = ?)", contactID, contactID)
	}
	if status != "" {
		baseQuery = baseQuery.Where("status = ?", status)
	}
	if relType != "" {
		baseQuery = baseQuery.Where("type = ?", relType)
	}

	var total int64
	if err := baseQuery.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to count relationship edges").WithError(err))
		return
	}

	var edges []models.RelationshipEdge
	if err := baseQuery.Session(&gorm.Session{}).
		Order("created_at DESC").
		Limit(pagination.Limit).
		Offset(pagination.Offset).
		Find(&edges).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve relationship edges").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"relationship_edges": edges,
		"total":              total,
		"page":               pagination.Page,
		"limit":              pagination.Limit,
	})
}

// UpdateRelationshipEdge replaces a RelationshipEdge's editable fields
// (SourceID/TargetID and their thin-creation alternatives, Type, Metadata,
// Sensitivity) -- full-replace semantics via the same RelationshipEdgeInput
// as create, matching UpdateLifeEvent's own precedent of treating every
// field in the shared Input DTO as replaceable, including which contact(s)
// the record points at. Status/Source/Confidence are untouched (they are
// not part of RelationshipEdgeInput at all) -- use AcceptRelationshipEdge to
// change Status.
func UpdateRelationshipEdge(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var edge models.RelationshipEdge
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&edge).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Relationship edge").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve relationship edge").WithError(err))
		}
		return
	}

	input, verr := middleware.GetValidated[models.RelationshipEdgeInput](c)
	if verr != nil {
		apperrors.AbortWithError(c, verr)
		return
	}

	if err := applyRelationshipEdgeInput(db, userID, &edge, input); err != nil {
		abortRelationshipEdgeError(c, err, "Failed to update relationship edge")
		return
	}

	c.JSON(http.StatusOK, edge)
}

// DeleteRelationshipEdge deletes a RelationshipEdge. For a Status: suggested
// edge this doubles as "reject" -- there is deliberately no separate reject
// endpoint (see AcceptRelationshipEdge's doc comment for the accept/reject
// asymmetry rationale).
func DeleteRelationshipEdge(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var edge models.RelationshipEdge
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&edge).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Relationship edge").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve relationship edge").WithError(err))
		}
		return
	}

	if err := db.Delete(&edge).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to delete relationship edge").WithError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Relationship edge deleted"})
}

// AcceptRelationshipEdge promotes a Status: suggested edge to confirmed --
// the only consumer WP-83's household-inference engine
// (services/household_service.go) has anywhere in the app today. A
// dedicated endpoint rather than folding this into UpdateRelationshipEdge's
// PUT: PUT is full-replace and requires resending Type/Sensitivity/Metadata/
// SourceID/TargetID, which is both bad ergonomics for a single-field
// transition and a real hazard -- a client trying to "just accept" would
// silently be able to edit the type/sensitivity/endpoints in the same call.
// Source/Confidence are left untouched deliberately: accepting only changes
// authority (Status), not provenance -- the edge should keep showing it
// originated from household inference even once accepted. Rejection has no
// equivalent dedicated endpoint: it's just DELETE (see
// DeleteRelationshipEdge).
func AcceptRelationshipEdge(c *gin.Context) {
	id := c.Param("id")
	db := c.MustGet("db").(*gorm.DB)
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var edge models.RelationshipEdge
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&edge).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apperrors.AbortWithError(c, apperrors.ErrNotFound("Relationship edge").WithDetails("id", id))
		} else {
			apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to retrieve relationship edge").WithError(err))
		}
		return
	}

	if edge.Status != models.RelationshipStatusSuggested {
		apperrors.AbortWithError(c, apperrors.ErrConflict("Relationship edge is not a pending suggestion"))
		return
	}

	edge.Status = models.RelationshipStatusConfirmed
	if err := db.Save(&edge).Error; err != nil {
		apperrors.AbortWithError(c, apperrors.ErrDatabase("Failed to accept relationship edge").WithError(err))
		return
	}

	c.JSON(http.StatusOK, edge)
}
