package ewrap

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

const (
	concurrentPoolGoroutines = 100
	smallCapacity            = 2
	smallErrorCount          = 5
	exactCapacity            = 4
	largeCapacity            = 8
	largeErrorCount          = 3
)

func TestErrorGroupPool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		initialCapacity int
		numErrors       int
	}{
		{"SmallCapacity", smallCapacity, smallErrorCount},
		{"ExactCapacity", exactCapacity, exactCapacity},
		{"LargeCapacity", largeCapacity, largeErrorCount},
		{"InvalidCapacity", -1, exactCapacity},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runPoolCase(t, tt.initialCapacity, tt.numErrors)
		})
	}
}

func runPoolCase(t *testing.T, initialCapacity, numErrors int) {
	t.Helper()

	pool := NewErrorGroupPool(initialCapacity)

	const groupCount = 3

	groups := make([]*ErrorGroup, groupCount)
	for i := range groups {
		groups[i] = pool.Get()

		for j := range numErrors {
			groups[i].Add(fmt.Errorf("%w %d", errIndexed, j))
		}
	}

	for _, eg := range groups {
		if got := len(eg.errors); got != numErrors {
			t.Errorf("Expected %d errors, got %d", numErrors, got)
		}

		eg.Release()
	}
}

func TestConcurrentPoolUsage(t *testing.T) {
	t.Parallel()

	pool := NewErrorGroupPool(exactCapacity)

	var wg sync.WaitGroup

	wg.Add(concurrentPoolGoroutines)

	for i := range concurrentPoolGoroutines {
		go func(id int) {
			defer wg.Done()

			eg := pool.Get()
			defer eg.Release()

			eg.Add(fmt.Errorf("%w %d", errFromGoroutine, id))

			if !eg.HasErrors() {
				t.Errorf("Expected errors in group %d", id)
			}
		}(i)
	}

	wg.Wait()
}

func BenchmarkErrorGroupPool(b *testing.B) {
	const sampleCount = 5

	sampleErrors := make([]error, sampleCount)
	for i := range sampleErrors {
		sampleErrors[i] = fmt.Errorf("%w %d", errIndexed, i)
	}

	b.Run("WithPool", func(b *testing.B) {
		benchPoolWithPool(b, sampleErrors)
	})

	b.Run("WithoutPool", func(b *testing.B) {
		benchPoolWithoutPool(b, sampleErrors)
	})
}

func benchPoolWithPool(b *testing.B, sampleErrors []error) {
	b.Helper()

	pool := NewErrorGroupPool(exactCapacity)

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
}

func benchPoolWithoutPool(b *testing.B, sampleErrors []error) {
	b.Helper()

	b.ReportAllocs()

	for range b.N {
		eg := NewErrorGroup()
		for _, err := range sampleErrors {
			eg.Add(err)
		}

		_ = eg.Error()
	}
}

func TestErrorGroupJoin(t *testing.T) {
	t.Parallel()

	eg := NewErrorGroup()
	eg.Add(errFirst)
	eg.Add(errSecond)

	joined := eg.Join()
	if joined == nil {
		t.Fatal("expected joined error")
	}

	if !errors.Is(joined, errFirst) || !errors.Is(joined, errSecond) {
		t.Fatal("joined error does not contain original errors")
	}

	eg.Clear()

	if eg.Join() != nil {
		t.Fatal("expected nil when joining empty group")
	}
}
