# demografix-go

Run demographic analysis over names — predicted gender, age, and nationality — from one Go client. The package covers genderize.io, agify.io, and nationalize.io.

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

`New` takes the API key as its first argument. The base URLs and the User-Agent are hardcoded constants, not options. Quota is read from a returned value or a raised error, never cached on the client.

## genderize

Single name and batch. Both return prediction fields plus a quota.

```go
g, err := client.Genderize(ctx, "peter")
// g.Name, g.Gender ("male"/"female"/""), g.Probability, g.Count, g.Quota.Remaining

batch, err := client.GenderizeBatch(ctx, []string{"peter", "lois", "michael"})
counts := map[string]int{}
for _, p := range batch.Results {
	counts[p.Gender]++ // aggregate into a gender split
}
```

`Gender` is the empty string when the API returns null. The batch maximum is ten names; a larger batch raises a `ValidationError` before any HTTP call.

## agify

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

## country_id

`Genderize` and `Agify` accept `WithCountry` to scope a prediction to an ISO 3166-1 alpha-2 country. The value is echoed back uppercase on each prediction as `CountryID`.

```go
g, err := client.Genderize(ctx, "kim", demografix.WithCountry("US"))
// g.CountryID == "US"

batch, err := client.AgifyBatch(ctx, names, demografix.WithCountry("US"))
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

Methods return `(T, error)`. Non-2xx responses map by status code to a typed error; transport failures map to `TransportError`. Discover a type with `errors.As`.

| Type | Cause |
|---|---|
| `AuthError` | 401, invalid or rejected API key |
| `SubscriptionError` | 402, inactive or expired subscription |
| `ValidationError` | 422, or a batch over ten names (client-side, no HTTP call) |
| `RateLimitError` | 429, quota exhausted |
| `DemografixError` | base type for any other non-2xx response |
| `TransportError` | network failure, timeout, or non-JSON body |

Each type embeds `DemografixError`, which carries `Status`, `Message`, and `*Quota`. `errors.As` matches both the concrete type and the base.

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

| Method | Returns |
|---|---|
| `Genderize(ctx, name, ...Option)` | `GenderizeResult` |
| `GenderizeBatch(ctx, names, ...Option)` | `GenderizeBatchResult` |
| `Agify(ctx, name, ...Option)` | `AgifyResult` |
| `AgifyBatch(ctx, names, ...Option)` | `AgifyBatchResult` |
| `Nationalize(ctx, name)` | `NationalizeResult` |
| `NationalizeBatch(ctx, names)` | `NationalizeBatchResult` |

`Option` is `WithCountry("US")`, accepted by the genderize and agify methods only. A single result embeds its prediction fields and adds `Quota`. A batch result holds `Results` plus one `Quota`.

## API keys

An API key is required. Creating one is free and includes 2,500 requests per month. Generate a key in your dashboard at genderize.io, agify.io, or nationalize.io. One key works across all three services. Full reference: https://genderize.io/documentation/api
