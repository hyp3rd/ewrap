package ewrap

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// TestDeepChain verifies Error()/Stack()/Unwrap walk arbitrarily deep chains
// without exhausting the stack or losing context.
func TestDeepChain(t *testing.T) {
	t.Parallel()

	const depth = 200

	root := errors.New("root")

	err := root
	for i := range depth {
		err = Wrap(err, fmt.Sprintf("layer-%d", i))
	}

	if !errors.Is(err, root) {
		t.Fatal("errors.Is should find root through deep chain")
	}

	msg := err.Error()
	if !strings.Contains(msg, "root") {
		t.Errorf("expected root in message, got %q", msg)
	}

	if !strings.Contains(msg, fmt.Sprintf("layer-%d", depth-1)) {
		t.Errorf("expected outermost layer in message, got %q", msg)
	}

	var ewrapErr *Error
	if !errors.As(err, &ewrapErr) {
		t.Fatal("errors.As should find an ewrap.Error")
	}

	if stack := ewrapErr.Stack(); stack == "" {
		t.Error("expected non-empty stack on outermost layer")
	}
}

// TestErrorsIs_ContractWithSentinel covers the contract that the previous
// broken Is() implementation violated.
func TestErrorsIs_ContractWithSentinel(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("sentinel")
	wrapped := Wrap(sentinel, "outer")

	if !errors.Is(wrapped, sentinel) {
		t.Error("expected sentinel match through ewrap.Wrap")
	}

	other := errors.New("sentinel") // same text, different identity
	if errors.Is(wrapped, other) {
		t.Error("must not match a different error with the same text")
	}
}

// TestNewfWithW asserts that %w preserves the cause chain and message.
func TestNewfWithW(t *testing.T) {
	t.Parallel()

	root := errors.New("root cause")
	err := Newf("wrapped: %w", root)

	if !errors.Is(err, root) {
		t.Error("errors.Is must walk through %w")
	}

	if got := err.Error(); got != "wrapped: root cause" {
		t.Errorf("got %q, want %q", got, "wrapped: root cause")
	}
}

// TestWrapStackCapturesWrapSite verifies Wrap captures its own frames.
func TestWrapStackCapturesWrapSite(t *testing.T) {
	t.Parallel()

	root := New("root")

	wrapped := wrapHelper(root)

	if len(wrapped.stack) == 0 {
		t.Fatal("wrapper stack must not be empty")
	}

	if wrapped.Stack() == root.Stack() {
		t.Error("wrapper stack must differ from root stack")
	}
}

// wrapHelper exists to give wrapHelper a distinct frame from the test.
//
//go:noinline
func wrapHelper(err error) *Error {
	return Wrap(err, "from helper")
}

// FuzzJSONRoundTrip checks that ToJSON is robust against arbitrary inputs.
func FuzzJSONRoundTrip(f *testing.F) {
	seeds := []string{"", "boom", "weird \x00 byte", strings.Repeat("a", 1024)}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, msg string) {
		err := New(msg).WithMetadata("fuzzed", true)

		jsonStr, jerr := err.ToJSON()
		if jerr != nil {
			t.Fatalf("ToJSON failed: %v", jerr)
		}

		var out ErrorOutput

		uerr := json.Unmarshal([]byte(jsonStr), &out)
		if uerr != nil {
			t.Fatalf("invalid JSON produced: %v", uerr)
		}

		if out.Message != msg {
			t.Errorf("message round-trip lost data: got %q, want %q", out.Message, msg)
		}
	})
}
