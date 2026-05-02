package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/hyp3rd/ewrap"
	ewrapslog "github.com/hyp3rd/ewrap/slog"
)

func main() {
	logger := ewrapslog.New(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	// Create context with request ID
	ctx := context.WithValue(context.Background(), "request_id", "123")

	// Create and format an error
	err := ewrap.New("database connection failed",
		ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityCritical),
		ewrap.WithLogger(logger)).
		WithMetadata("host", "db.example.com").
		WithMetadata("port", 5432)

	// Convert to JSON
	jsonOutput, _ := err.ToJSON(
		ewrap.WithTimestampFormat(time.RFC3339),
		ewrap.WithStackTrace(true),
	)

	fmt.Fprintln(os.Stdout, "json", jsonOutput)

	// Convert to YAML
	yamlOutput, _ := err.ToYAML(
		ewrap.WithTimestampFormat(time.RFC3339),
		ewrap.WithStackTrace(true),
	)

	fmt.Fprintln(os.Stdout, "yaml", yamlOutput)

	// Wrap and log
	err = ewrap.Wrap(err, "failed to initialize application",
		ewrap.WithLogger(logger))
	err.Log()
}
