// Command example reads a list of names and reports the aggregate demographic mix:
// a gender split, an age distribution, and a nationality breakdown.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	demografix "github.com/DemografixGenderize/demografix-go"
)

func main() {
	// An API key is required. Generate one in your dashboard at genderize.io,
	// agify.io, or nationalize.io.
	client := demografix.New(
		os.Getenv("DEMOGRAFIX_API_KEY"),
		demografix.WithTimeout(10*time.Second),
	)

	names := []string{"peter", "lois", "michael", "matthew", "nguyen", "kim"}

	ctx := context.Background()

	genderSplit(ctx, client, names)
	ageDistribution(ctx, client, names)
	nationalityMix(ctx, client, names)
}

// genderSplit reports how many names in the list resolve to each gender.
func genderSplit(ctx context.Context, client *demografix.Client, names []string) {
	res, err := client.GenderizeBatch(ctx, names)
	if err != nil {
		report("genderize", err)
		return
	}

	counts := map[string]int{}
	for _, p := range res.Results {
		label := p.Gender
		if label == "" {
			label = "unknown"
		}
		counts[label]++
	}

	fmt.Println("Gender split:")
	for _, g := range []string{"male", "female", "unknown"} {
		fmt.Printf("  %-8s %d\n", g, counts[g])
	}
	fmt.Printf("  quota remaining: %d\n\n", res.Quota.Remaining)
}

// ageDistribution buckets the predicted ages of the list into decades.
func ageDistribution(ctx context.Context, client *demografix.Client, names []string) {
	res, err := client.AgifyBatch(ctx, names)
	if err != nil {
		report("agify", err)
		return
	}

	buckets := map[int]int{}
	for _, p := range res.Results {
		if p.Age == nil {
			continue
		}
		buckets[(*p.Age/10)*10]++
	}

	decades := make([]int, 0, len(buckets))
	for d := range buckets {
		decades = append(decades, d)
	}
	sort.Ints(decades)

	fmt.Println("Age distribution:")
	for _, d := range decades {
		fmt.Printf("  %d-%d  %d\n", d, d+9, buckets[d])
	}
	fmt.Printf("  quota remaining: %d\n\n", res.Quota.Remaining)
}

// nationalityMix tallies the top predicted country for each name in the list.
func nationalityMix(ctx context.Context, client *demografix.Client, names []string) {
	res, err := client.NationalizeBatch(ctx, names)
	if err != nil {
		report("nationalize", err)
		return
	}

	counts := map[string]int{}
	for _, p := range res.Results {
		if len(p.Country) == 0 {
			counts["unknown"]++
			continue
		}
		counts[p.Country[0].CountryID]++
	}

	fmt.Println("Nationality mix (top country per name):")
	for country, n := range counts {
		fmt.Printf("  %-8s %d\n", country, n)
	}
	fmt.Printf("  quota remaining: %d\n", res.Quota.Remaining)
}

// report demonstrates the typed error hierarchy. A RateLimitError carries the reset
// delay; other typed errors carry status and message.
func report(api string, err error) {
	var rate *demografix.RateLimitError
	if errors.As(err, &rate) {
		fmt.Printf("%s: rate limited, retry in %d seconds\n", api, rate.Quota.Reset)
		return
	}

	var de *demografix.DemografixError
	if errors.As(err, &de) {
		fmt.Printf("%s: error %d: %s\n", api, de.Status, de.Message)
		return
	}

	fmt.Printf("%s: %v\n", api, err)
}
