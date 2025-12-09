package test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/hyp3rd/ewrap"
)

// TestProfileErrorOperations runs a comprehensive profiling suite for error operations.
// It generates CPU, memory, and goroutine profiles to analyze the performance characteristics
// of our error handling implementation.
func TestProfileErrorOperations(t *testing.T) {
	// Skip in normal testing
	if testing.Short() {
		t.Skip("Skipping profiling in short mode")
	}

	// Enable memory profiling with a rate of 1 means we sample every allocation
	runtime.MemProfileRate = 1

	profiles := []struct {
		name     string
		profName string // The name pprof uses internally
		setup    func()
		cleanup  func()
		profile  func(f *os.File) error
	}{
		{
			name: "cpu",
			setup: func() {
				// No specific setup needed for CPU profiling
			},
			cleanup: func() {
				// No specific cleanup needed for CPU profiling
			},
			profile: func(f *os.File) error {
				err := pprof.StartCPUProfile(f)
				if err != nil {
					return fmt.Errorf("could not start CPU profile: %w", err)
				}

				profileCPU()
				pprof.StopCPUProfile()

				return nil
			},
		},
		{
			name:     "heap",
			profName: "heap",
			setup: func() {
				// Force garbage collection before memory profiling
				runtime.GC()
			},
			cleanup: func() {
				// Force garbage collection after memory profiling
				runtime.GC()
			},
			profile: func(f *os.File) error {
				p := pprof.Lookup("heap")
				if p == nil {
					return errors.New("could not find heap profile")
				}

				profileMemory()

				return p.WriteTo(f, 0)
			},
		},
		{
			name:     "goroutine",
			profName: "goroutine",
			setup:    func() {},
			cleanup:  func() {},
			profile: func(f *os.File) error {
				p := pprof.Lookup("goroutine")
				if p == nil {
					return errors.New("could not find goroutine profile")
				}

				profileGoroutines()

				return p.WriteTo(f, 0)
			},
		},
	}

	for _, profile := range profiles {
		t.Run(profile.name, func(t *testing.T) {
			// Create profile file
			filename := fmt.Sprintf("profile_%s.prof", profile.name)

			f, err := os.Create(filename)
			if err != nil {
				t.Fatalf("could not create %s profile file: %v", profile.name, err)
			}

			defer func() {
				err := f.Close()
				if err != nil {
					t.Errorf("error closing profile file: %v", err)
				}
			}()

			// Run setup
			profile.setup()

			// Run profiling
			if err := profile.profile(f); err != nil {
				t.Fatalf("error writing %s profile: %v", profile.name, err)
			}

			// Run cleanup
			profile.cleanup()

			t.Logf("Profile written to %s", filename)
		})
	}
}

func profileCPU() {
	// Simulate intensive error handling operations
	ctx := context.Background()
	logger := &mockLogger{}

	for i := range 10000 {
		err := ewrap.New(fmt.Sprintf("error %d", i),
			ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityCritical),
			ewrap.WithLogger(logger))

		err = ewrap.Wrap(err, "wrapped")
		err.WithMetadata("key", i)

		group := ewrap.NewErrorGroup()
		for range 5 {
			group.Add(err)
		}

		_, _ = err.ToJSON()
		_, _ = err.ToYAML()
	}
}

func profileMemory() {
	// Force GC before profiling
	runtime.GC()

	// Simulate memory-intensive operations
	var errors []*ewrap.Error

	ctx := context.Background()

	for i := range 10000 {
		err := ewrap.New(fmt.Sprintf("error %d", i),
			ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityCritical))

		for j := range 10 {
			err = ewrap.Wrap(err, fmt.Sprintf("layer %d", j))
			err.WithMetadata(fmt.Sprintf("key%d", j), j)
		}

		errors = append(errors, err)
	}

	for err := range errors {
		fmt.Printf("err: %v\n", err)
	}
}

func profileGoroutines() {
	// Simulate concurrent error handling
	const numGoroutines = 100

	done := make(chan bool)

	cb := ewrap.NewCircuitBreaker("test", 1000, time.Second)

	for i := range numGoroutines {
		go func(id int) {
			for j := range 1000 {
				if cb.CanExecute() {
					err := ewrap.New(fmt.Sprintf("error %d-%d", id, j))
					err.WithMetadata("goroutine", id)
					cb.RecordSuccess()
				} else {
					cb.RecordFailure()
				}
			}

			done <- true
		}(i)
	}

	for range numGoroutines {
		<-done
	}
}
