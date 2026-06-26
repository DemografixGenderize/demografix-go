package demografix

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// fixtureHeaders are the three rate-limit headers present on every response in the
// INTERFACE.md fixtures.
func fixtureHeaders() http.Header {
	h := http.Header{}
	h.Set("x-rate-limit-limit", "25000")
	h.Set("x-rate-limit-remaining", "24987")
	h.Set("x-rate-limit-reset", "1314000")
	return h
}

// roundTripFunc is a function adapter implementing http.RoundTripper, the transport
// seam. The Client's hardcoded host is left untouched; the request is intercepted
// here and a canned response is returned instead of a network call.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// captureClient builds a Client with a transport that records the last request into
// the returned pointer slot and replies with the canned response.
func captureClient(status int, headers http.Header, body string, slot **http.Request) *Client {
	c := New("test-key")
	c.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		*slot = req
		return &http.Response{
			StatusCode: status,
			Header:     headers,
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})
	return c
}

// failClient builds a Client whose transport returns an error, simulating a network
// failure. It also records whether RoundTrip was ever invoked.
func failClient(called *bool) *Client {
	c := New("test-key")
	c.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		*called = true
		return nil, errors.New("dial tcp: connection refused")
	})
	return c
}

const (
	genderizeSingle    = `{ "count": 1352696, "name": "peter", "gender": "male", "probability": 1.0 }`
	genderizeCountry   = `{ "count": 196601, "name": "kim", "gender": "female", "country_id": "US", "probability": 0.94 }`
	genderizeNull      = `{ "name": "xÿz", "gender": null, "probability": 0.0, "count": 0 }`
	agifyBatch         = `[ { "count": 311558, "name": "michael", "age": 57 }, { "count": 55682, "name": "matthew", "age": 48 } ]`
	agifyNull          = `{ "name": "xÿz", "age": null, "count": 0 }`
	nationalizeSingle  = `{ "count": 100783, "name": "nguyen", "country": [ { "country_id": "VN", "probability": 0.891132 }, { "country_id": "MO", "probability": 0.019031 } ] }`
	nationalizeNull    = `{ "name": "xÿz", "country": [], "count": 0 }`
	genderizeBatchBody = `[ { "count": 1352696, "name": "peter", "gender": "male", "probability": 1.0 }, { "count": 196601, "name": "lois", "gender": "female", "probability": 0.98 } ]`
)

// --- Assertion 1: single parse + quota.remaining == 24987 ---

func TestGenderizeSingle(t *testing.T) {
	var req *http.Request
	c := captureClient(http.StatusOK, fixtureHeaders(), genderizeSingle, &req)

	res, err := c.Genderize(context.Background(), "peter")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Name != "peter" {
		t.Errorf("name = %q, want peter", res.Name)
	}
	if res.Gender != "male" {
		t.Errorf("gender = %q, want male", res.Gender)
	}
	if res.Probability != 1.0 {
		t.Errorf("probability = %v, want 1.0", res.Probability)
	}
	if res.Count != 1352696 {
		t.Errorf("count = %d, want 1352696", res.Count)
	}
	if res.Quota.Remaining != 24987 {
		t.Errorf("quota.remaining = %d, want 24987", res.Quota.Remaining)
	}
	if res.Quota.Limit != 25000 {
		t.Errorf("quota.limit = %d, want 25000", res.Quota.Limit)
	}
	if res.Quota.Reset != 1314000 {
		t.Errorf("quota.reset = %d, want 1314000", res.Quota.Reset)
	}

	// Single-name query uses name=, not name[].
	q := req.URL.Query()
	if q.Get("name") != "peter" {
		t.Errorf("query name = %q, want peter", q.Get("name"))
	}
	if _, ok := q["name[]"]; ok {
		t.Error("single request must not use name[]")
	}
	if got := req.Header.Get("User-Agent"); got != "demografix-go/0.1.0" {
		t.Errorf("User-Agent = %q, want demografix-go/0.1.0", got)
	}
	if req.URL.Host != "api.genderize.io" {
		t.Errorf("host = %q, want api.genderize.io", req.URL.Host)
	}
}

func TestAgifySingle(t *testing.T) {
	var req *http.Request
	c := captureClient(http.StatusOK, fixtureHeaders(), `{ "count": 311558, "name": "michael", "age": 57 }`, &req)

	res, err := c.Agify(context.Background(), "michael")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Age == nil || *res.Age != 57 {
		t.Errorf("age = %v, want 57", res.Age)
	}
	if res.Count != 311558 {
		t.Errorf("count = %d, want 311558", res.Count)
	}
	if res.Quota.Remaining != 24987 {
		t.Errorf("quota.remaining = %d, want 24987", res.Quota.Remaining)
	}
}

func TestNationalizeSingle(t *testing.T) {
	var req *http.Request
	c := captureClient(http.StatusOK, fixtureHeaders(), nationalizeSingle, &req)

	res, err := c.Nationalize(context.Background(), "nguyen")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Country) != 2 {
		t.Fatalf("country length = %d, want 2", len(res.Country))
	}
	if res.Country[0].CountryID != "VN" {
		t.Errorf("country[0].country_id = %q, want VN", res.Country[0].CountryID)
	}
	if res.Country[0].Probability != 0.891132 {
		t.Errorf("country[0].probability = %v, want 0.891132", res.Country[0].Probability)
	}
	if res.Count != 100783 {
		t.Errorf("count = %d, want 100783", res.Count)
	}
	if res.Quota.Remaining != 24987 {
		t.Errorf("quota.remaining = %d, want 24987", res.Quota.Remaining)
	}
}

// --- Assertion 2: batch parses results in order + quota ---

func TestAgifyBatchOrder(t *testing.T) {
	var req *http.Request
	c := captureClient(http.StatusOK, fixtureHeaders(), agifyBatch, &req)

	res, err := c.AgifyBatch(context.Background(), []string{"michael", "matthew"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Results) != 2 {
		t.Fatalf("results length = %d, want 2", len(res.Results))
	}
	if res.Results[0].Name != "michael" || res.Results[1].Name != "matthew" {
		t.Errorf("results order = [%q %q], want [michael matthew]", res.Results[0].Name, res.Results[1].Name)
	}
	if res.Results[0].Age == nil || *res.Results[0].Age != 57 {
		t.Errorf("results[0].age = %v, want 57", res.Results[0].Age)
	}
	if res.Results[1].Age == nil || *res.Results[1].Age != 48 {
		t.Errorf("results[1].age = %v, want 48", res.Results[1].Age)
	}
	if res.Quota.Remaining != 24987 {
		t.Errorf("quota.remaining = %d, want 24987", res.Quota.Remaining)
	}

	// Batch query uses repeated name[]= in input order.
	names := req.URL.Query()["name[]"]
	if len(names) != 2 || names[0] != "michael" || names[1] != "matthew" {
		t.Errorf("name[] = %v, want [michael matthew]", names)
	}
	if req.URL.Query().Get("name") != "" {
		t.Error("batch request must not use name=")
	}
}

func TestGenderizeBatchOrder(t *testing.T) {
	var req *http.Request
	c := captureClient(http.StatusOK, fixtureHeaders(), genderizeBatchBody, &req)

	res, err := c.GenderizeBatch(context.Background(), []string{"peter", "lois"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Results) != 2 {
		t.Fatalf("results length = %d, want 2", len(res.Results))
	}
	if res.Results[0].Name != "peter" || res.Results[1].Name != "lois" {
		t.Errorf("results order = [%q %q], want [peter lois]", res.Results[0].Name, res.Results[1].Name)
	}
	if res.Results[1].Gender != "female" {
		t.Errorf("results[1].gender = %q, want female", res.Results[1].Gender)
	}
}

// --- Assertion 3: null prediction returns null/empty without error ---

func TestGenderizeNull(t *testing.T) {
	var req *http.Request
	c := captureClient(http.StatusOK, fixtureHeaders(), genderizeNull, &req)

	res, err := c.Genderize(context.Background(), "xÿz")
	if err != nil {
		t.Fatalf("null prediction must not error: %v", err)
	}
	if res.Gender != "" {
		t.Errorf("gender = %q, want empty string for null", res.Gender)
	}
	if res.Probability != 0.0 {
		t.Errorf("probability = %v, want 0.0", res.Probability)
	}
	if res.Count != 0 {
		t.Errorf("count = %d, want 0", res.Count)
	}
}

func TestAgifyNull(t *testing.T) {
	var req *http.Request
	c := captureClient(http.StatusOK, fixtureHeaders(), agifyNull, &req)

	res, err := c.Agify(context.Background(), "xÿz")
	if err != nil {
		t.Fatalf("null prediction must not error: %v", err)
	}
	if res.Age != nil {
		t.Errorf("age = %v, want nil", res.Age)
	}
	if res.Count != 0 {
		t.Errorf("count = %d, want 0", res.Count)
	}
}

func TestNationalizeNull(t *testing.T) {
	var req *http.Request
	c := captureClient(http.StatusOK, fixtureHeaders(), nationalizeNull, &req)

	res, err := c.Nationalize(context.Background(), "xÿz")
	if err != nil {
		t.Fatalf("null prediction must not error: %v", err)
	}
	if len(res.Country) != 0 {
		t.Errorf("country = %v, want empty", res.Country)
	}
	if res.Count != 0 {
		t.Errorf("count = %d, want 0", res.Count)
	}
}

// --- Assertion 4: country_id round-trips into the request and back ---

func TestGenderizeCountryRoundTrip(t *testing.T) {
	var req *http.Request
	c := captureClient(http.StatusOK, fixtureHeaders(), genderizeCountry, &req)

	res, err := c.Genderize(context.Background(), "kim", WithCountry("US"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.URL.Query().Get("country_id"); got != "US" {
		t.Errorf("request country_id = %q, want US", got)
	}
	if res.CountryID != "US" {
		t.Errorf("response country_id = %q, want US", res.CountryID)
	}
	if res.Gender != "female" {
		t.Errorf("gender = %q, want female", res.Gender)
	}
}

func TestAgifyBatchCountryRoundTrip(t *testing.T) {
	var req *http.Request
	body := `[ { "count": 311558, "name": "michael", "age": 57, "country_id": "US" }, { "count": 55682, "name": "matthew", "age": 48, "country_id": "US" } ]`
	c := captureClient(http.StatusOK, fixtureHeaders(), body, &req)

	res, err := c.AgifyBatch(context.Background(), []string{"michael", "matthew"}, WithCountry("us"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.URL.Query().Get("country_id"); got != "us" {
		t.Errorf("request country_id = %q, want us (passed through verbatim)", got)
	}
	if res.Results[0].CountryID != "US" {
		t.Errorf("response country_id = %q, want US", res.Results[0].CountryID)
	}
}

// --- Assertion 5: batch of 11 names raises ValidationError with NO HTTP call ---

func TestBatchTooLargeNoHTTP(t *testing.T) {
	called := false
	c := failClient(&called)

	names := make([]string, 11)
	for i := range names {
		names[i] = "n"
	}

	_, err := c.GenderizeBatch(context.Background(), names)
	if err == nil {
		t.Fatal("expected ValidationError, got nil")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("error type = %T, want *ValidationError", err)
	}
	if called {
		t.Error("client-side validation must not send an HTTP request")
	}

	// Agify and nationalize enforce the same limit.
	called = false
	if _, err := c.AgifyBatch(context.Background(), names); !errors.As(err, &ve) {
		t.Errorf("agify batch error = %T, want *ValidationError", err)
	}
	if called {
		t.Error("agify batch must not send an HTTP request")
	}
	called = false
	if _, err := c.NationalizeBatch(context.Background(), names); !errors.As(err, &ve) {
		t.Errorf("nationalize batch error = %T, want *ValidationError", err)
	}
	if called {
		t.Error("nationalize batch must not send an HTTP request")
	}
}

func TestBatchExactlyTenAllowed(t *testing.T) {
	var req *http.Request
	c := captureClient(http.StatusOK, fixtureHeaders(), `[]`, &req)

	names := make([]string, 10)
	for i := range names {
		names[i] = "n"
	}
	if _, err := c.GenderizeBatch(context.Background(), names); err != nil {
		t.Fatalf("batch of 10 must be allowed: %v", err)
	}
	if req == nil {
		t.Fatal("a batch of 10 must send an HTTP request")
	}
}

// --- Assertion 6: 401/402/422/429 map to typed errors with status, message, quota ---

func TestErrorMapping(t *testing.T) {
	cases := []struct {
		status  int
		body    string
		message string
		check   func(error) bool
	}{
		{401, `{ "error": "Invalid API key" }`, "Invalid API key", func(e error) bool {
			var t *AuthError
			return errors.As(e, &t)
		}},
		{402, `{ "error": "Subscription is not active" }`, "Subscription is not active", func(e error) bool {
			var t *SubscriptionError
			return errors.As(e, &t)
		}},
		{422, `{ "error": "Missing 'name' parameter" }`, "Missing 'name' parameter", func(e error) bool {
			var t *ValidationError
			return errors.As(e, &t)
		}},
		{429, `{ "error": "Request limit reached" }`, "Request limit reached", func(e error) bool {
			var t *RateLimitError
			return errors.As(e, &t)
		}},
	}

	for _, tc := range cases {
		var req *http.Request
		c := captureClient(tc.status, fixtureHeaders(), tc.body, &req)

		_, err := c.Genderize(context.Background(), "peter")
		if err == nil {
			t.Fatalf("status %d: expected an error", tc.status)
		}
		if !tc.check(err) {
			t.Errorf("status %d: error type = %T, did not match expected type", tc.status, err)
		}

		// The base is discoverable, carrying status, message, and quota.
		var de *DemografixError
		if !errors.As(err, &de) {
			t.Fatalf("status %d: error not discoverable as *DemografixError", tc.status)
		}
		if de.Status != tc.status {
			t.Errorf("status %d: de.Status = %d", tc.status, de.Status)
		}
		if de.Message != tc.message {
			t.Errorf("status %d: de.Message = %q, want %q", tc.status, de.Message, tc.message)
		}
		if de.Quota == nil {
			t.Fatalf("status %d: quota must be attached on errors", tc.status)
		}
		if de.Quota.Remaining != 24987 {
			t.Errorf("status %d: quota.remaining = %d, want 24987", tc.status, de.Quota.Remaining)
		}
		if err.Error() != tc.message {
			t.Errorf("status %d: Error() = %q, want %q", tc.status, err.Error(), tc.message)
		}
	}
}

func TestRateLimitErrorCarriesReset(t *testing.T) {
	var req *http.Request
	c := captureClient(429, fixtureHeaders(), `{ "error": "Request limit reached" }`, &req)

	_, err := c.Agify(context.Background(), "peter")
	var rl *RateLimitError
	if !errors.As(err, &rl) {
		t.Fatalf("error type = %T, want *RateLimitError", err)
	}
	if rl.Quota == nil {
		t.Fatal("rate-limit error must carry quota")
	}
	if rl.Quota.Reset != 1314000 {
		t.Errorf("quota.reset = %d, want 1314000", rl.Quota.Reset)
	}
}

// --- Transport-level failures map to TransportError ---

func TestNetworkFailureIsTransportError(t *testing.T) {
	called := false
	c := failClient(&called)

	_, err := c.Genderize(context.Background(), "peter")
	var te *TransportError
	if !errors.As(err, &te) {
		t.Fatalf("error type = %T, want *TransportError", err)
	}
	if !called {
		t.Error("a successful single call must reach the transport")
	}
}

func TestNonJSONBodyIsTransportError(t *testing.T) {
	var req *http.Request
	c := captureClient(http.StatusOK, fixtureHeaders(), `<html>not json</html>`, &req)

	_, err := c.Genderize(context.Background(), "peter")
	var te *TransportError
	if !errors.As(err, &te) {
		t.Fatalf("error type = %T, want *TransportError", err)
	}
}

// errSentinel is a transport failure used to assert the underlying cause is
// reachable through the TransportError.
var errSentinel = errors.New("dial tcp: connection refused")

// TestTransportErrorWrapsCause confirms a transport failure carries the underlying
// error: errors.Is reaches the wrapped cause and errors.As still finds both the
// concrete *TransportError and the embedded *DemografixError.
func TestTransportErrorWrapsCause(t *testing.T) {
	c := New("test-key")
	c.http.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errSentinel
	})

	_, err := c.Genderize(context.Background(), "peter")
	if !errors.Is(err, errSentinel) {
		t.Errorf("errors.Is did not reach the wrapped cause: %v", err)
	}
	var te *TransportError
	if !errors.As(err, &te) {
		t.Fatalf("error type = %T, want *TransportError", err)
	}
	if !errors.Is(te.Err, errSentinel) {
		t.Errorf("te.Err = %v, want the wrapped sentinel", te.Err)
	}
	var de *DemografixError
	if !errors.As(err, &de) {
		t.Error("transport error must still be discoverable as *DemografixError")
	}
}

// --- apikey is always sent (every client carries a key) ---

func TestAPIKeyAlwaysSent(t *testing.T) {
	var req *http.Request
	c := captureClient(http.StatusOK, fixtureHeaders(), genderizeSingle, &req)
	if _, err := c.Genderize(context.Background(), "peter"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.URL.Query().Get("apikey"); got != "test-key" {
		t.Errorf("apikey = %q, want test-key (always present on the request)", got)
	}
}

func TestAPIKeySentWhenSet(t *testing.T) {
	var req *http.Request
	c := New("secret")
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		req = r
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     fixtureHeaders(),
			Body:       io.NopCloser(strings.NewReader(genderizeSingle)),
		}, nil
	})
	if _, err := c.Genderize(context.Background(), "peter"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.URL.Query().Get("apikey"); got != "secret" {
		t.Errorf("apikey = %q, want secret", got)
	}
}

// --- Assertion 7: constructing without a usable api_key fails client-side, no HTTP ---

func TestMissingAPIKeyNoHTTP(t *testing.T) {
	// An empty key and a blank (whitespace-only) key both fail before any request.
	for _, key := range []string{"", "   "} {
		called := false
		c := New(key)
		c.http.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
			called = true
			return nil, errors.New("transport must not be reached")
		})

		_, err := c.Genderize(context.Background(), "peter")
		if err == nil {
			t.Fatalf("key %q: expected a ValidationError, got nil", key)
		}
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Fatalf("key %q: error type = %T, want *ValidationError", key, err)
		}
		if called {
			t.Errorf("key %q: a missing api_key must not send an HTTP request", key)
		}

		// The same guard fires on agify and nationalize, again with no HTTP call.
		called = false
		if _, err := c.Agify(context.Background(), "peter"); !errors.As(err, &ve) {
			t.Errorf("key %q: agify error = %T, want *ValidationError", key, err)
		}
		called = false
		if _, err := c.Nationalize(context.Background(), "peter"); !errors.As(err, &ve) {
			t.Errorf("key %q: nationalize error = %T, want *ValidationError", key, err)
		}
		if called {
			t.Errorf("key %q: a missing api_key must not send an HTTP request", key)
		}
	}
}

// Compile-time guard documenting the transport seam: roundTripFunc satisfies
// http.RoundTripper, the interface the Client's *http.Client.Transport accepts.
var _ http.RoundTripper = roundTripFunc(nil)
