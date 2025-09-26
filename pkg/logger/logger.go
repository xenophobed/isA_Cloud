package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Logger wraps slog.Logger with additional functionality
type Logger struct {
	*slog.Logger
}

// New creates a new logger with the specified level and debug mode
func New(level string, debug bool) *Logger {
	var logLevel slog.Level
	
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	var handler slog.Handler
	if debug {
		// Text handler for development
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		// JSON handler for production
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
	}
}

// WithService returns a logger with service name context
func (l *Logger) WithService(service string) *Logger {
	return &Logger{
		Logger: l.Logger.With("service", service),
	}
}

// WithRequest returns a logger with request context
func (l *Logger) WithRequest(requestID, userID string) *Logger {
	return &Logger{
		Logger: l.Logger.With(
			"request_id", requestID,
			"user_id", userID,
		),
	}
}

// WithError returns a logger with error context
func (l *Logger) WithError(err error) *Logger {
	return &Logger{
		Logger: l.Logger.With("error", err),
	}
}