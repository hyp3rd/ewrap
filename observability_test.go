package ewrap

import "testing"

// recordingObserver implements Observer for tests.
type recordingObserver struct {
	errorCount int
}

func (r *recordingObserver) RecordError(string) {
	r.errorCount++
}

func TestErrorLogRecordsObserver(t *testing.T) {
	t.Parallel()

	obs := &recordingObserver{}

	err := New("boom", WithObserver(obs))
	err.Log()

	if obs.errorCount != 1 {
		t.Fatalf("expected 1 error recorded, got %d", obs.errorCount)
	}
}

func TestObserverInheritanceInWrap(t *testing.T) {
	t.Parallel()

	obs := &recordingObserver{}

	original := New("original error", WithObserver(obs))
	wrapped := Wrap(original, "wrapped error")

	original.Log()
	wrapped.Log()

	if obs.errorCount != 2 {
		t.Fatalf("expected 2 errors recorded, got %d", obs.errorCount)
	}
}

func TestObserverIsOptional(t *testing.T) {
	t.Parallel()

	err := New(msgTestError)
	err.Log() // Should not panic without an observer
}
