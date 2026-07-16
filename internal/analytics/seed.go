package analytics

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/aloisdeniel/moth/internal/store"
)

// SeedOptions tunes Seed. The zero value seeds 90 days with seed 1.
type SeedOptions struct {
	// Days of history to generate, ending yesterday (project-local).
	Days int
	// Seed makes the generated stream deterministic.
	Seed uint64
	// Now anchors "yesterday"; nil means time.Now.
	Now func() time.Time
}

// Seed inserts a deterministic, realistic-looking synthetic event history
// for one project: a slowly growing user base with a weekly usage cycle,
// mixed providers (password/Google/Apple) and platforms, sampled token
// refreshes, a base rate of login failures and one elevated-failure
// incident day. It writes raw events only — run a Rollup afterwards to
// materialize daily_stats. It returns how many events were inserted.
//
// The synthetic user ids ("seed-user-N") reference no users row, which the
// schema allows; the data is for dashboards and tests, not for auth.
func Seed(ctx context.Context, st store.EventStore, p store.Project, o SeedOptions) (int, error) {
	if o.Days <= 0 {
		o.Days = 90
	}
	if o.Seed == 0 {
		o.Seed = 1
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	rng := rand.New(rand.NewPCG(o.Seed, o.Seed))
	loc := p.Settings.RollupLocation()
	now := o.Now().In(loc)

	providers := weighted{
		{store.IdentityProviderPassword, 55},
		{store.IdentityProviderGoogle, 30},
		{store.IdentityProviderApple, 15},
	}
	platforms := weighted{
		{store.PlatformIOS, 45},
		{store.PlatformAndroid, 35},
		{store.PlatformWeb, 12},
		{"", 8}, // SDK too old to report one
	}
	incident := o.Days / 3 // one bad day (expired provider key, say)

	var (
		users   []string
		batch   []store.Event
		total   int
		eventID int
	)
	newEvent := func(day time.Time, typ, userID, provider string) store.Event {
		eventID++
		at := day.Add(time.Duration(rng.Int64N(int64(24 * time.Hour))))
		return store.Event{
			ID:         fmt.Sprintf("seed-%s-%06d", p.ID, eventID),
			ProjectID:  p.ID,
			UserID:     userID,
			Type:       typ,
			Provider:   provider,
			Platform:   platforms.pick(rng),
			SDKVersion: "1.0.0",
			CreatedAt:  at.UTC(),
		}
	}

	for d := 0; d < o.Days; d++ {
		age := o.Days - 1 - d // days ago, so volumes grow toward today
		day := time.Date(now.Year(), now.Month(), now.Day()-1-age, 0, 0, 0, 0, loc)
		// Growth curve with a weekend dip and per-day noise.
		activity := 1.0 + 2.0*float64(d)/float64(o.Days)
		if wd := day.Weekday(); wd == time.Saturday || wd == time.Sunday {
			activity *= 0.6
		}
		activity *= 0.8 + 0.4*rng.Float64()

		// Signups create users; the pool feeds the login volume.
		signups := int(float64(2+rng.IntN(4)) * activity)
		for i := 0; i < signups; i++ {
			id := fmt.Sprintf("seed-user-%d", len(users)+1)
			users = append(users, id)
			batch = append(batch, newEvent(day, store.EventUserSignup, id, providers.pick(rng)))
		}

		// Logins from a random slice of the existing pool.
		logins := min(len(users), int(float64(5+rng.IntN(10))*activity))
		for i := 0; i < logins; i++ {
			user := users[rng.IntN(len(users))]
			batch = append(batch, newEvent(day, store.EventUserLogin, user, providers.pick(rng)))
			// Some sessions refresh later the same day (post-sampling rate).
			if rng.Float64() < 0.3 {
				batch = append(batch, newEvent(day, store.EventTokenRefresh, user, ""))
			}
		}

		// Failures: a small base rate, elevated tenfold on the incident day.
		failures := 1 + logins/20 + rng.IntN(2)
		if d == incident {
			failures = 5 + logins/2
		}
		for i := 0; i < failures; i++ {
			e := newEvent(day, store.EventUserLoginFailed, "", providers.pick(rng))
			e.Metadata = `{"reason":"invalid_credentials"}`
			batch = append(batch, e)
		}

		if len(batch) >= 500 || d == o.Days-1 {
			if err := st.InsertEvents(ctx, batch); err != nil {
				return total, fmt.Errorf("seed events: %w", err)
			}
			total += len(batch)
			batch = batch[:0]
		}
	}
	return total, nil
}

// weighted is a tiny weighted picker for provider/platform distributions.
type weighted []struct {
	value  string
	weight int
}

func (w weighted) pick(rng *rand.Rand) string {
	sum := 0
	for _, c := range w {
		sum += c.weight
	}
	n := rng.IntN(sum)
	for _, c := range w {
		if n < c.weight {
			return c.value
		}
		n -= c.weight
	}
	return w[len(w)-1].value
}
