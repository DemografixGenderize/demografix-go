package demografix

// Quota reports the rate-limit state carried on every response, parsed from the
// x-rate-limit-* headers. It is read from a returned value or a raised error and
// is never cached on the Client.
type Quota struct {
	// Limit is the number of names allowed in the current window.
	Limit int
	// Remaining is the number of names left in the current window.
	Remaining int
	// Reset is the number of seconds until the window resets.
	Reset int
}

// GenderizePrediction is one genderize result for a single name. Gender is "male",
// "female", or "" when the API returns null. CountryID is populated only when the
// request sent a country_id.
type GenderizePrediction struct {
	Name        string  `json:"name"`
	Gender      string  `json:"gender"`
	Probability float64 `json:"probability"`
	Count       int     `json:"count"`
	CountryID   string  `json:"country_id"`
}

// AgifyPrediction is one agify result for a single name. Age is nil when the API
// returns null. CountryID is populated only when the request sent a country_id.
type AgifyPrediction struct {
	Name      string `json:"name"`
	Age       *int   `json:"age"`
	Count     int    `json:"count"`
	CountryID string `json:"country_id"`
}

// NationalizeCountry is one candidate country for a nationalize prediction.
type NationalizeCountry struct {
	CountryID   string  `json:"country_id"`
	Probability float64 `json:"probability"`
}

// NationalizePrediction is one nationalize result for a single name. Country holds
// up to five candidates in descending probability and is empty on no match.
type NationalizePrediction struct {
	Name    string               `json:"name"`
	Country []NationalizeCountry `json:"country"`
	Count   int                  `json:"count"`
}

// GenderizeResult is a single genderize prediction plus the response quota.
type GenderizeResult struct {
	GenderizePrediction
	Quota Quota
}

// AgifyResult is a single agify prediction plus the response quota.
type AgifyResult struct {
	AgifyPrediction
	Quota Quota
}

// NationalizeResult is a single nationalize prediction plus the response quota.
type NationalizeResult struct {
	NationalizePrediction
	Quota Quota
}

// GenderizeBatchResult holds the per-name genderize predictions in input order
// plus one quota for the whole response.
type GenderizeBatchResult struct {
	Results []GenderizePrediction
	Quota   Quota
}

// AgifyBatchResult holds the per-name agify predictions in input order plus one
// quota for the whole response.
type AgifyBatchResult struct {
	Results []AgifyPrediction
	Quota   Quota
}

// NationalizeBatchResult holds the per-name nationalize predictions in input order
// plus one quota for the whole response.
type NationalizeBatchResult struct {
	Results []NationalizePrediction
	Quota   Quota
}
