package logger

import (
	"io"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	// Logger is the global logger instance
	Logger zerolog.Logger
)

// Config holds logger configuration
type Config struct {
	Level      string // debug, info, warn, error
	Pretty     bool   // Enable pretty console output
	TimeFormat string // Time format for logs
}

// InitLogger initializes the global logger with the given configuration
func InitLogger(cfg Config) {
	// Set log level
	level := parseLevel(cfg.Level)
	zerolog.SetGlobalLevel(level)

	// Configure output
	var output io.Writer = os.Stdout

	if cfg.Pretty {
		// Pretty console output for development
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
			NoColor:    false,
		}
	}

	// Set time format
	if cfg.TimeFormat != "" {
		zerolog.TimeFieldFormat = cfg.TimeFormat
	} else {
		zerolog.TimeFieldFormat = time.RFC3339
	}

	// Create logger
	Logger = zerolog.New(output).With().
		Timestamp().
		Caller().
		Logger()

	// Set global logger
	log.Logger = Logger

	Logger.Info().
		Str("level", cfg.Level).
		Bool("pretty", cfg.Pretty).
		Msg("Logger initialized")
}

// parseLevel converts string level to zerolog.Level
func parseLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}

// FromContext extracts logger from gin context with request-specific fields
func FromContext(c *gin.Context) *zerolog.Logger {
	l := Logger.With().Logger()

	// Add request ID if available
	if requestID, exists := c.Get("request_id"); exists {
		if id, ok := requestID.(string); ok {
			l = l.With().Str("request_id", id).Logger()
		}
	}

	// Add user ID if available
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(uint); ok {
			l = l.With().Uint("user_id", id).Logger()
		}
	}

	// Add request info
	l = l.With().
		Str("method", c.Request.Method).
		Str("path", c.Request.URL.Path).
		Str("ip", c.ClientIP()).
		Logger()

	return &l
}

// Info returns a logger at info level
func Info() *zerolog.Event {
	return Logger.Info()
}

// Debug returns a logger at debug level
func Debug() *zerolog.Event {
	return Logger.Debug()
}

// Warn returns a logger at warn level
func Warn() *zerolog.Event {
	return Logger.Warn()
}

// Error returns a logger at error level
func Error() *zerolog.Event {
	return Logger.Error()
}

// Fatal returns a logger at fatal level
func Fatal() *zerolog.Event {
	return Logger.Fatal()
}

// Panic returns a logger at panic level
func Panic() *zerolog.Event {
	return Logger.Panic()
}
