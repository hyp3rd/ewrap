package test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/hyp3rd/ewrap"
	"github.com/hyp3rd/ewrap/breaker"
)

const (
	profileIterations    = 10000
	profileWrapDepth     = 10
	profileGoroutines    = 100
	profileGoroutineWork = 1000
	profileGroupAdds     = 5
	profileBreakerMax    = 1000
)

var (
	errHeapProfileMissing      = errors.New("could not find heap profile")
	errGoroutineProfileMissing = errors.New("could not find goroutine profile")
)

type profileCase struct {
	name    string
	setup   func()
	cleanup func()
	profile func(file *os.File) error
}

func cpuProfileCase() profileCase {
	return profileCase{
		name:    "cpu",
		setup:   func() {},
		cleanup: func() {},
		profile: func(file *os.File) error {
			err := pprof.StartCPUProfile(file)
			if err != nil {
				return fmt.Errorf("could not start CPU profile: %w", err)
			}

			profileCPU()
			pprof.StopCPUProfile()

			return nil
		},
	}
}

func heapProfileCase() profileCase {
	return profileCase{
		name:    "heap",
		setup:   forceGC,
		cleanup: forceGC,
		profile: func(file *os.File) error {
			prof := pprof.Lookup("heap")
			if prof == nil {
				return errHeapProfileMissing
			}

			profileMemory()

			return prof.WriteTo(file, 0)
		},
	}
}

func goroutineProfileCase() profileCase {
	return profileCase{
		name:    "goroutine",
		setup:   func() {},
		cleanup: func() {},
		profile: func(file *os.File) error {
			prof := pprof.Lookup("goroutine")
			if prof == nil {
				return errGoroutineProfileMissing
			}

			profileGoroutinesFn()

			return prof.WriteTo(file, 0)
		},
	}
}

// forceGC explicitly triggers garbage collection so heap profiles capture a
// post-collection snapshot. Profile-only helper; production code never needs
// this.
//
//nolint:revive
func forceGC() {
	// explicit GC is intentional for profile snapshots.
	runtime.GC()
}

// TestProfileErrorOperations runs a comprehensive profiling suite for error operations.
// It generates CPU, memory, and goroutine profiles to analyze the performance characteristics
// of our error handling implementation.
//
// This test mutates the global runtime.MemProfileRate and emits profile files
// to the working directory, so it cannot be parallelized.
//
//nolint:paralleltest // mutates runtime.MemProfileRate global state
func TestProfileErrorOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping profiling in short mode")
	}

	runtime.MemProfileRate = 1

	for _, profile := range []profileCase{cpuProfileCase(), heapProfileCase(), goroutineProfileCase()} {
		//nolint:paralleltest // sequential by design — see TestProfileErrorOperations
		t.Run(profile.name, func(t *testing.T) {
			runProfileCase(t, profile)
		})
	}
}

func runProfileCase(t *testing.T, profile profileCase) {
	t.Helper()

	filename := filepath.Clean(fmt.Sprintf("profile_%s.prof", profile.name))

	file, err := os.Create(filename)
	if err != nil {
		t.Fatalf("could not create %s profile file: %v", profile.name, err)
	}

	defer func() {
		closeErr := file.Close()
		if closeErr != nil {
			t.Errorf("error closing profile file: %v", closeErr)
		}
	}()

	profile.setup()

	profileErr := profile.profile(file)
	if profileErr != nil {
		t.Fatalf("error writing %s profile: %v", profile.name, profileErr)
	}

	profile.cleanup()

	t.Logf("Profile written to %s", filename)
}

func profileCPU() {
	ctx := context.Background()
	logger := &mockLogger{}

	for i := range profileIterations {
		err := ewrap.New(fmt.Sprintf("error %d", i),
			ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityCritical),
			ewrap.WithLogger(logger))

		err = ewrap.Wrap(err, "wrapped")
		_ = err.WithMetadata("key", i)

		group := ewrap.NewErrorGroup()
		for range profileGroupAdds {
			group.Add(err)
		}

		_, _ = err.ToJSON()
		_, _ = err.ToYAML()
	}
}

func profileMemory() {
	forceGC()

	captured := make([]*ewrap.Error, 0, profileIterations)
	ctx := context.Background()

	for i := range profileIterations {
		err := ewrap.New(fmt.Sprintf("error %d", i),
			ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityCritical))

		for j := range profileWrapDepth {
			err = ewrap.Wrap(err, fmt.Sprintf("layer %d", j))
			_ = err.WithMetadata(fmt.Sprintf("key%d", j), j)
		}

		captured = append(captured, err)
	}

	if len(captured) == 0 {
		panic("captured slice unexpectedly empty")
	}
}

func profileGoroutinesFn() {
	done := make(chan bool)

	cb := breaker.New("test", profileBreakerMax, time.Second)

	for i := range profileGoroutines {
		go runProfileGoroutine(cb, i, done)
	}

	for range profileGoroutines {
		<-done
	}
}

func runProfileGoroutine(cb *breaker.Breaker, id int, done chan<- bool) {
	for j := range profileGoroutineWork {
		if cb.CanExecute() {
			err := ewrap.New(fmt.Sprintf("error %d-%d", id, j))

			_ = err.WithMetadata("goroutine", id)

			cb.RecordSuccess()
		} else {
			cb.RecordFailure()
		}
	}

	done <- true
}
