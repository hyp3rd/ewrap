package ewrap

import "errors"

// Shared test fixtures. Centralising these silences goconst/revive
// add-constant warnings and keeps tests readable.

const (
	msgTest         = "test"
	msgTestError    = "test error"
	msgBoom         = "boom"
	msgKey          = "key"
	msgValue        = "value"
	msgSomeStack    = "some stack trace"
	msgOriginal     = "original"
	msgOriginalErr  = "original error"
	msgCauseError   = "cause error"
	msgFirst        = "first"
	msgSecond       = "second"
	msgRoot         = "root"
	msgRootCause    = "root cause"
	msgPlain        = "plain"
	msgSentinel     = "sentinel"
	msgWrapped      = "wrapped"
	msgStandardErr  = "standard error"
	msgErrorMessage = "error message"

	defaultMaxAttempts = 3
	smallStringLength  = 100
	concurrencyLimit   = 100
)

var (
	errOriginal      = errors.New(msgOriginal)
	errOriginalLong  = errors.New(msgOriginalErr)
	errCause         = errors.New(msgCauseError)
	errFirst         = errors.New(msgFirst)
	errSecond        = errors.New(msgSecond)
	errSentinel      = errors.New(msgSentinel)
	errOtherSentinel = errors.New(msgSentinel) // distinct identity, identical text
	errOther         = errors.New("other")
	errPlain         = errors.New(msgPlain)
	errStandard      = errors.New(msgStandardErr)
	errRoot          = errors.New(msgRoot)
	errRootCause     = errors.New(msgRootCause)
	errFromGoroutine = errors.New("error from goroutine")
	errIndexed       = errors.New("indexed error")
)
