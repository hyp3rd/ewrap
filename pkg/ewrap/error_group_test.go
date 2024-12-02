package ewrap

import (
	"fmt"
	"sync"
	"testing"
)

func TestErrorGroupPool(t *testing.T) {
	// Test pool with different capacities
	tests := []struct {
		name            string
		initialCapacity int
		numErrors       int
	}{
		{"SmallCapacity", 2, 5},
		{"ExactCapacity", 4, 4},
		{"LargeCapacity", 8, 3},
		{"InvalidCapacity", -1, 4}, // Should use default capacity
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewErrorGroupPool(tt.initialCapacity)

			// Get multiple groups from the pool
			groups := make([]*ErrorGroup, 3)
			for i := range groups {
				groups[i] = pool.Get()

				// Add errors
				for j := 0; j < tt.numErrors; j++ {
					groups[i].Add(fmt.Errorf("error %d", j))
				}
			}

			// Verify each group works correctly
			for _, eg := range groups {
				if got := len(eg.errors); got != tt.numErrors {
					t.Errorf("Expected %d errors, got %d", tt.numErrors, got)
				}
				eg.Release()
			}
		})
	}
}

func TestConcurrentPoolUsage(t *testing.T) {
	pool := NewErrorGroupPool(4)
	var wg sync.WaitGroup
	numGoroutines := 100

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			eg := pool.Get()
			defer eg.Release()

			eg.Add(fmt.Errorf("error from goroutine %d", id))

			if !eg.HasErrors() {
				t.Errorf("Expected errors in group %d", id)
			}
		}(i)
	}

	wg.Wait()
}

func BenchmarkErrorGroupPool(b *testing.B) {
	sampleErrors := make([]error, 5)
	for i := range sampleErrors {
		sampleErrors[i] = fmt.Errorf("error %d", i)
	}

	b.Run("WithPool", func(b *testing.B) {
		pool := NewErrorGroupPool(4)
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				eg := pool.Get()
				for _, err := range sampleErrors {
					eg.Add(err)
				}
				_ = eg.Error()
				eg.Release()
			}
		})
	})

	b.Run("WithoutPool", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			eg := NewErrorGroup()
			for _, err := range sampleErrors {
				eg.Add(err)
			}
			_ = eg.Error()
		}
	})
}
