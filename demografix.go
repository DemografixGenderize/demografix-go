// Package demografix is the official Go client for the Demografix APIs:
// genderize.io (gender), agify.io (age), and nationalize.io (nationality). One
// client covers all three services through the same shape and reports the
// remaining quota carried on every response.
package demografix

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// Version is the SDK version, sent in the User-Agent on every request.
	Version = "0.1.0"

	userAgent = "demografix-go/" + Version

	genderizeBaseURL   = "https://api.genderize.io/"
	agifyBaseURL       = "https://api.agify.io/"
	nationalizeBaseURL = "https://api.nationalize.io/"

	defaultTimeout = 10 * time.Second

	// maxBatch is the largest number of names allowed in a single request.
	maxBatch = 10
)

// Client calls the three Demografix services. Construct one with New and reuse it;
// it is safe for concurrent use. The hosts and the User-Agent are hardcoded
// constants, not options.
type Client struct {
	apiKey string
	http   *http.Client
}

// Option configures a Client in New.
type Option func(*Client)

// WithAPIKey sets the API key sent as the apikey query parameter on every request.
// Without a key, requests go out on the free per-IP tier.
func WithAPIKey(apiKey string) Option {
	return func(c *Client) { c.apiKey = apiKey }
}

// WithTimeout sets the per-request timeout. The default is ten seconds.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) { c.http.Timeout = timeout }
}

// New builds a Client. Pass WithAPIKey and WithTimeout to configure it.
func New(opts ...Option) *Client {
	c := &Client{
		http: &http.Client{Timeout: defaultTimeout},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// RequestOption configures a single genderize or agify request. The only option is
// WithCountry.
type RequestOption func(*requestConfig)

type requestConfig struct {
	countryID string
}

// WithCountry scopes a genderize or agify prediction to an ISO 3166-1 alpha-2
// country. The nationalize methods do not accept it.
func WithCountry(countryID string) RequestOption {
	return func(rc *requestConfig) { rc.countryID = countryID }
}

func newRequestConfig(opts []RequestOption) requestConfig {
	var rc requestConfig
	for _, opt := range opts {
		opt(&rc)
	}
	return rc
}

// Genderize predicts the gender for one name.
func (c *Client) Genderize(ctx context.Context, name string, opts ...RequestOption) (GenderizeResult, error) {
	rc := newRequestConfig(opts)
	var pred GenderizePrediction
	quota, err := c.do(ctx, genderizeBaseURL, []string{name}, rc.countryID, &pred)
	if err != nil {
		return GenderizeResult{}, err
	}
	return GenderizeResult{GenderizePrediction: pred, Quota: quota}, nil
}

// GenderizeBatch predicts the gender for up to ten names. Results are returned in
// input order. A batch of more than ten names raises a ValidationError before any
// HTTP call.
func (c *Client) GenderizeBatch(ctx context.Context, names []string, opts ...RequestOption) (GenderizeBatchResult, error) {
	if err := checkBatch(names); err != nil {
		return GenderizeBatchResult{}, err
	}
	rc := newRequestConfig(opts)
	var preds []GenderizePrediction
	quota, err := c.do(ctx, genderizeBaseURL, names, rc.countryID, &preds)
	if err != nil {
		return GenderizeBatchResult{}, err
	}
	return GenderizeBatchResult{Results: preds, Quota: quota}, nil
}

// Agify predicts the age for one name.
func (c *Client) Agify(ctx context.Context, name string, opts ...RequestOption) (AgifyResult, error) {
	rc := newRequestConfig(opts)
	var pred AgifyPrediction
	quota, err := c.do(ctx, agifyBaseURL, []string{name}, rc.countryID, &pred)
	if err != nil {
		return AgifyResult{}, err
	}
	return AgifyResult{AgifyPrediction: pred, Quota: quota}, nil
}

// AgifyBatch predicts the age for up to ten names. Results are returned in input
// order. A batch of more than ten names raises a ValidationError before any HTTP
// call.
func (c *Client) AgifyBatch(ctx context.Context, names []string, opts ...RequestOption) (AgifyBatchResult, error) {
	if err := checkBatch(names); err != nil {
		return AgifyBatchResult{}, err
	}
	rc := newRequestConfig(opts)
	var preds []AgifyPrediction
	quota, err := c.do(ctx, agifyBaseURL, names, rc.countryID, &preds)
	if err != nil {
		return AgifyBatchResult{}, err
	}
	return AgifyBatchResult{Results: preds, Quota: quota}, nil
}

// Nationalize predicts the nationality for one name.
func (c *Client) Nationalize(ctx context.Context, name string) (NationalizeResult, error) {
	var pred NationalizePrediction
	quota, err := c.do(ctx, nationalizeBaseURL, []string{name}, "", &pred)
	if err != nil {
		return NationalizeResult{}, err
	}
	return NationalizeResult{NationalizePrediction: pred, Quota: quota}, nil
}

// NationalizeBatch predicts the nationality for up to ten names. Results are
// returned in input order. A batch of more than ten names raises a
// ValidationError before any HTTP call.
func (c *Client) NationalizeBatch(ctx context.Context, names []string) (NationalizeBatchResult, error) {
	if err := checkBatch(names); err != nil {
		return NationalizeBatchResult{}, err
	}
	var preds []NationalizePrediction
	quota, err := c.do(ctx, nationalizeBaseURL, names, "", &preds)
	if err != nil {
		return NationalizeBatchResult{}, err
	}
	return NationalizeBatchResult{Results: preds, Quota: quota}, nil
}

// checkBatch enforces the ten-name limit client-side.
func checkBatch(names []string) error {
	if len(names) > maxBatch {
		return newValidationError("batch holds more than 10 names")
	}
	return nil
}

// do builds and sends one request, parses the quota headers, and decodes the body
// into out. When names holds one element the query uses name=v; otherwise it uses
// repeated name[]=v. The country_id and apikey parameters are added only when set.
func (c *Client) do(ctx context.Context, baseURL string, names []string, countryID string, out any) (Quota, error) {
	req, err := c.buildRequest(ctx, baseURL, names, countryID)
	if err != nil {
		return Quota{}, wrapTransportError(err.Error(), err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return Quota{}, wrapTransportError(err.Error(), err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Quota{}, wrapTransportError(err.Error(), err)
	}

	quota := parseQuota(resp.Header)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return quota, decodeError(resp.StatusCode, body, quota)
	}

	if err := json.Unmarshal(body, out); err != nil {
		return quota, wrapTransportError("response body is not valid JSON: "+err.Error(), err)
	}
	return quota, nil
}

// buildRequest assembles the GET request with the query string and headers.
func (c *Client) buildRequest(ctx context.Context, baseURL string, names []string, countryID string) (*http.Request, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	q := url.Values{}
	if len(names) == 1 {
		q.Set("name", names[0])
	} else {
		for _, n := range names {
			q.Add("name[]", n)
		}
	}
	if countryID != "" {
		q.Set("country_id", countryID)
	}
	if c.apiKey != "" {
		q.Set("apikey", c.apiKey)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	return req, nil
}

// errorBody is the shape of a non-2xx response body.
type errorBody struct {
	Error string `json:"error"`
}

// decodeError parses the error message and maps the status to a typed error. A
// body that is not JSON becomes a TransportError.
func decodeError(status int, body []byte, quota Quota) error {
	var eb errorBody
	if err := json.Unmarshal(body, &eb); err != nil {
		return wrapTransportError("error response body is not valid JSON: "+err.Error(), err)
	}
	return newAPIError(status, eb.Error, &quota)
}

// parseQuota reads the three rate-limit headers case-insensitively. Header lookup
// in net/http is already case-insensitive via canonicalization; missing or
// non-integer values become zero.
func parseQuota(h http.Header) Quota {
	return Quota{
		Limit:     headerInt(h, "X-Rate-Limit-Limit"),
		Remaining: headerInt(h, "X-Rate-Limit-Remaining"),
		Reset:     headerInt(h, "X-Rate-Limit-Reset"),
	}
}

func headerInt(h http.Header, key string) int {
	v := strings.TrimSpace(h.Get(key))
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return n
}

// AsDemografixError reports whether err is a Demografix SDK error and, if so,
// returns the embedded base. It is a convenience over errors.As for callers that
// want the status, message, and quota without naming a concrete type.
func AsDemografixError(err error) (*DemografixError, bool) {
	var de *DemografixError
	if errors.As(err, &de) {
		return de, true
	}
	return nil, false
}
