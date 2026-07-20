// Package entitlements holds the pure entitlement-derivation engine: given a
// user's subscription rows and grants, it computes the set of entitlement
// identifiers the user currently holds. It is deliberately free of I/O, proto
// and store access so every cell of the status x grant matrix can be unit
// tested exhaustively.
//
// The store is the source of truth; moth mirrors it and this engine derives
// entitlements from the mirror. The matrix (plan/11 §Model):
//
//	subscription status         -> access?
//	  active                    -> GRANTED
//	  trialing                  -> GRANTED
//	  in_grace_period           -> GRANTED  (store keeps access during a card hiccup)
//	  in_billing_retry          -> GRANTED  (Google "on hold" / Apple billing retry)
//	  paused                    -> not granted
//	  expired                   -> not granted
//	  revoked                   -> not granted
//	operator grant, not revoked and not past its expiry -> GRANTED until expiry
//	no subscription and no grant                        -> `none`, the free
//	                                                       default (empty set,
//	                                                       always valid, never
//	                                                       an error)
package entitlements

import (
	"sort"
	"time"

	"github.com/aloisdeniel/moth/internal/store"
)

// Entitlement source values (also the wire strings for moth.server.v1).
const (
	SourceStore = "store"
	SourceGrant = "grant"
	SourceNone  = "none"
)

// Entitlement is one derived, currently-held capability.
type Entitlement struct {
	// Identifier is the stable capability the app gates on (e.g. "pro").
	Identifier string
	// ExpireTime is when it lapses; zero means indefinite (an unbounded grant).
	ExpireTime time.Time
	// Source is SourceStore (a store subscription) or SourceGrant (an operator
	// grant).
	Source string
	// ProductIdentifier is the moth product that granted it when Source is
	// SourceStore; empty for grants.
	ProductIdentifier string
	// IsSandbox reports that this entitlement is backed by a store sandbox /
	// license-tester subscription (never a production purchase). A developer
	// backend gating paid server-side features must be able to tell a free
	// tester grant from a real one; grants are never sandbox.
	IsSandbox bool
}

// StatusGrantsAccess reports whether a subscription in the given status entitles
// the user. active/trialing/in_grace_period/in_billing_retry keep access;
// paused/expired/revoked (and anything unknown) do not.
func StatusGrantsAccess(status string) bool {
	switch status {
	case store.SubscriptionStatusActive,
		store.SubscriptionStatusTrialing,
		store.SubscriptionStatusInGracePeriod,
		store.SubscriptionStatusInBillingRetry:
		return true
	default:
		return false
	}
}

// Derive computes the entitlement set a user holds at now. It unions the
// entitlements granted by access-granting subscriptions (resolved through the
// product -> entitlements mapping) with the entitlements of active operator
// grants, deduplicating by identifier and keeping, per identifier, the grant
// with the most generous expiry (an unbounded expiry beats any dated one, and a
// later date beats an earlier one). The result is sorted by identifier for
// determinism. An empty result is the free `none` state.
//
// grants may include revoked and expired rows; Derive filters them itself so
// callers can pass a user's full grant history and the matrix stays testable
// here.
func Derive(now time.Time, ents []store.Entitlement, products []store.Product, subs []store.Subscription, grants []store.SubscriptionGrant) []Entitlement {
	entByID := make(map[string]store.Entitlement, len(ents))
	for _, e := range ents {
		entByID[e.ID] = e
	}
	prodByID := make(map[string]store.Product, len(products))
	for _, p := range products {
		prodByID[p.ID] = p
	}

	held := make(map[string]Entitlement)
	add := func(cand Entitlement) {
		if cand.Identifier == "" {
			return
		}
		if existing, ok := held[cand.Identifier]; ok && !moreGenerous(cand.ExpireTime, existing.ExpireTime) {
			return
		}
		held[cand.Identifier] = cand
	}

	// Store subscriptions: an unmapped subscription (no moth product) grants
	// nothing — moth cannot know which entitlements the SKU is meant to unlock.
	for _, sub := range subs {
		if !StatusGrantsAccess(sub.Status) {
			continue
		}
		prod, ok := prodByID[sub.ProductID]
		if !ok {
			continue
		}
		var expire time.Time
		if sub.CurrentPeriodEnd != nil {
			expire = *sub.CurrentPeriodEnd
		}
		for _, eid := range prod.EntitlementIDs {
			ent, ok := entByID[eid]
			if !ok {
				continue
			}
			add(Entitlement{
				Identifier:        ent.Identifier,
				ExpireTime:        expire,
				Source:            SourceStore,
				ProductIdentifier: prod.Identifier,
				IsSandbox:         sub.Environment == store.SubscriptionEnvironmentSandbox,
			})
		}
	}

	// Operator grants: independent of store state, granted until expiry.
	for _, g := range grants {
		if g.RevokedAt != nil {
			continue
		}
		if g.ExpiresAt != nil && !g.ExpiresAt.After(now) {
			continue
		}
		ent, ok := entByID[g.EntitlementID]
		if !ok {
			continue
		}
		var expire time.Time
		if g.ExpiresAt != nil {
			expire = *g.ExpiresAt
		}
		add(Entitlement{Identifier: ent.Identifier, ExpireTime: expire, Source: SourceGrant})
	}

	out := make([]Entitlement, 0, len(held))
	for _, e := range held {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Identifier < out[j].Identifier })
	return out
}

// moreGenerous reports whether candidate expiry a is at least as good as b: an
// unbounded (zero) expiry beats any dated one, and a later date beats an
// earlier one.
func moreGenerous(a, b time.Time) bool {
	if a.IsZero() {
		return true
	}
	if b.IsZero() {
		return false
	}
	return !a.Before(b)
}
