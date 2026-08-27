# demografix-go

Predict gender, age, and nationality from names. One Go client covers all three Demografix
APIs — [genderize.io](https://genderize.io) (gender), [agify.io](https://agify.io) (age), and
[nationalize.io](https://nationalize.io) (nationality) — with single-name lookups and batches of up
to 100 names per request.

[![Go Reference](https://pkg.go.dev/badge/github.com/DemografixGenderize/demografix-go.svg)](https://pkg.go.dev/github.com/DemografixGenderize/demografix-go)
[![CI](https://github.com/DemografixGenderize/demografix-go/actions/workflows/ci.yml/badge.svg)](https://github.com/DemografixGenderize/demografix-go/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

## Install

```sh
go get github.com/DemografixGenderize/demografix-go
```

Requires Go 1.21 or later. The package has no third-party dependencies.

## Quickstart

Construct a client, run a batch over a list of names, read the predictions, and read the remaining quota.

```go
package main

import (
	"context"
	"fmt"

	demografix "github.com/DemografixGenderize/demografix-go"
)

func main() {
	client := demografix.New("YOUR_API_KEY")

	names := []string{"peter", "lois", "michael", "matthew"}

	res, err := client.GenderizeBatch(context.Background(), names)
	if err != nil {
		panic(err)
	}

	split := map[string]int{}
	for _, p := range res.Results {
		split[p.Gender]++
	}

	fmt.Println(split)               // gender split of the list
	fmt.Println(res.Quota.Remaining) // 24987
}
```

`New` takes the API key as its first argument. The base URLs and the User-Agent are hardcoded
constants, not options. Quota is read from a returned value or a raised error, never cached on the
client.

## genderize

Predict gender from names. Single name and batch both return prediction fields plus a quota.

```go
g, err := client.Genderize(ctx, "peter")
// g.Name, g.Gender ("male"/"female"/""), g.Probability, g.Count, g.Quota.Remaining

batch, err := client.GenderizeBatch(ctx, []string{"peter", "lois", "michael"})
counts := map[string]int{}
for _, p := range batch.Results {
	counts[p.Gender]++ // aggregate into a gender split
}
```

`Gender` is the empty string when the API returns null. That is a successful response, not an error.

## agify

Predict age from names.

```go
a, err := client.Agify(ctx, "michael")
// a.Age (*int, nil when null), a.Count, a.Quota.Remaining

batch, err := client.AgifyBatch(ctx, []string{"michael", "matthew", "jane"})
buckets := map[int]int{}
for _, p := range batch.Results {
	if p.Age != nil {
		buckets[(*p.Age/10)*10]++ // aggregate into an age distribution
	}
}
```

`Age` is a `*int` and is nil when the API returns null.

## nationalize

Predict nationality from names.

```go
n, err := client.Nationalize(ctx, "nguyen")
// n.Country is up to five {CountryID, Probability} candidates, descending probability

batch, err := client.NationalizeBatch(ctx, []string{"nguyen", "schmidt", "rossi"})
mix := map[string]int{}
for _, p := range batch.Results {
	if len(p.Country) > 0 {
		mix[p.Country[0].CountryID]++ // aggregate into a nationality mix
	}
}
```

`Country` is empty on no match. The nationalize methods do not accept a country option.

## Batch limit

Each batch accepts at most 100 names. A larger batch returns a `ValidationError` before any HTTP
call. Chunk a longer list and aggregate across the chunks.

```go
split := map[string]int{}
for start := 0; start < len(roster); start += 100 {
	end := min(start+100, len(roster))

	batch, err := client.GenderizeBatch(ctx, roster[start:end])
	if err != nil {
		return err
	}
	for _, p := range batch.Results {
		split[p.Gender]++
	}
}
```

## country_id

`Genderize` and `Agify` accept `WithCountry` to scope a prediction to an ISO 3166-1 alpha-2 country.
The value is echoed back uppercase on each prediction as `CountryID`.

```go
g, err := client.Genderize(ctx, "kim", demografix.WithCountry("US"))
// g.CountryID == "US"

batch, err := client.AgifyBatch(ctx, names, demografix.WithCountry("US"))
```

Scoping changes the prediction: `andrea` reads female with probability 0.99 in the United States and
male with probability 0.79 in Italy.

```go
us, _ := client.Genderize(ctx, "andrea", demografix.WithCountry("US")) // us.Gender == "female"
it, _ := client.Genderize(ctx, "andrea", demografix.WithCountry("IT")) // it.Gender == "male"
```

The nationalize methods do not take `WithCountry`.

## Quota

Every result and every typed error carries a `Quota` read from the response headers.

| Field | Meaning |
|---|---|
| `Limit` | names allowed in the current window |
| `Remaining` | names left in the current window |
| `Reset` | seconds until the window resets |

```go
res, _ := client.GenderizeBatch(ctx, names)
fmt.Println(res.Quota.Remaining)
```

## Errors

Methods return `(T, error)`. Non-2xx responses map by status code to a typed error; transport
failures map to `TransportError`. Discover a type with `errors.As`.

| Type | Cause |
|---|---|
| `AuthError` | 401, invalid or rejected API key |
| `SubscriptionError` | 402, inactive or expired subscription |
| `ValidationError` | 422, or a batch over 100 names (client-side, no HTTP call) |
| `RateLimitError` | 429, quota exhausted |
| `DemografixError` | base type for any other non-2xx response |
| `TransportError` | network failure, timeout, or non-JSON body |

Each type embeds `DemografixError`, which carries `Status`, `Message`, and `*Quota`. `errors.As`
matches both the concrete type and the base.

A `TransportError` wraps the underlying cause in its `Err` field, so `errors.Is` reaches it:

```go
_, err := client.Genderize(ctx, "peter")
if errors.Is(err, context.DeadlineExceeded) {
	// the request timed out
}
```

A `RateLimitError` always carries quota. Read `Quota.Reset` to back off before retrying.

```go
res, err := client.GenderizeBatch(ctx, names)
if err != nil {
	var rate *demografix.RateLimitError
	if errors.As(err, &rate) {
		time.Sleep(time.Duration(rate.Quota.Reset) * time.Second)
		res, err = client.GenderizeBatch(ctx, names)
	}
}
```

## Methods

| Method | Returns | country_id |
|---|---|---|
| `Genderize(ctx, name, ...RequestOption)` | `GenderizeResult` | yes |
| `GenderizeBatch(ctx, names, ...RequestOption)` | `GenderizeBatchResult` | yes |
| `Agify(ctx, name, ...RequestOption)` | `AgifyResult` | yes |
| `AgifyBatch(ctx, names, ...RequestOption)` | `AgifyBatchResult` | yes |
| `Nationalize(ctx, name)` | `NationalizeResult` | no |
| `NationalizeBatch(ctx, names)` | `NationalizeBatchResult` | no |

There are two option types. `RequestOption` is a per-call option: the only one is
`WithCountry("US")`, accepted by the genderize and agify methods. `Option` is a client option passed
to `New`: the only one is `WithTimeout(d)`, which defaults to ten seconds. A single result embeds its
prediction fields and adds `Quota`. A batch result holds `Results` plus one `Quota`.

## API keys

An API key is required. Creating one is free and includes 2,500 names per month.

Quota counts **names, not requests**. A single-name call costs 1. A batch of 100 names costs 100. The
free tier therefore covers 2,500 names in a month however they are split across calls.

Generate a key in your dashboard at [genderize.io](https://genderize.io),
[agify.io](https://agify.io), or [nationalize.io](https://nationalize.io). One key works across all
three services. Full reference:
[genderize.io/documentation/api](https://genderize.io/documentation/api).

## License

MIT. See [LICENSE](LICENSE).
