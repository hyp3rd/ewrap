package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"

	"github.com/hyp3rd/ewrap"
	"github.com/hyp3rd/ewrap/adapters"
)

func main() {
	// Initialize different loggers
	zapLogger, _ := zap.NewProduction()
	logrusLogger := logrus.New()
	zerologLogger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Create adapters
	zapAdapter := adapters.NewZapAdapter(zapLogger)
	logrusAdapter := adapters.NewLogrusAdapter(logrusLogger)
	zerologAdapter := adapters.NewZerologAdapter(zerologLogger)

	// Create context with request ID
	ctx := context.WithValue(context.Background(), "request_id", "123")

	// Create and format an error
	err := ewrap.New("database connection failed",
		ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityCritical),
		ewrap.WithLogger(zapAdapter)).
		WithMetadata("host", "db.example.com").
		WithMetadata("port", 5432)

	// Convert to JSON
	jsonOutput, _ := err.ToJSON(
		ewrap.WithTimestampFormat(time.RFC3339),
		ewrap.WithStackTrace(true),
	)

	//nolint:forbidigo
	fmt.Println("json", jsonOutput)

	// Convert to YAML
	yamlOutput, _ := err.ToYAML(
		ewrap.WithTimestampFormat(time.RFC3339),
		ewrap.WithStackTrace(true),
	)
	//nolint:forbidigo
	fmt.Println("yaml", yamlOutput)

	// Log the error using different loggers
	err = ewrap.Wrap(err, "failed to initialize application",
		ewrap.WithLogger(logrusAdapter))
	err.Log()

	err = ewrap.Wrap(err, "application startup failed",
		ewrap.WithLogger(zerologAdapter))
	err.Log()
}
