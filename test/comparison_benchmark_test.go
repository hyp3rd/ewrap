package test

import (
	"fmt"
	"testing"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/multierr"

	"github.com/hyp3rd/ewrap"
)

// This test suite compares our implementation against popular error handling libraries
// to provide performance insights and identify optimization opportunities.

func BenchmarkErrorCreation(b *testing.B) {
	const msg = "test error"

	b.Run("ewrap/New", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = ewrap.New(msg)
		}
	})

	b.Run("pkg/errors/New", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = errors.New(msg)
		}
	})

	// b.Run("emperror/errors", func(b *testing.B) {
	// 	b.ReportAllocs()
	// 	var handler emperror.Handler = newHandler()
	// 	// Recover from panics and handle them as errors
	// 	defer emperror.HandleRecover(handler)
	// 	for i := 0; i < b.N; i++ {
	// 		_ = emperror.WithDetails(msg,
	// 			keyval.Pairs{"operation": "test"})
	// 	}
	// })
}

func BenchmarkErrorWrapping(b *testing.B) {
	baseErr := fmt.Errorf("base error")
	const wrapMsg = "wrapped error"

	b.Run("ewrap/Wrap", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = ewrap.Wrap(baseErr, wrapMsg)
		}
	})

	b.Run("pkg/errors/Wrap", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = errors.Wrap(baseErr, wrapMsg)
		}
	})

	b.Run("emperror/Wrap", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = emperror.Wrap(baseErr, wrapMsg)
		}
	})
}

func BenchmarkErrorGroups(b *testing.B) {
	errs := make([]error, 10)
	for i := range errs {
		errs[i] = fmt.Errorf("error %d", i)
	}

	b.Run("ewrap/ErrorGroup", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			group := ewrap.NewErrorGroup()
			for _, err := range errs {
				group.Add(err)
			}
			_ = group.Error()
		}
	})

	b.Run("hashicorp/multierror", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var result *multierror.Error

			for _, err := range errs {
				result = multierror.Append(result, err)
			}
			_ = result.Error()
		}
	})

	b.Run("uber/multierr", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var err error

			for _, e := range errs {
				err = multierr.Append(err, e)
			}

			_ = err.Error()
		}
	})
}
