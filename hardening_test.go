package ewrap

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/goccy/go-json"
)

const (
	deepChainDepth          = 200
	expectedHardeningSeed   = 1024
	hardeningExpectedSuffix = "boom"
)

// TestDeepChain verifies Error()/Stack()/Unwrap walk arbitrarily deep chains
// without exhausting the stack or losing context.
func TestDeepChain(t *testing.T) {
	t.Parallel()

	err := errRoot
	for i := range deepChainDepth {
		err = Wrap(err, fmt.Sprintf("layer-%d", i))
	}

	if !errors.Is(err, errRoot) {
		t.Fatal("errors.Is should find root through deep chain")
	}

	msg := err.Error()
	if !strings.Contains(msg, msgRoot) {
		t.Errorf("expected root in message, got %q", msg)
	}

	if !strings.Contains(msg, fmt.Sprintf("layer-%d", deepChainDepth-1)) {
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

	wrapped := Wrap(errSentinel, "outer")

	if !errors.Is(wrapped, errSentinel) {
		t.Error("expected sentinel match through ewrap.Wrap")
	}

	if errors.Is(wrapped, errOtherSentinel) {
		t.Error("must not match a different error with the same text")
	}
}

// TestNewfWithW asserts that %w preserves the cause chain and message.
func TestNewfWithW(t *testing.T) {
	t.Parallel()

	err := Newf("wrapped: %w", errRootCause)

	if !errors.Is(err, errRootCause) {
		t.Error("errors.Is must walk through %w")
	}

	const want = "wrapped: root cause"

	if got := err.Error(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestWrapStackCapturesWrapSite verifies Wrap captures its own frames.
func TestWrapStackCapturesWrapSite(t *testing.T) {
	t.Parallel()

	root := New(msgRoot)

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
	seeds := []string{"", hardeningExpectedSuffix, "weird \x00 byte", strings.Repeat("a", expectedHardeningSeed)}
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
