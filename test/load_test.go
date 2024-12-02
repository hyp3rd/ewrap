package test

// import (
// 	"context"
// 	"math/rand"
// 	"sync"
// 	"testing"
// 	"time"

// 	"github.com/hyp3rd/ewrap/internal/logger"
// )

// // LoadTest simulates real-world error handling scenarios under load
// type LoadTest struct {
// 	duration       time.Duration
// 	concurrency    int
// 	errorRate      float64
// 	circuitBreaker *CircuitBreaker
// 	errorGroup     *ErrorGroup
// 	logger         logger.Logger
// 	stats          *LoadTestStats
// 	wg             sync.WaitGroup
// }

// // LoadTestStats tracks statistics during load testing
// type LoadTestStats struct {
// 	totalOperations int64
// 	successfulOps   int64
// 	failedOps       int64
// 	avgResponseTime time.Duration
// 	maxResponseTime time.Duration
// 	errorsGenerated int64
// 	circuitBreaks   int64
// 	mu              sync.Mutex
// }

// func NewLoadTest(duration time.Duration, concurrency int, errorRate float64) *LoadTest {
// 	return &LoadTest{
// 		duration:       duration,
// 		concurrency:    concurrency,
// 		errorRate:      errorRate,
// 		circuitBreaker: NewCircuitBreaker("loadtest", 100, time.Second),
// 		errorGroup:     NewErrorGroup(),
// 		logger:         &mockLogger{},
// 		stats:          &LoadTestStats{},
// 	}
// }

// func (lt *LoadTest) Run(t *testing.T) {
// 	start := time.Now()
// 	done := make(chan struct{})

// 	// Start monitoring goroutine
// 	go lt.monitor(t, start, done)

// 	// Start worker goroutines
// 	for i := 0; i < lt.concurrency; i++ {
// 		lt.wg.Add(1)
// 		go lt.worker(i)
// 	}

// 	// Wait for duration
// 	time.Sleep(lt.duration)
// 	close(done)
// 	lt.wg.Wait()

// 	// Report results
// 	lt.reportResults(t)
// }

// func (lt *LoadTest) worker(id int) {
// 	defer lt.wg.Done()

// 	ctx := context.Background()
// 	rand.Seed(time.Now().UnixNano())

// 	for {
// 		start := time.Now()

// 		if lt.circuitBreaker.CanExecute() {
// 			if rand.Float64() < lt.errorRate {
// 				// Simulate error scenario
// 				err := New("simulated error",
// 					WithContext(ctx, ErrorTypeDatabase, SeverityCritical),
// 					WithLogger(lt.logger))

// 				lt.errorGroup.Add(err)
// 				lt.circuitBreaker.RecordFailure()

// 				lt.updateStats(false, time.Since(start))
// 			} else {
// 				// Simulate success scenario
// 				lt.circuitBreaker.RecordSuccess()
// 				lt.updateStats(true, time.Since(start))
// 			}
// 		} else {
// 			lt.stats.mu.Lock()
// 			lt.stats.circuitBreaks++
// 			lt.stats.mu.Unlock()
// 		}

// 		// Simulate variable processing time
// 		time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
// 	}
// }

// func (lt *LoadTest) monitor(t *testing.T, start time.Time, done chan struct{}) {
// 	ticker := time.NewTicker(time.Second)
// 	defer ticker.Stop()

// 	for {
// 		select {
// 		case <-ticker.C:
// 			lt.stats.mu.Lock()
// 			t.Logf("Operations: %d, Success: %d, Failed: %d, Circuit Breaks: %d",
// 				lt.stats.totalOperations,
// 				lt.stats.successfulOps,
// 				lt.stats.failedOps,
// 				lt.stats.circuitBreaks)
// 			lt.stats.mu.Unlock()
// 		case <-done:
// 			return
// 		}
// 	}
// }

// func (lt *LoadTest) updateStats(success bool, duration time.Duration) {
// 	lt.stats.mu.Lock()
// 	defer lt.stats.mu.Unlock()

// 	lt.stats.totalOperations++
// 	if success {
// 		lt.stats.successfulOps++
// 	} else {
// 		lt.stats.failedOps++
// 		lt.stats.errorsGenerated++
// 	}

// 	if duration > lt.stats.maxResponseTime {
// 		lt.stats.maxResponseTime = duration
// 	}

// 	// Update average response time
// 	lt.stats.avgResponseTime = time.Duration(
// 		(int64(lt.stats.avgResponseTime)*lt.stats.totalOperations +
// 			int64(duration)) / (lt.stats.totalOperations + 1))
// }

// func (lt *LoadTest) reportResults(t *testing.T) {
// 	lt.stats.mu.Lock()
// 	defer lt.stats.mu.Unlock()

// 	t.Logf("\nLoad Test Results:")
// 	t.Logf("================")
// 	t.Logf("Duration: %v", lt.duration)
// 	t.Logf("Concurrency: %d", lt.concurrency)
// 	t.Logf("Total Operations: %d", lt.stats.totalOperations)
// 	t.Logf("Successful Operations: %d", lt.stats.successfulOps)
// 	t.Logf("Failed Operations: %d", lt.stats.failedOps)
// 	t.Logf("Circuit Breaks: %d", lt.stats.circuitBreaks)
// 	t.Logf("Average Response Time: %v", lt.stats.avgResponseTime)
// 	t.Logf("Max Response Time: %v", lt.stats.maxResponseTime)
// 	t.Logf("Error Rate: %.2f%%", float64(lt.stats.failedOps)/float64(lt.stats.totalOperations)*100)
// }

// func TestUnderLoad(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("Skipping load test in short mode")
// 	}

// 	scenarios := []struct {
// 		name        string
// 		duration    time.Duration
// 		concurrency int
// 		errorRate   float64
// 	}{
// 		{"LowConcurrency", 5 * time.Second, 10, 0.1},
// 		{"HighConcurrency", 5 * time.Second, 100, 0.1},
// 		{"HighErrorRate", 5 * time.Second, 50, 0.5},
// 		{"StressTest", 10 * time.Second, 200, 0.3},
// 	}

// 	for _, scenario := range scenarios {
// 		t.Run(scenario.name, func(t *testing.T) {
// 			loadTest := NewLoadTest(
// 				scenario.duration,
// 				scenario.concurrency,
// 				scenario.errorRate,
// 			)
// 			loadTest.Run(t)
// 		})
// 	}
// }
