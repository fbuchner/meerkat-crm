package controllers

import (
	"fmt"
	"meerkat/config"
	apperrors "meerkat/errors"
	"meerkat/logger"
	"meerkat/middleware"
	"meerkat/models"
	"meerkat/services"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// importSessions owns all in-progress import wizard state. Controllers only validate
// HTTP input and delegate to the manager.
var importSessions = services.NewImportSessionManager()

// UploadCSVForImport handles CSV file upload and returns headers with suggested mappings
func UploadCSVForImport(c *gin.Context) {
	log := logger.FromContext(c)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	go importSessions.CleanupExpired()

	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		log.Warn().Err(err).Msg("No file uploaded")
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("file", "No file uploaded"))
		return
	}

	if file.Size > services.MaxCSVSize {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("file", fmt.Sprintf("File too large. Maximum size is %d MB", services.MaxCSVSize/(1024*1024))))
		return
	}

	if !strings.HasSuffix(strings.ToLower(file.Filename), ".csv") {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("file", "File must be a CSV file"))
		return
	}

	f, err := file.Open()
	if err != nil {
		log.Error().Err(err).Msg("Failed to open uploaded file")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to process file"))
		return
	}
	defer f.Close()

	// Parse CSV using service
	headers, dataRows, err := services.ParseCSV(f)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to parse CSV")
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("file", err.Error()))
		return
	}

	sessionID := importSessions.CreateCSVSession(userID, headers, dataRows)

	// Get sample data for preview
	sampleData := make([][]string, 0, services.SampleRows)
	for i := 0; i < len(dataRows) && i < services.SampleRows; i++ {
		sampleData = append(sampleData, dataRows[i])
	}

	// Suggest column mappings using service
	suggestedMappings := services.SuggestColumnMappings(headers)

	log.Info().
		Str("session_id", sessionID).
		Int("headers", len(headers)).
		Int("rows", len(dataRows)).
		Msg("CSV uploaded successfully")

	c.JSON(http.StatusOK, models.ImportUploadResponse{
		SessionID:         sessionID,
		Headers:           headers,
		SuggestedMappings: suggestedMappings,
		RowCount:          len(dataRows),
		SampleData:        sampleData,
	})
}

// UploadVCFForImport handles VCF file upload and returns preview directly (no mapping needed)
func UploadVCFForImport(c *gin.Context, cfg *config.Config) {
	db := c.MustGet("db").(*gorm.DB)
	log := logger.FromContext(c)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	go importSessions.CleanupExpired()

	file, err := c.FormFile("file")
	if err != nil {
		log.Warn().Err(err).Msg("No file uploaded")
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("file", "No file uploaded"))
		return
	}

	if file.Size > services.MaxVCFSize {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("file", fmt.Sprintf("File too large. Maximum size is %d MB", services.MaxVCFSize/(1024*1024))))
		return
	}

	if !strings.HasSuffix(strings.ToLower(file.Filename), ".vcf") {
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("file", "File must be a VCF file"))
		return
	}

	f, err := file.Open()
	if err != nil {
		log.Error().Err(err).Msg("Failed to open uploaded file")
		apperrors.AbortWithError(c, apperrors.ErrInternal("Failed to process file"))
		return
	}
	defer f.Close()

	// Parse VCF using service
	vcfContacts, previews, stats, err := services.ParseVCF(f, db, userID)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to parse VCF")
		apperrors.AbortWithError(c, apperrors.ErrInvalidInput("file", err.Error()))
		return
	}

	sessionID := importSessions.CreateVCFSession(userID, vcfContacts, previews)

	log.Info().
		Str("session_id", sessionID).
		Int("contacts", len(vcfContacts)).
		Int("valid", stats.ValidCount).
		Int("duplicates", stats.DuplicateCount).
		Int("errors", stats.ErrorCount).
		Msg("VCF uploaded and parsed successfully")

	c.JSON(http.StatusOK, models.ImportPreviewResponse{
		SessionID:      sessionID,
		Rows:           previews,
		TotalRows:      len(previews),
		ValidRows:      stats.ValidCount,
		DuplicateCount: stats.DuplicateCount,
		ErrorCount:     stats.ErrorCount,
	})
}

// PreviewImport applies mappings and returns preview with duplicate detection
func PreviewImport(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	log := logger.FromContext(c)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	request, err := middleware.GetValidated[models.ImportPreviewRequest](c)
	if err != nil {
		apperrors.AbortWithError(c, err)
		return
	}

	response, appErr := importSessions.PreviewCSV(db, userID, *request)
	if appErr != nil {
		apperrors.AbortWithError(c, appErr)
		return
	}

	log.Info().
		Str("session_id", response.SessionID).
		Int("total", response.TotalRows).
		Int("valid", response.ValidRows).
		Int("duplicates", response.DuplicateCount).
		Int("errors", response.ErrorCount).
		Msg("Import preview generated")

	c.JSON(http.StatusOK, response)
}

// ConfirmImport executes the import with user-specified actions per row
func ConfirmImport(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	log := logger.FromContext(c)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	request, validationErr := middleware.GetValidated[models.ImportConfirmRequest](c)
	if validationErr != nil {
		apperrors.AbortWithError(c, validationErr)
		return
	}

	result, appErr := importSessions.Confirm(db, userID, *request, log)
	if appErr != nil {
		apperrors.AbortWithError(c, appErr)
		return
	}

	c.JSON(http.StatusOK, result)
}

// ConfirmVCFImport executes VCF import with photo processing
func ConfirmVCFImport(c *gin.Context, cfg *config.Config) {
	db := c.MustGet("db").(*gorm.DB)
	log := logger.FromContext(c)

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	request, validationErr := middleware.GetValidated[models.ImportConfirmRequest](c)
	if validationErr != nil {
		apperrors.AbortWithError(c, validationErr)
		return
	}

	result, appErr := importSessions.ConfirmVCF(db, userID, *request, cfg, log)
	if appErr != nil {
		apperrors.AbortWithError(c, appErr)
		return
	}

	c.JSON(http.StatusOK, result)
}
