package demografix

// DemografixError is the base error type for every failure the SDK reports. Every
// typed error embeds it, so it carries the HTTP status, the API message, and the
// response quota when the rate-limit headers were present.
//
// Discover a specific error type with errors.As:
//
//	var rate *demografix.RateLimitError
//	if errors.As(err, &rate) {
//	        time.Sleep(time.Duration(rate.Quota.Reset) * time.Second)
//	}
//
// The base type itself is also matchable, so errors.As against *DemografixError
// succeeds for any SDK error.
type DemografixError struct {
	// Status is the HTTP status code, or 0 when no response was received.
	Status int
	// Message is the API error string passed through unchanged.
	Message string
	// Quota is the parsed rate-limit state, or nil when no headers were present.
	Quota *Quota
}

// Error implements the error interface.
func (e *DemografixError) Error() string { return e.Message }

// AuthError reports an invalid or rejected API key (HTTP 401).
type AuthError struct{ DemografixError }

// Unwrap exposes the embedded base, so errors.As matches *DemografixError too.
func (e *AuthError) Unwrap() error { return &e.DemografixError }

// SubscriptionError reports an inactive or expired subscription (HTTP 402).
type SubscriptionError struct{ DemografixError }

// Unwrap exposes the embedded base, so errors.As matches *DemografixError too.
func (e *SubscriptionError) Unwrap() error { return &e.DemografixError }

// ValidationError reports a rejected request (HTTP 422). It is also raised
// client-side, before any HTTP call, when a batch holds more than ten names.
type ValidationError struct{ DemografixError }

// Unwrap exposes the embedded base, so errors.As matches *DemografixError too.
func (e *ValidationError) Unwrap() error { return &e.DemografixError }

// RateLimitError reports an exhausted quota (HTTP 429). Quota is always populated;
// read Quota.Reset for the seconds to wait before retrying.
type RateLimitError struct{ DemografixError }

// Unwrap exposes the embedded base, so errors.As matches *DemografixError too.
func (e *RateLimitError) Unwrap() error { return &e.DemografixError }

// TransportError reports a network failure, a timeout, or a response body that is
// not JSON. Status and Quota may be absent. When the failure came from an
// underlying call, Err holds the original error, so errors.Is and errors.As reach
// it (for example errors.Is(err, context.DeadlineExceeded) on a timeout).
type TransportError struct {
	DemografixError
	// Err is the underlying cause, or nil for a synthesized failure such as a
	// non-JSON response body.
	Err error
}

// Unwrap exposes both the embedded base and the underlying cause, so errors.As
// matches *DemografixError and errors.Is reaches the wrapped transport error.
func (e *TransportError) Unwrap() []error {
	if e.Err == nil {
		return []error{&e.DemografixError}
	}
	return []error{&e.DemografixError, e.Err}
}

// newAPIError maps an HTTP status code to the matching typed error. Each returned
// value embeds DemografixError, so the base fields and the error interface are
// available, and errors.As against the concrete type succeeds.
func newAPIError(status int, message string, quota *Quota) error {
	base := DemografixError{Status: status, Message: message, Quota: quota}
	switch status {
	case 401:
		return &AuthError{base}
	case 402:
		return &SubscriptionError{base}
	case 422:
		return &ValidationError{base}
	case 429:
		return &RateLimitError{base}
	default:
		return &base
	}
}

// newValidationError builds a client-side ValidationError with no status or quota.
func newValidationError(message string) error {
	return &ValidationError{DemografixError{Status: 0, Message: message}}
}

// wrapTransportError builds a transport failure that carries the underlying cause,
// so errors.Is and errors.As can reach it.
func wrapTransportError(message string, err error) error {
	return &TransportError{DemografixError: DemografixError{Status: 0, Message: message}, Err: err}
}
