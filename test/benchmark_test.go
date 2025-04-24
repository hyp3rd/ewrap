package test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/hyp3rd/ewrap/pkg/ewrap"
)

// mockLogger implements a minimal logger for benchmarking
type mockLogger struct{}

func (m *mockLogger) Error(msg string, keysAndValues ...any) {}
func (m *mockLogger) Debug(msg string, keysAndValues ...any) {}
func (m *mockLogger) Info(msg string, keysAndValues ...any)  {}

// BenchmarkNew measures the performance of creating new errors
func BenchmarkNew(b *testing.B) {
	logger := &mockLogger{}
	ctx := context.Background()

	b.Run("Simple", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			_ = ewrap.New("simple error")
		}
	})

	b.Run("WithContext", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			_ = ewrap.New("error with context",
				ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityError))
		}
	})

	b.Run("WithLogger", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_ = ewrap.New("error with logger",
				ewrap.WithLogger(logger))
		}
	})

	b.Run("FullFeatures", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			_ = ewrap.New("full featured error",
				ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityError),
				ewrap.WithLogger(logger)).
				WithMetadata("key1", "value1").
				WithMetadata("key2", 42)
		}
	})
}

// BenchmarkWrap measures the performance of wrapping errors
func BenchmarkWrap(b *testing.B) {
	logger := &mockLogger{}
	ctx := context.Background()
	baseErr := errors.New("base error")

	b.Run("Simple", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_ = ewrap.Wrap(baseErr, "wrapped error")
		}
	})

	b.Run("NestedWraps", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			err1 := ewrap.Wrap(baseErr, "level 1")
			err2 := ewrap.Wrap(err1, "level 2")
			_ = ewrap.Wrap(err2, "level 3")
		}
	})

	b.Run("WithContext", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_ = ewrap.Wrap(baseErr, "wrapped with context",
				ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityError))
		}
	})

	b.Run("FullFeatures", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_ = ewrap.Wrap(baseErr, "full featured wrap",
				ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityError),
				ewrap.WithLogger(logger)).
				WithMetadata("key1", "value1")
		}
	})
}

// BenchmarkErrorGroup measures the performance of error group operations
func BenchmarkErrorGroup(b *testing.B) {
	b.Run("AddErrors", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			group := ewrap.NewErrorGroup()

			for j := 0; j < 10; j++ {
				group.Add(fmt.Errorf("error %d", j))
			}
		}
	})

	b.Run("ConcurrentAdd", func(b *testing.B) {
		group := ewrap.NewErrorGroup()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				group.Add(fmt.Errorf("error %d", i))
				i++
			}
		})
	})
}

// BenchmarkFormatting measures the performance of error formatting
func BenchmarkFormatting(b *testing.B) {
	err := ewrap.New("test error",
		ewrap.WithContext(context.Background(), ewrap.ErrorTypeDatabase, ewrap.SeverityError)).
		WithMetadata("key1", "value1").
		WithMetadata("key2", 42)

	b.Run("ToJSON", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			_, _ = err.ToJSON()
		}
	})

	b.Run("ToJSONWithOptions", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = err.ToJSON(
				ewrap.WithTimestampFormat(time.RFC3339),
				ewrap.WithStackTrace(true),
			)
		}
	})

	b.Run("ToYAML", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			_, _ = err.ToYAML()
		}
	})
}

// BenchmarkCircuitBreaker measures the performance of circuit breaker operations
func BenchmarkCircuitBreaker(b *testing.B) {
	b.Run("RecordFailure", func(b *testing.B) {
		cb := ewrap.NewCircuitBreaker("test", 5, time.Second)

		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			cb.RecordFailure()
			if i%5 == 0 {
				cb.RecordSuccess() // Reset occasionally
			}
		}
	})

	b.Run("ConcurrentOperations", func(b *testing.B) {
		cb := ewrap.NewCircuitBreaker("test", 1000, time.Second)

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				if cb.CanExecute() {
					cb.RecordSuccess()
				} else {
					cb.RecordFailure()
				}
			}
		})
	})
}

// BenchmarkMetadataOperations measures the performance of metadata operations
func BenchmarkMetadataOperations(b *testing.B) {
	b.Run("AddMetadata", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			err := ewrap.New("test error")
			for j := 0; j < 5; j++ {
				err.WithMetadata(fmt.Sprintf("key%d", j), j)
			}
		}
	})

	b.Run("GetMetadata", func(b *testing.B) {
		err := ewrap.New("test error")
		for i := 0; i < 5; i++ {
			err.WithMetadata(fmt.Sprintf("key%d", i), i)
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = err.GetMetadata("key3")
		}
	})
}

// BenchmarkStackTrace measures the performance of stack trace operations
func BenchmarkStackTrace(b *testing.B) {
	err := ewrap.New("test error")

	b.Run("CaptureStack", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			_ = ewrap.CaptureStack()
		}
	})

	b.Run("FormatStack", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			_ = err.Stack()
		}
	})
}
