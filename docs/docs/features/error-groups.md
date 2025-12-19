# Error Groups

Error Groups in ewrap provide a powerful way to collect, manage, and handle multiple errors together. They are particularly useful in concurrent operations, validation scenarios, or any situation where multiple errors might occur and need to be handled cohesively.

## Understanding Error Groups

An Error Group acts as a thread-safe container for multiple errors. Think of it as a collector that can gather errors from various operations while ensuring that all errors are properly tracked and can be processed together. What makes our implementation special is its efficient memory usage through a pooling mechanism.

## Basic Usage

Let's start with the fundamental ways to use Error Groups:

```go
// Create a pool for error groups
pool := ewrap.NewErrorGroupPool(4)  // Initial capacity of 4 errors

// Get an error group from the pool
eg := pool.Get()
defer eg.Release()  // Don't forget to release it back to the pool

// Add errors to the group
eg.Add(ewrap.New("validation failed for email"))
eg.Add(ewrap.New("validation failed for password"))

// Aggregate errors using errors.Join
if err := eg.Join(); err != nil {
    fmt.Printf("Encountered errors: %v\n", err)
}
```

## Error Group Pooling

Our Error Group implementation uses a pool to reuse instances efficiently. This is particularly valuable in high-throughput scenarios where creating and destroying error groups frequently could impact performance.

### How Pooling Works

The pooling mechanism works behind the scenes to manage memory efficiently:

```go
// Create a pool with specific capacity
pool := ewrap.NewErrorGroupPool(4)

func processUserRegistration(user User) error {
    // Get an error group from the pool
    eg := pool.Get()
    defer eg.Release()  // Returns the group to the pool when done

    // Validate different aspects of the user
    if err := validateEmail(user.Email); err != nil {
        eg.Add(err)
    }

    if err := validatePassword(user.Password); err != nil {
        eg.Add(err)
    }

    if err := validateAge(user.Age); err != nil {
        eg.Add(err)
    }

    return eg.Error()
}
```

## Concurrent Operations

Error Groups are particularly useful in concurrent operations. They're designed to be thread-safe and can safely collect errors from multiple goroutines:

```go
func processItems(items []Item) error {
    pool := ewrap.NewErrorGroupPool(len(items))
    eg := pool.Get()
    defer eg.Release()

    var wg sync.WaitGroup

    for _, item := range items {
        wg.Add(1)
        go func(item Item) {
            defer wg.Done()

            if err := processItem(item); err != nil {
                eg.Add(ewrap.Wrap(err, fmt.Sprintf("failed to process item %d", item.ID)))
            }
        }(item)
    }

    wg.Wait()
    return eg.Error()
}
```

## Validation Scenarios

Error Groups excel at collecting validation errors, allowing you to report all validation failures at once rather than stopping at the first error:

```go
func validateUser(user User) error {
    pool := ewrap.NewErrorGroupPool(4)
    eg := pool.Get()
    defer eg.Release()

    // Validate email format
    if !isValidEmail(user.Email) {
        eg.Add(ewrap.New("invalid email format",
            ewrap.WithErrorType(ewrap.ErrorTypeValidation)))
    }

    // Validate password strength
    if !isStrongPassword(user.Password) {
        eg.Add(ewrap.New("password too weak",
            ewrap.WithErrorType(ewrap.ErrorTypeValidation)))
    }

    // Validate age
    if user.Age < 18 {
        eg.Add(ewrap.New("user must be 18 or older",
            ewrap.WithErrorType(ewrap.ErrorTypeValidation)))
    }

    return eg.Error()
}
```

## Advanced Usage Patterns

### Hierarchical Error Collection

You can create hierarchical error structures by nesting error groups:

```go
func validateOrder(order Order) error {
    mainPool := ewrap.NewErrorGroupPool(2)
    mainGroup := mainPool.Get()
    defer mainGroup.Release()

    // Validate customer details
    if err := func() error {
        customerPool := ewrap.NewErrorGroupPool(3)
        customerGroup := customerPool.Get()
        defer customerGroup.Release()

        if err := validateCustomerEmail(order.Customer.Email); err != nil {
            customerGroup.Add(err)
        }
        if err := validateCustomerAddress(order.Customer.Address); err != nil {
            customerGroup.Add(err)
        }

        return customerGroup.Error()
    }(); err != nil {
        mainGroup.Add(ewrap.Wrap(err, "customer validation failed"))
    }

    // Validate order items
    if err := func() error {
        itemsPool := ewrap.NewErrorGroupPool(len(order.Items))
        itemsGroup := itemsPool.Get()
        defer itemsGroup.Release()

        for _, item := range order.Items {
            if err := validateOrderItem(item); err != nil {
                itemsGroup.Add(err)
            }
        }

        return itemsGroup.Error()
    }(); err != nil {
        mainGroup.Add(ewrap.Wrap(err, "order items validation failed"))
    }

    return mainGroup.Error()
}
```

### Error Group with Circuit Breaker

Combine Error Groups with Circuit Breakers for robust error handling:

```go
func processOrderBatch(orders []Order) error {
    pool := ewrap.NewErrorGroupPool(len(orders))
    eg := pool.Get()
    defer eg.Release()

    cb := ewrap.NewCircuitBreaker("order-processing", 5, time.Minute)

    for _, order := range orders {
        if !cb.CanExecute() {
            eg.Add(ewrap.New("circuit breaker open: too many failures",
                ewrap.WithErrorType(ewrap.ErrorTypeInternal)))
            break
        }

        if err := processOrder(order); err != nil {
            cb.RecordFailure()
            eg.Add(ewrap.Wrap(err, fmt.Sprintf("failed to process order %s", order.ID)))
        } else {
            cb.RecordSuccess()
        }
    }

    return eg.Error()
}
```

## Performance Considerations

The pooled Error Group implementation is designed for high performance, but there are some best practices to follow:

1. **Choose Appropriate Pool Capacity**:

    ```go
    // For known size operations
    pool := ewrap.NewErrorGroupPool(len(items))

    // For variable size operations, estimate typical case
    pool := ewrap.NewErrorGroupPool(4)  // If you typically expect 1-4 errors
    ```

1. **Release Groups Properly**:

    ```go
    func processWithErrors() error {
        eg := pool.Get()
        // Always release with defer to prevent leaks
        defer eg.Release()

        // Use the error group...
        return eg.Error()
    }
    ```

1. **Reuse Pools**:

    ```go
    // Good: Create pool once and reuse
    var validationPool = ewrap.NewErrorGroupPool(4)

    func validateData(data Data) error {
        eg := validationPool.Get()
        defer eg.Release()
        // Use the error group...
    }

    // Less efficient: Creating new pools frequently
    func validateData(data Data) error {
        pool := ewrap.NewErrorGroupPool(4)  // Don't do this
        eg := pool.Get()
        // ...
    }

    ```

## Best Practices

1. **Always Release Error Groups**:
    Use defer to ensure Error Groups are always released back to their pool:

    ```go
    eg := pool.Get()
    defer eg.Release()
    ```

1. **Size Pools Appropriately**:
    Choose pool sizes based on your expected error cases:

    ```go
    // For validation where you know the maximum possible errors
    pool := ewrap.NewErrorGroupPool(len(validationRules))
    ```

1. **Handle Nested Operations**:
    When dealing with nested operations, manage Error Groups carefully:

    ```go
    func processComplex() error {
        outerPool := ewrap.NewErrorGroupPool(2)
        outerGroup := outerPool.Get()
        defer outerGroup.Release()

        for _, item := range items {
            innerPool := ewrap.NewErrorGroupPool(4)
            innerGroup := innerPool.Get()

            // Process with inner group...

            if err := innerGroup.Error(); err != nil {
                outerGroup.Add(err)
            }
            innerGroup.Release()
        }

        return outerGroup.Error()
    }
    ```
