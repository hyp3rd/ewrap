# Error Metadata

Metadata is additional information attached to errors that provides crucial context for debugging and error handling. Think of metadata as tags or labels that give you deeper insight into what was happening when an error occurred. In ewrap, metadata is implemented as a flexible key-value store that travels with the error through your application.

## Understanding Error Metadata

When an error occurs, the error message alone often doesn't tell the complete story. For example, if a database query fails, you might want to know:

- What query was being executed?
- How long did it take before failing?
- What parameters were used?
- How many retries were attempted?

Metadata allows you to capture all this contextual information in a structured way.

## Basic Metadata Usage

Let's start with the fundamentals of adding and retrieving metadata:

```go
func processUserOrder(userID string, orderID string) error {
    err := processOrder(orderID)
    if err != nil {
        return ewrap.Wrap(err, "failed to process order").
            WithMetadata("user_id", userID).
            WithMetadata("order_id", orderID).
            WithMetadata("timestamp", time.Now()).
            WithMetadata("attempt", 1)
    }
    return nil
}
```

Retrieving metadata is just as straightforward:

```go
func handleError(err error) {
    if wrappedErr, ok := err.(*ewrap.Error); ok {
        // Get specific metadata values
        if userID, exists := wrappedErr.GetMetadata("user_id"); exists {
            fmt.Printf("Error occurred for user: %v\n", userID)
        }

        // Log all metadata for debugging
        if timestamp, exists := wrappedErr.GetMetadata("timestamp"); exists {
            fmt.Printf("Error occurred at: %v\n", timestamp)
        }
    }
}
```

## Structured Metadata Patterns

While metadata values can be of any type, it's often helpful to use structured data for complex information:

```go
type QueryMetadata struct {
    SQL        string
    Parameters []interface{}
    Duration   time.Duration
    Table      string
}

func executeQuery(query string, params ...interface{}) error {
    start := time.Now()

    result, err := db.Exec(query, params...)
    if err != nil {
        queryMeta := QueryMetadata{
            SQL:        query,
            Parameters: params,
            Duration:   time.Since(start),
            Table:      extractTableName(query),
        }

        return ewrap.Wrap(err, "database query failed").
            WithMetadata("query_info", queryMeta).
            WithMetadata("query_attempt", 1)
    }

    return nil
}
```

## Dynamic Metadata Collection

Sometimes you need to build metadata progressively as an operation proceeds:

```go
func processComplexOperation(ctx context.Context, data []byte) error {
    // Create error with initial metadata
    err := ewrap.New("starting complex operation",
        ewrap.WithContext(ctx, ewrap.ErrorTypeInternal, ewrap.SeverityInfo)).
        WithMetadata("start_time", time.Now()).
        WithMetadata("data_size", len(data))

    // Process stages and collect metadata
    stages := []string{"validation", "transformation", "storage"}
    metrics := make(map[string]time.Duration)

    for _, stage := range stages {
        stageStart := time.Now()

        if err := processStage(stage, data); err != nil {
            // Add stage-specific metadata to error
            return ewrap.Wrap(err, fmt.Sprintf("%s stage failed", stage)).
                WithMetadata("failed_stage", stage).
                WithMetadata("stage_metrics", metrics).
                WithMetadata("stage_duration", time.Since(stageStart))
        }

        metrics[stage] = time.Since(stageStart)
    }

    return nil
}
```

## Metadata for Debugging and Monitoring

Metadata is particularly valuable for debugging and monitoring. Here's a pattern that combines metadata with logging:

```go
type OperationTracker struct {
    StartTime   time.Time
    Steps      []string
    Metrics    map[string]interface{}
    Attributes map[string]string
}

func NewOperationTracker() *OperationTracker {
    return &OperationTracker{
        StartTime:   time.Now(),
        Steps:      make([]string, 0),
        Metrics:    make(map[string]interface{}),
        Attributes: make(map[string]string),
    }
}

func (ot *OperationTracker) AddStep(step string) {
    ot.Steps = append(ot.Steps, step)
}

func (ot *OperationTracker) AddMetric(key string, value interface{}) {
    ot.Metrics[key] = value
}

func processWithTracking(ctx context.Context, data []byte) error {
    tracker := NewOperationTracker()

    // Track operation progress
    err := func() error {
        tracker.AddStep("initialization")

        if err := validate(data); err != nil {
            return ewrap.Wrap(err, "validation failed").
                WithMetadata("tracker", tracker)
        }
        tracker.AddStep("validation")

        if err := transform(data); err != nil {
            return ewrap.Wrap(err, "transformation failed").
                WithMetadata("tracker", tracker)
        }
        tracker.AddStep("transformation")

        tracker.AddMetric("processing_time", time.Since(tracker.StartTime))
        return nil
    }()

    if err != nil {
        return ewrap.Wrap(err, "operation failed").
            WithMetadata("final_state", tracker)
    }

    return nil
}
```

## Metadata Best Practices

### 1. Keep Metadata Serializable

Ensure your metadata can be properly serialized when needed:

```go
// Good - uses simple types
err = ewrap.New("processing failed").
    WithMetadata("count", 42).
    WithMetadata("status", "incomplete")

// Better - uses structured data that can be serialized
type ProcessMetadata struct {
    Count    int    `json:"count"`
    Status   string `json:"status"`
    Duration string `json:"duration"`
}

meta := ProcessMetadata{
    Count:    42,
    Status:   "incomplete",
    Duration: time.Since(start).String(),
}

err = ewrap.New("processing failed").
    WithMetadata("process_info", meta)
```

### 2. Use Consistent Keys

Maintain consistent metadata keys across your application:

```go
// Define common metadata keys as constants
const (
    MetaKeyUserID      = "user_id"
    MetaKeyRequestID   = "request_id"
    MetaKeyDuration    = "duration"
    MetaKeyRetryCount  = "retry_count"
)

func processRequest(ctx context.Context, userID string) error {
    requestID := ctx.Value("request_id").(string)
    start := time.Now()

    err := performOperation()
    if err != nil {
        return ewrap.Wrap(err, "operation failed").
            WithMetadata(MetaKeyUserID, userID).
            WithMetadata(MetaKeyRequestID, requestID).
            WithMetadata(MetaKeyDuration, time.Since(start))
    }
    return nil
}
```

### 3. Structure Complex Data

For complex metadata, use structured types:

```go
type HTTPRequestMetadata struct {
    Method      string
    URL         string
    StatusCode  int
    Duration    time.Duration
    Headers     map[string][]string
}

func makeAPICall(ctx context.Context, req *http.Request) error {
    start := time.Now()
    resp, err := http.DefaultClient.Do(req)

    requestMeta := HTTPRequestMetadata{
        Method:   req.Method,
        URL:      req.URL.String(),
        Duration: time.Since(start),
    }

    if err != nil {
        return ewrap.Wrap(err, "API call failed").
            WithMetadata("request_details", requestMeta)
    }

    requestMeta.StatusCode = resp.StatusCode
    requestMeta.Headers = resp.Header

    if resp.StatusCode >= 400 {
        return ewrap.New("API returned error status").
            WithMetadata("request_details", requestMeta)
    }

    return nil
}
```
