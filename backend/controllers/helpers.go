package controllers

import (
	apperrors "meerkat/errors"
	"meerkat/config"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	defaultPage  = 1
	defaultLimit = 25
	maxLimit     = 100
)

// likeEscape is the ESCAPE clause that must follow every LIKE matched against likeContains.
const likeEscape = ` ESCAPE '\'`

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// likeContains builds a "contains" LIKE pattern from a user search term, escaping
// the LIKE wildcards so that "50%" or "a_b" match literally.
func likeContains(term string) string {
	return "%" + likeEscaper.Replace(term) + "%"
}

// PaginationParams represents sanitized pagination query values.
type PaginationParams struct {
	Page   int
	Limit  int
	Offset int
}

func currentUserID(c *gin.Context) (uint, bool) {
	value, exists := c.Get("userID")
	if !exists {
		apperrors.AbortWithError(c, apperrors.ErrUnauthorized("Authentication required"))
		return 0, false
	}

	userID, ok := value.(uint)
	if !ok {
		apperrors.AbortWithError(c, apperrors.ErrUnauthorized("Authentication required"))
		return 0, false
	}

	return userID, true
}

func currentConfig(c *gin.Context) config.Config {
	if val, exists := c.Get("cfg"); exists {
		if cfg, ok := val.(config.Config); ok {
			return cfg
		}
	}
	return config.Config{}
}

// GetPaginationParams extracts pagination query params using shared defaults and bounds.
func GetPaginationParams(c *gin.Context) PaginationParams {
	page := parsePositiveOrDefault(c.DefaultQuery("page", "1"), defaultPage)
	limit := parsePositiveOrDefault(c.DefaultQuery("limit", "25"), defaultLimit)
	if limit > maxLimit {
		limit = maxLimit
	}

	return PaginationParams{
		Page:   page,
		Limit:  limit,
		Offset: (page - 1) * limit,
	}
}

func parsePositiveOrDefault(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	return value
}
