# Circuit Breaker Pattern

The Circuit Breaker pattern is like a safety switch in an electrical system - it prevents cascade failures by "breaking the circuit" when too many errors occur. This pattern is crucial for building resilient systems that can gracefully handle failures in distributed environments.

## Understanding Circuit Breakers

Imagine you're calling a database service. Without a circuit breaker, if the database becomes slow or unresponsive, your application might:

1. Keep trying and failing
2. Accumulate resource-consuming connections
3. Eventually crash or become unresponsive itself

A circuit breaker prevents this by monitoring failures and automatically stopping attempts when a threshold is reached, giving the system time to recover.

## Basic Circuit Breaker Usage

Let's start with a simple example of how to use circuit breakers in ewrap:

```go
// Create a circuit breaker that will:
// - Open after 5 failures
// - Stay open for 1 minute before attempting recovery
cb := ewrap.NewCircuitBreaker("database-operations", 5, time.Minute)

func queryDatabase() error {
    // Check if we can execute the operation
    if !cb.CanExecute() {
        return ewrap.New("circuit breaker is open",
            ewrap.WithErrorType(ewrap.ErrorTypeDatabase),
            ewrap.WithMetadata("breaker_name", "database-operations"))
    }

    err := performDatabaseQuery()
    if err != nil {
        // Record the failure
        cb.RecordFailure()
        return ewrap.Wrap(err, "database query failed")
    }

    // Record the success
    cb.RecordSuccess()
    return nil
}
```

## Circuit Breaker States

A circuit breaker can be in one of three states:

### Closed State (Normal Operation)

```go
if cb.CanExecute() {  // Returns true when circuit is closed
    // Normal operation - requests are allowed through
    err := performOperation()
    if err != nil {
        cb.RecordFailure()
    } else {
        cb.RecordSuccess()
    }
}
```

### Open State (Failure Prevention)

```go
if !cb.CanExecute() {  // Returns false when circuit is open
    // Circuit is open - fail fast without attempting operation
    return ewrap.New("service unavailable",
        ewrap.WithErrorType(ewrap.ErrorTypeInternal),
        ewrap.WithMetadata("circuit_state", "open"))
}
```

### Half-Open State (Recovery Attempt)

```go
// After timeout period, circuit moves to half-open
// Allowing a single request through to test the service
if cb.CanExecute() {
    err := performOperation()
    if err != nil {
        cb.RecordFailure()  // Returns to open state
        return err
    }
    cb.RecordSuccess()  // Returns to closed state
    return nil
}
```

## Advanced Circuit Breaker Patterns

### Monitoring Multiple Services

When your application depends on multiple services, you can use separate circuit breakers for each:

```go
type ServiceManager struct {
    dbBreaker    *ewrap.CircuitBreaker
    cacheBreaker *ewrap.CircuitBreaker
    apiBreaker   *ewrap.CircuitBreaker
}

func NewServiceManager() *ServiceManager {
    return &ServiceManager{
        dbBreaker:    ewrap.NewCircuitBreaker("database", 5, time.Minute),
        cacheBreaker: ewrap.NewCircuitBreaker("cache", 3, time.Second*30),
        apiBreaker:   ewrap.NewCircuitBreaker("external-api", 10, time.Minute*2),
    }
}

func (sm *ServiceManager) GetUserData(userID string) (*UserData, error) {
    // Try cache first
    if sm.cacheBreaker.CanExecute() {
        data, err := tryCache(userID)
        if err == nil {
            sm.cacheBreaker.RecordSuccess()
            return data, nil
        }
        sm.cacheBreaker.RecordFailure()
    }

    // Fall back to database
    if sm.dbBreaker.CanExecute() {
        data, err := queryDatabase(userID)
        if err == nil {
            sm.dbBreaker.RecordSuccess()
            return data, nil
        }
        sm.dbBreaker.RecordFailure()
    }

    return nil, ewrap.New("all data sources unavailable",
        ewrap.WithErrorType(ewrap.ErrorTypeInternal),
        ewrap.WithSeverity(ewrap.SeverityCritical))
}
```

### Circuit Breaker with Fallback Strategies

Implement graceful degradation when services fail:

```go
type CacheService struct {
    primaryBreaker   *ewrap.CircuitBreaker
    secondaryBreaker *ewrap.CircuitBreaker
    localCache       *cache.Cache
}

func (cs *CacheService) GetValue(key string) (interface{}, error) {
    // Try primary cache
    if cs.primaryBreaker.CanExecute() {
        value, err := cs.getPrimaryCache(key)
        if err == nil {
            cs.primaryBreaker.RecordSuccess()
            return value, nil
        }
        cs.primaryBreaker.RecordFailure()
    }

    // Try secondary cache
    if cs.secondaryBreaker.CanExecute() {
        value, err := cs.getSecondaryCache(key)
        if err == nil {
            cs.secondaryBreaker.RecordSuccess()
            return value, nil
        }
        cs.secondaryBreaker.RecordFailure()
    }

    // Fall back to local cache
    if value, found := cs.localCache.Get(key); found {
        return value, nil
    }

    return nil, ewrap.New("all cache layers unavailable",
        ewrap.WithErrorType(ewrap.ErrorTypeInternal))
}
```

## Combining with Error Groups

Circuit Breakers work particularly well with Error Groups for batch operations:

```go
func processBatch(items []Item) error {
    pool := ewrap.NewErrorGroupPool(len(items))
    eg := pool.Get()
    defer eg.Release()

    cb := ewrap.NewCircuitBreaker("batch-processor", 5, time.Minute)

    for _, item := range items {
        if !cb.CanExecute() {
            eg.Add(ewrap.New("circuit breaker open: too many failures"))
            break
        }

        if err := processItem(item); err != nil {
            cb.RecordFailure()
            eg.Add(ewrap.Wrap(err, fmt.Sprintf("failed to process item %d", item.ID)))
        } else {
            cb.RecordSuccess()
        }
    }

    return eg.Error()
}
```

## Best Practices

### 1. Choose Appropriate Thresholds

Consider your service's characteristics when configuring circuit breakers:

```go
// For critical, fast operations
cb := ewrap.NewCircuitBreaker("critical-service", 3, time.Second*30)

// For less critical, slower operations
cb := ewrap.NewCircuitBreaker("background-service", 10, time.Minute*5)
```

### 2. Monitor Circuit Breaker States

Implement monitoring to track circuit breaker behavior:

```go
type MonitoredCircuitBreaker struct {
    *ewrap.CircuitBreaker
    metrics *metrics.Recorder
}

func (mcb *MonitoredCircuitBreaker) RecordFailure() {
    mcb.CircuitBreaker.RecordFailure()
    mcb.metrics.Increment("circuit_breaker.failures")
}

func (mcb *MonitoredCircuitBreaker) RecordSuccess() {
    mcb.CircuitBreaker.RecordSuccess()
    mcb.metrics.Increment("circuit_breaker.successes")
}
```

### 3. Implement Graceful Degradation

Plan for circuit breaker activation:

```go
func getUserProfile(userID string) (*Profile, error) {
    if !profileBreaker.CanExecute() {
        // Return cached or minimal profile when circuit is open
        return getMinimalProfile(userID)
    }

    profile, err := getFullProfile(userID)
    if err != nil {
        profileBreaker.RecordFailure()
        // Fall back to minimal profile
        return getMinimalProfile(userID)
    }

    profileBreaker.RecordSuccess()
    return profile, nil
}
```

### 4. Use Context-Aware Circuit Breakers

Consider request context when making circuit breaker decisions:

```go
func processWithContext(ctx context.Context, data []byte) error {
    if deadline, ok := ctx.Deadline(); ok {
        // Adjust circuit breaker timeout based on context deadline
        timeout := time.Until(deadline)
        cb := ewrap.NewCircuitBreaker("context-aware", 5, timeout/2)

        if !cb.CanExecute() {
            return ewrap.New("circuit breaker open",
                ewrap.WithContext(ctx, ewrap.ErrorTypeTimeout, ewrap.SeverityWarning))
        }
        // Process with context-aware circuit breaker
    }
    // ... rest of processing
}
```
