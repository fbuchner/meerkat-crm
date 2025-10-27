package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HealthResponse represents the health check response structure
type HealthResponse struct {
	Status    string         `json:"status"`
	Timestamp string         `json:"timestamp"`
	Database  DatabaseHealth `json:"database"`
	Version   string         `json:"version"`
}

// DatabaseHealth represents the database health status
type DatabaseHealth struct {
	Status       string  `json:"status"`
	ResponseTime float64 `json:"response_time_ms"`
}

// HealthCheck handles the health check endpoint
// GET /health
func HealthCheck(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	// Check database connectivity
	dbHealth := checkDatabaseHealth(db)

	// Determine overall status
	status := "healthy"
	httpStatus := http.StatusOK

	if dbHealth.Status == "unhealthy" {
		status = "unhealthy"
		httpStatus = http.StatusServiceUnavailable
	}

	response := HealthResponse{
		Status:    status,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Database:  dbHealth,
		Version:   "0.1.0",
	}

	c.JSON(httpStatus, response)
}

// checkDatabaseHealth checks if the database is accessible and responsive
func checkDatabaseHealth(db *gorm.DB) DatabaseHealth {
	start := time.Now()

	sqlDB, err := db.DB()
	if err != nil {
		return DatabaseHealth{
			Status:       "unhealthy",
			ResponseTime: 0,
		}
	}

	// Ping the database
	err = sqlDB.Ping()
	duration := time.Since(start).Milliseconds()

	if err != nil {
		return DatabaseHealth{
			Status:       "unhealthy",
			ResponseTime: float64(duration),
		}
	}

	return DatabaseHealth{
		Status:       "healthy",
		ResponseTime: float64(duration),
	}
}
