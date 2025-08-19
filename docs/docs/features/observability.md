# Observability and Monitoring

ewrap provides comprehensive observability features that allow you to monitor error patterns, track system health, and gain insights into application behavior. These features are designed to integrate seamlessly with modern monitoring and alerting systems.

## Observer Interface

The Observer interface allows you to receive notifications about various error-related events in your application:

```go
type Observer interface {
    OnErrorCreated(err *Error, context ErrorContext)
    OnCircuitBreakerStateChange(name string, from, to CircuitState)
    OnRecoverySuggestionTriggered(suggestion string, context ErrorContext)
}
```

## Setting Up Observability

### Global Observer Registration

Register observers globally to monitor all error activity:

```go
// Create a custom observer
type MetricsObserver struct {
    metricsClient *prometheus.Client
    logger        *slog.Logger
}

func (m *MetricsObserver) OnErrorCreated(err *Error, context ErrorContext) {
    // Track error frequency by type
    m.metricsClient.Counter("errors_total").
        WithLabelValues(string(err.ErrorType()), string(err.Severity())).
        Inc()

    // Log structured error information
    m.logger.Error("error created",
        "type", err.ErrorType(),
        "severity", err.Severity(),
        "message", err.Error(),
        "context", context)
}

func (m *MetricsObserver) OnCircuitBreakerStateChange(name string, from, to CircuitState) {
    // Track circuit breaker state transitions
    m.metricsClient.Counter("circuit_breaker_state_changes").
        WithLabelValues(name, string(from), string(to)).
        Inc()

    // Alert on circuit breaker openings
    if to == CircuitStateOpen {
        m.logger.Warn("circuit breaker opened",
            "name", name,
            "previous_state", from)
    }
}

func (m *MetricsObserver) OnRecoverySuggestionTriggered(suggestion string, context ErrorContext) {
    // Track recovery suggestion effectiveness
    m.metricsClient.Counter("recovery_suggestions_total").
        WithLabelValues(suggestion).
        Inc()
}

// Register the observer globally
ewrap.RegisterGlobalObserver(&MetricsObserver{
    metricsClient: prometheusClient,
    logger:        slogLogger,
})
```

### Component-Specific Observers

Attach observers to specific components:

```go
// Create circuit breaker with observer
cb := ewrap.NewCircuitBreaker("payment-service", 5, time.Minute*2,
    ewrap.WithObserver(&PaymentServiceObserver{
        alertManager: alertMgr,
        metrics:     metrics,
    }))

// Observer for specific circuit breaker
type PaymentServiceObserver struct {
    alertManager *AlertManager
    metrics      *Metrics
}

func (p *PaymentServiceObserver) OnCircuitBreakerStateChange(name string, from, to CircuitState) {
    if to == CircuitStateOpen {
        p.alertManager.SendAlert(&Alert{
            Severity: "critical",
            Summary:  fmt.Sprintf("Payment service circuit breaker opened"),
            Details: map[string]string{
                "service":         name,
                "previous_state":  string(from),
                "current_state":   string(to),
                "timestamp":       time.Now().Format(time.RFC3339),
            },
        })
    }
}
```

## Error Pattern Analysis

### Frequency Tracking

Monitor error patterns to identify systemic issues:

```go
type ErrorAnalyzer struct {
    errorCounts   map[string]int
    errorWindows  map[string][]time.Time
    mutex         sync.RWMutex
}

func (ea *ErrorAnalyzer) OnErrorCreated(err *Error, context ErrorContext) {
    ea.mutex.Lock()
    defer ea.mutex.Unlock()

    errorKey := fmt.Sprintf("%s:%s", err.ErrorType(), err.Severity())

    // Track error counts
    ea.errorCounts[errorKey]++

    // Track error timing for rate analysis
    now := time.Now()
    ea.errorWindows[errorKey] = append(ea.errorWindows[errorKey], now)

    // Clean old entries (keep last hour)
    cutoff := now.Add(-time.Hour)
    filtered := ea.errorWindows[errorKey][:0]
    for _, timestamp := range ea.errorWindows[errorKey] {
        if timestamp.After(cutoff) {
            filtered = append(filtered, timestamp)
        }
    }
    ea.errorWindows[errorKey] = filtered

    // Check for error spikes
    if len(ea.errorWindows[errorKey]) > 100 { // More than 100 errors in last hour
        ea.triggerErrorSpike(errorKey, len(ea.errorWindows[errorKey]))
    }
}

func (ea *ErrorAnalyzer) triggerErrorSpike(errorKey string, count int) {
    log.Printf("ERROR SPIKE DETECTED: %s - %d errors in last hour", errorKey, count)
}
```

## Circuit Breaker Monitoring

### State Transition Tracking

Monitor circuit breaker health and performance:

```go
type CircuitBreakerMonitor struct {
    stateHistory   map[string][]StateTransition
    healthMetrics  map[string]*HealthMetrics
    mutex          sync.RWMutex
}

type StateTransition struct {
    From      CircuitState
    To        CircuitState
    Timestamp time.Time
}

type HealthMetrics struct {
    TotalRequests    int64
    SuccessfulReqs   int64
    FailedRequests   int64
    OpenDuration     time.Duration
    LastStateChange  time.Time
}

func (cbm *CircuitBreakerMonitor) OnCircuitBreakerStateChange(name string, from, to CircuitState) {
    cbm.mutex.Lock()
    defer cbm.mutex.Unlock()

    // Record state transition
    transition := StateTransition{
        From:      from,
        To:        to,
        Timestamp: time.Now(),
    }

    cbm.stateHistory[name] = append(cbm.stateHistory[name], transition)

    // Update health metrics
    if cbm.healthMetrics[name] == nil {
        cbm.healthMetrics[name] = &HealthMetrics{}
    }

    metrics := cbm.healthMetrics[name]
    metrics.LastStateChange = transition.Timestamp

    // Track open duration
    if from == CircuitStateOpen && to == CircuitStateHalfOpen {
        // Find when it opened
        for i := len(cbm.stateHistory[name]) - 2; i >= 0; i-- {
            if cbm.stateHistory[name][i].To == CircuitStateOpen {
                openDuration := transition.Timestamp.Sub(cbm.stateHistory[name][i].Timestamp)
                metrics.OpenDuration += openDuration
                break
            }
        }
    }

    // Generate health report
    cbm.generateHealthReport(name, metrics)
}

func (cbm *CircuitBreakerMonitor) generateHealthReport(name string, metrics *HealthMetrics) {
    if metrics.TotalRequests > 0 {
        successRate := float64(metrics.SuccessfulReqs) / float64(metrics.TotalRequests) * 100

        log.Printf("Circuit Breaker Health Report - %s: Success Rate: %.2f%%, Open Duration: %v",
            name, successRate, metrics.OpenDuration)
    }
}
```

## Recovery Suggestion Tracking

Monitor the effectiveness of recovery suggestions:

```go
type RecoveryTracker struct {
    suggestions     map[string]*SuggestionMetrics
    mutex          sync.RWMutex
}

type SuggestionMetrics struct {
    Count          int
    FirstSeen      time.Time
    LastSeen       time.Time
    Contexts       []ErrorContext
}

func (rt *RecoveryTracker) OnRecoverySuggestionTriggered(suggestion string, context ErrorContext) {
    rt.mutex.Lock()
    defer rt.mutex.Unlock()

    if rt.suggestions[suggestion] == nil {
        rt.suggestions[suggestion] = &SuggestionMetrics{
            FirstSeen: time.Now(),
            Contexts:  make([]ErrorContext, 0),
        }
    }

    metrics := rt.suggestions[suggestion]
    metrics.Count++
    metrics.LastSeen = time.Now()
    metrics.Contexts = append(metrics.Contexts, context)

    // Generate actionable insights
    if metrics.Count > 10 {
        rt.generateRecoveryInsights(suggestion, metrics)
    }
}

func (rt *RecoveryTracker) generateRecoveryInsights(suggestion string, metrics *SuggestionMetrics) {
    frequency := float64(metrics.Count) / time.Since(metrics.FirstSeen).Hours()

    log.Printf("Recovery Suggestion Analysis - '%s': Count: %d, Frequency: %.2f/hour, Duration: %v",
        suggestion, metrics.Count, frequency, time.Since(metrics.FirstSeen))

    // Analyze context patterns
    contextTypes := make(map[string]int)
    for _, ctx := range metrics.Contexts {
        contextTypes[string(ctx.ErrorType)]++
    }

    for contextType, count := range contextTypes {
        percentage := float64(count) / float64(len(metrics.Contexts)) * 100
        log.Printf("  Context Type '%s': %d occurrences (%.1f%%)", contextType, count, percentage)
    }
}
```

## Integration with Monitoring Systems

### Prometheus Integration

```go
import "github.com/prometheus/client_golang/prometheus"

type PrometheusObserver struct {
    errorCounter         *prometheus.CounterVec
    circuitBreakerGauge  *prometheus.GaugeVec
    recoveryCounter      *prometheus.CounterVec
}

func NewPrometheusObserver() *PrometheusObserver {
    return &PrometheusObserver{
        errorCounter: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "ewrap_errors_total",
                Help: "Total number of errors created",
            },
            []string{"type", "severity"},
        ),
        circuitBreakerGauge: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Name: "ewrap_circuit_breaker_state",
                Help: "Current circuit breaker state (0=closed, 1=open, 2=half-open)",
            },
            []string{"name"},
        ),
        recoveryCounter: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "ewrap_recovery_suggestions_total",
                Help: "Total number of recovery suggestions triggered",
            },
            []string{"suggestion_type"},
        ),
    }
}

func (p *PrometheusObserver) OnCircuitBreakerStateChange(name string, from, to CircuitState) {
    var stateValue float64
    switch to {
    case CircuitStateClosed:
        stateValue = 0
    case CircuitStateOpen:
        stateValue = 1
    case CircuitStateHalfOpen:
        stateValue = 2
    }

    p.circuitBreakerGauge.WithLabelValues(name).Set(stateValue)
}
```

## Best Practices

### Performance Considerations

1. **Lightweight Observers**: Keep observer implementations fast to avoid impacting error handling performance
2. **Async Processing**: Use goroutines for expensive operations in observers
3. **Buffered Channels**: Use buffered channels for high-throughput scenarios

### Monitoring Strategy

1. **Error Rate Monitoring**: Track error rates by type and severity
2. **Circuit Breaker Health**: Monitor state transitions and success rates
3. **Recovery Effectiveness**: Analyze which recovery suggestions are most common
4. **Performance Impact**: Monitor the overhead of observability features

### Alert Configuration

1. **Error Spikes**: Alert on sudden increases in error rates
2. **Circuit Breaker Openings**: Immediate alerts when services become unavailable
3. **Recovery Pattern Changes**: Notify when new types of errors appear frequently

The observability features in ewrap provide deep insights into your application's error patterns and system health, enabling proactive monitoring and faster incident response.
