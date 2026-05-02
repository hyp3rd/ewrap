package ewrap

import (
	"fmt"
	"net/http"
	"testing"
)

func TestHTTPStatus(t *testing.T) {
	t.Parallel()

	t.Run("unset returns zero", func(t *testing.T) {
		t.Parallel()

		err := New(msgPlain)
		if got := HTTPStatus(err); got != 0 {
			t.Errorf("got %d, want 0", got)
		}
	})

	t.Run("explicit on outer error", func(t *testing.T) {
		t.Parallel()

		err := New("forbidden", WithHTTPStatus(http.StatusForbidden))
		if got := HTTPStatus(err); got != http.StatusForbidden {
			t.Errorf("got %d, want %d", got, http.StatusForbidden)
		}
	})

	t.Run("walks chain to find status", func(t *testing.T) {
		t.Parallel()

		root := New("not found", WithHTTPStatus(http.StatusNotFound))
		wrapped := fmt.Errorf("layered: %w", root)

		if got := HTTPStatus(wrapped); got != http.StatusNotFound {
			t.Errorf("got %d, want %d", got, http.StatusNotFound)
		}
	})

	t.Run("non-ewrap error returns zero", func(t *testing.T) {
		t.Parallel()

		if got := HTTPStatus(errPlain); got != 0 {
			t.Errorf("got %d, want 0", got)
		}
	})
}

func TestIsRetryable(t *testing.T) {
	t.Parallel()

	t.Run("explicit retryable true", func(t *testing.T) {
		t.Parallel()

		err := New("transient", WithRetryable(true))
		if !IsRetryable(err) {
			t.Error("expected retryable true")
		}
	})

	t.Run("explicit retryable false", func(t *testing.T) {
		t.Parallel()

		err := New("permanent", WithRetryable(false))
		if IsRetryable(err) {
			t.Error("expected retryable false")
		}
	})

	t.Run("unset and no Temporary defaults false", func(t *testing.T) {
		t.Parallel()

		if IsRetryable(New(msgPlain)) {
			t.Error("expected retryable false for unclassified error")
		}
	})

	t.Run("falls through to Temporary interface", func(t *testing.T) {
		t.Parallel()

		err := temporaryError{msg: "transient", temp: true}
		if !IsRetryable(err) {
			t.Error("expected retryable true via Temporary fallback")
		}
	})

	t.Run("walks chain", func(t *testing.T) {
		t.Parallel()

		inner := New("transient", WithRetryable(true))
		outer := Wrap(inner, "outer")

		if !IsRetryable(outer) {
			t.Error("expected retryable true via chain inheritance")
		}
	})
}

type temporaryError struct {
	msg  string
	temp bool
}

func (t temporaryError) Error() string   { return t.msg }
func (t temporaryError) Temporary() bool { return t.temp }

func TestSafeError(t *testing.T) {
	t.Parallel()

	t.Run("uses safe message when set", func(t *testing.T) {
		t.Parallel()

		err := New("user secret123 failed", WithSafeMessage("user [redacted] failed"))
		if got := err.SafeError(); got != "user [redacted] failed" {
			t.Errorf("got %q, want redacted form", got)
		}
	})

	t.Run("falls back to msg when no safe set", func(t *testing.T) {
		t.Parallel()

		err := New("public message")
		if got := err.SafeError(); got != "public message" {
			t.Errorf("got %q, want %q", got, "public message")
		}
	})

	t.Run("walks ewrap chain redacting each layer", func(t *testing.T) {
		t.Parallel()

		root := New("token=abc", WithSafeMessage("token=[redacted]"))
		wrapped := Wrap(root, "auth failed for user@example.com",
			WithSafeMessage("auth failed for [redacted]"))

		got := wrapped.SafeError()
		want := "auth failed for [redacted]: token=[redacted]"

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestFormatVerbs(t *testing.T) {
	t.Parallel()

	err := New(msgBoom)

	if got := fmt.Sprintf("%s", err); got != msgBoom {
		t.Errorf("%%s: got %q, want %q", got, msgBoom)
	}

	if got := fmt.Sprintf("%v", err); got != msgBoom {
		t.Errorf("%%v: got %q, want %q", got, msgBoom)
	}

	const quotedBoom = `"boom"`

	if got := fmt.Sprintf("%q", err); got != quotedBoom {
		t.Errorf("%%q: got %q, want %q", got, quotedBoom)
	}

	plus := fmt.Sprintf("%+v", err)
	if !errorStartsWithMessage(plus, msgBoom) {
		t.Errorf("%%+v: got %q, expected message followed by stack", plus)
	}
}

// errorStartsWithMessage tests that the formatted output begins with the
// expected error message and includes additional content (the stack trace).
func errorStartsWithMessage(formatted, message string) bool {
	if len(formatted) <= len(message) {
		return false
	}

	return formatted[:len(message)] == message
}
