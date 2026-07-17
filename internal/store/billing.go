package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Subscription stores (the `store` column).
const (
	SubscriptionStoreApple  = "apple"
	SubscriptionStoreGoogle = "google"
)

// Subscription environments (the `environment` column).
const (
	SubscriptionEnvironmentSandbox    = "sandbox"
	SubscriptionEnvironmentProduction = "production"
)

// Subscription statuses, mapped from both stores. The entitlement-derivation
// engine (a separate layer) maps these to granted/not-granted; the store only
// persists the string.
const (
	SubscriptionStatusActive         = "active"
	SubscriptionStatusTrialing       = "trialing"
	SubscriptionStatusInGracePeriod  = "in_grace_period"
	SubscriptionStatusInBillingRetry = "in_billing_retry"
	SubscriptionStatusPaused         = "paused"
	SubscriptionStatusExpired        = "expired"
	SubscriptionStatusRevoked        = "revoked"
)

// Subscription-event types written into subscription_events for milestone-14
// revenue analytics. Emitted by the milestone-11 engine on state changes.
const (
	SubscriptionEventPurchased    = "subscription.purchased"
	SubscriptionEventRenewed      = "subscription.renewed"
	SubscriptionEventTrialStarted = "subscription.trial_started"
	// SubscriptionEventConverted marks a trial converting to a paid
	// subscription. The milestone-14 rollup consumes it for trial-conversion
	// stats; wiring its emission from the billing engine (a trialing sub going
	// active) is a milestone-11 follow-up — until then trials_converted is 0.
	SubscriptionEventConverted = "subscription.converted"
	SubscriptionEventCanceled  = "subscription.canceled"
	SubscriptionEventExpired   = "subscription.expired"
	SubscriptionEventRefunded  = "subscription.refunded"
	SubscriptionEventGranted   = "subscription.granted"
	SubscriptionEventRevoked   = "subscription.revoked"
)

// Entitlement is a named capability a project's apps gate features on.
type Entitlement struct {
	ID          string
	ProjectID   string
	Identifier  string
	DisplayName string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Product is one subscription tier. EntitlementIDs are the entitlements the
// product grants while active (the product_entitlements join), loaded by the
// read methods and written by Create/UpdateProduct.
type Product struct {
	ID                     string
	ProjectID              string
	Identifier             string
	DisplayName            string
	AppleProductID         string
	GoogleProductID        string
	BillingPeriod          string
	PriceAmountMicros      int64
	Currency               string
	TrialPeriod            string
	IntroPriceAmountMicros int64
	IntroPeriod            string
	Offering               string
	SortOrder              int
	EntitlementIDs         []string
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// Subscription is moth's mirror of one store subscription.
type Subscription struct {
	ID                 string
	ProjectID          string
	UserID             string
	Store              string
	ProductID          string // "" when the SKU is unmapped
	StoreTransactionID string // apple original_transaction_id | google purchase_token
	SubscriptionID     string // google subscriptionId; "" for apple
	Status             string
	CurrentPeriodEnd   *time.Time
	AutoRenew          bool
	Environment        string
	RawState           string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// SubscriptionGrant is a manual/promotional entitlement grant.
type SubscriptionGrant struct {
	ID            string
	ProjectID     string
	UserID        string
	EntitlementID string
	ExpiresAt     *time.Time
	Reason        string
	GrantedBy     string
	CreatedAt     time.Time
	RevokedAt     *time.Time
}

// StoreNotification is a raw store notification kept for idempotency and audit.
type StoreNotification struct {
	ID             string
	ProjectID      string
	Store          string
	NotificationID string
	Type           string
	Subtype        string
	RawPayload     string
	ProcessedAt    *time.Time
	CreatedAt      time.Time
}

// BillingCredentials is a project's store API credentials. The *Enc fields are
// AES-GCM ciphertext under the master key (encryption in the handler layer):
// nil on upsert keeps the stored value, an empty slice clears it, non-empty
// sets it (write-only secret semantics).
type BillingCredentials struct {
	ProjectID                  string
	AppleIAPKeyID              string
	AppleIAPIssuerID           string
	AppleIAPKeyEnc             []byte
	AppleBundleID              string
	AppleAppAppleID            string
	AppleNotificationSecretEnc []byte
	// AppleNotificationURL is the App Store Server Notification URL moth has
	// registered (Apple has no read for it); "" means none. On upsert, "" keeps
	// the stored value (the CLI writes it only after a successful registration).
	AppleNotificationURL    string
	GoogleServiceAccountEnc []byte
	GooglePackageName       string
	GooglePubsubTopic       string
	GoogleRTDNSecretEnc     []byte
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// SubscriptionEvent is one row of the revenue event stream (milestone 14).
type SubscriptionEvent struct {
	ID                string
	ProjectID         string
	Type              string
	UserID            string // "" when no subject
	ProductID         string // "" when no product
	Store             string
	PriceAmountMicros int64
	Currency          string
	Environment       string
	CreatedAt         time.Time
}

// --- Entitlements ---------------------------------------------------------

func (s *Store) CreateEntitlement(ctx context.Context, e Entitlement) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO entitlements (id, project_id, identifier, display_name, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		e.ID, e.ProjectID, e.Identifier, e.DisplayName, formatTime(e.CreatedAt), formatTime(e.UpdatedAt))
	if err != nil {
		if ce := conflictErr(err); errors.Is(ce, ErrConflict) {
			return ce
		}
		return fmt.Errorf("create entitlement: %w", err)
	}
	return nil
}

func (s *Store) GetEntitlement(ctx context.Context, projectID, id string) (Entitlement, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, identifier, display_name, created_at, updated_at
		   FROM entitlements WHERE project_id = ? AND id = ?`, projectID, id)
	e, err := scanEntitlement(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Entitlement{}, ErrNotFound
	}
	return e, err
}

func (s *Store) GetEntitlementByIdentifier(ctx context.Context, projectID, identifier string) (Entitlement, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, identifier, display_name, created_at, updated_at
		   FROM entitlements WHERE project_id = ? AND identifier = ?`, projectID, identifier)
	e, err := scanEntitlement(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Entitlement{}, ErrNotFound
	}
	return e, err
}

func (s *Store) ListEntitlements(ctx context.Context, projectID string) ([]Entitlement, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, identifier, display_name, created_at, updated_at
		   FROM entitlements WHERE project_id = ? ORDER BY identifier`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list entitlements: %w", err)
	}
	defer rows.Close()
	var out []Entitlement
	for rows.Next() {
		e, err := scanEntitlement(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) UpdateEntitlement(ctx context.Context, e Entitlement) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE entitlements SET display_name = ?, updated_at = ?
		  WHERE project_id = ? AND id = ?`,
		e.DisplayName, formatTime(e.UpdatedAt), e.ProjectID, e.ID)
	if err != nil {
		return fmt.Errorf("update entitlement: %w", err)
	}
	return requireRow(res)
}

func (s *Store) DeleteEntitlement(ctx context.Context, projectID, id string) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM entitlements WHERE project_id = ? AND id = ?`, projectID, id)
	if err != nil {
		return fmt.Errorf("delete entitlement: %w", err)
	}
	return requireRow(res)
}

func scanEntitlement(row rowScanner) (Entitlement, error) {
	var e Entitlement
	var createdAt, updatedAt string
	if err := row.Scan(&e.ID, &e.ProjectID, &e.Identifier, &e.DisplayName, &createdAt, &updatedAt); err != nil {
		return Entitlement{}, err
	}
	var err error
	if e.CreatedAt, err = parseTime(createdAt); err != nil {
		return Entitlement{}, fmt.Errorf("parse entitlement created_at: %w", err)
	}
	if e.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return Entitlement{}, fmt.Errorf("parse entitlement updated_at: %w", err)
	}
	return e, nil
}

// --- Products -------------------------------------------------------------

// CreateProduct inserts the product and its entitlement grants in one
// transaction.
func (s *Store) CreateProduct(ctx context.Context, p Product) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("create product: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO products (id, project_id, identifier, display_name, apple_product_id,
		        google_product_id, billing_period, price_amount_micros, currency, trial_period,
		        intro_price_amount_micros, intro_period, offering, sort_order, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.ProjectID, p.Identifier, p.DisplayName, nullString(p.AppleProductID),
		nullString(p.GoogleProductID), p.BillingPeriod, p.PriceAmountMicros, p.Currency, p.TrialPeriod,
		p.IntroPriceAmountMicros, p.IntroPeriod, p.Offering, p.SortOrder,
		formatTime(p.CreatedAt), formatTime(p.UpdatedAt)); err != nil {
		if ce := conflictErr(err); errors.Is(ce, ErrConflict) {
			return ce
		}
		return fmt.Errorf("insert product: %w", err)
	}
	if err := replaceProductEntitlements(ctx, tx, p.ID, p.EntitlementIDs); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create product: %w", err)
	}
	return nil
}

// UpdateProduct updates the product row and replaces its entitlement grants in
// one transaction.
func (s *Store) UpdateProduct(ctx context.Context, p Product) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("update product: %w", err)
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx,
		`UPDATE products SET identifier = ?, display_name = ?, apple_product_id = ?,
		        google_product_id = ?, billing_period = ?, price_amount_micros = ?, currency = ?,
		        trial_period = ?, intro_price_amount_micros = ?, intro_period = ?, offering = ?,
		        sort_order = ?, updated_at = ?
		  WHERE project_id = ? AND id = ?`,
		p.Identifier, p.DisplayName, nullString(p.AppleProductID), nullString(p.GoogleProductID),
		p.BillingPeriod, p.PriceAmountMicros, p.Currency, p.TrialPeriod, p.IntroPriceAmountMicros,
		p.IntroPeriod, p.Offering, p.SortOrder, formatTime(p.UpdatedAt), p.ProjectID, p.ID)
	if err != nil {
		return fmt.Errorf("update product: %w", err)
	}
	if err := requireRow(res); err != nil {
		return err
	}
	if err := replaceProductEntitlements(ctx, tx, p.ID, p.EntitlementIDs); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit update product: %w", err)
	}
	return nil
}

func replaceProductEntitlements(ctx context.Context, tx *sql.Tx, productID string, entitlementIDs []string) error {
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM product_entitlements WHERE product_id = ?`, productID); err != nil {
		return fmt.Errorf("clear product entitlements: %w", err)
	}
	for _, eid := range entitlementIDs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO product_entitlements (product_id, entitlement_id) VALUES (?, ?)`,
			productID, eid); err != nil {
			return fmt.Errorf("insert product entitlement: %w", err)
		}
	}
	return nil
}

func (s *Store) GetProduct(ctx context.Context, projectID, id string) (Product, error) {
	row := s.db.QueryRowContext(ctx, selectProduct+` WHERE project_id = ? AND id = ?`, projectID, id)
	p, err := scanProduct(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Product{}, ErrNotFound
	}
	if err != nil {
		return Product{}, err
	}
	if p.EntitlementIDs, err = s.productEntitlementIDs(ctx, p.ID); err != nil {
		return Product{}, err
	}
	return p, nil
}

// ListProducts returns the project's products ordered by (sort_order, id) —
// the offering listing order. Entitlement grants are loaded per product.
func (s *Store) ListProducts(ctx context.Context, projectID string) ([]Product, error) {
	rows, err := s.db.QueryContext(ctx,
		selectProduct+` WHERE project_id = ? ORDER BY sort_order, id`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()
	var out []Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	for i := range out {
		ids, err := s.productEntitlementIDs(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].EntitlementIDs = ids
	}
	return out, nil
}

func (s *Store) DeleteProduct(ctx context.Context, projectID, id string) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM products WHERE project_id = ? AND id = ?`, projectID, id)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	return requireRow(res)
}

func (s *Store) productEntitlementIDs(ctx context.Context, productID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT entitlement_id FROM product_entitlements WHERE product_id = ? ORDER BY entitlement_id`,
		productID)
	if err != nil {
		return nil, fmt.Errorf("list product entitlements: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan product entitlement: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

const selectProduct = `SELECT id, project_id, identifier, display_name,
	COALESCE(apple_product_id, ''), COALESCE(google_product_id, ''), billing_period,
	price_amount_micros, currency, trial_period, intro_price_amount_micros, intro_period,
	offering, sort_order, created_at, updated_at FROM products`

func scanProduct(row rowScanner) (Product, error) {
	var p Product
	var createdAt, updatedAt string
	if err := row.Scan(&p.ID, &p.ProjectID, &p.Identifier, &p.DisplayName, &p.AppleProductID,
		&p.GoogleProductID, &p.BillingPeriod, &p.PriceAmountMicros, &p.Currency, &p.TrialPeriod,
		&p.IntroPriceAmountMicros, &p.IntroPeriod, &p.Offering, &p.SortOrder, &createdAt, &updatedAt); err != nil {
		return Product{}, err
	}
	var err error
	if p.CreatedAt, err = parseTime(createdAt); err != nil {
		return Product{}, fmt.Errorf("parse product created_at: %w", err)
	}
	if p.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return Product{}, fmt.Errorf("parse product updated_at: %w", err)
	}
	return p, nil
}

// --- Subscriptions --------------------------------------------------------

// UpsertSubscription inserts or updates the subscription keyed on its store
// identity (project_id, store, store_transaction_id) and returns the stored
// row. The caller's ID and CreatedAt are used only on first insert; on a
// conflict the existing id/created_at are preserved and the mutable fields are
// overwritten from the fresh store read.
func (s *Store) UpsertSubscription(ctx context.Context, sub Subscription) (Subscription, error) {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO subscriptions (id, project_id, user_id, store, product_id, store_transaction_id,
		        subscription_id, status, current_period_end, auto_renew, environment, raw_state,
		        created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (project_id, store, store_transaction_id) DO UPDATE SET
		        user_id = excluded.user_id,
		        product_id = excluded.product_id,
		        subscription_id = excluded.subscription_id,
		        status = excluded.status,
		        current_period_end = excluded.current_period_end,
		        auto_renew = excluded.auto_renew,
		        environment = excluded.environment,
		        raw_state = excluded.raw_state,
		        updated_at = excluded.updated_at`,
		sub.ID, sub.ProjectID, sub.UserID, sub.Store, nullString(sub.ProductID), sub.StoreTransactionID,
		sub.SubscriptionID, sub.Status, formatNullTime(sub.CurrentPeriodEnd), sub.AutoRenew,
		sub.Environment, sub.RawState, formatTime(sub.CreatedAt), formatTime(sub.UpdatedAt))
	if err != nil {
		return Subscription{}, fmt.Errorf("upsert subscription: %w", err)
	}
	return s.GetSubscriptionByStoreID(ctx, sub.ProjectID, sub.Store, sub.StoreTransactionID)
}

func (s *Store) GetSubscription(ctx context.Context, projectID, id string) (Subscription, error) {
	row := s.db.QueryRowContext(ctx, selectSubscription+` WHERE project_id = ? AND id = ?`, projectID, id)
	return scanSubscriptionRow(row)
}

func (s *Store) GetSubscriptionByStoreID(ctx context.Context, projectID, storeName, storeTransactionID string) (Subscription, error) {
	row := s.db.QueryRowContext(ctx,
		selectSubscription+` WHERE project_id = ? AND store = ? AND store_transaction_id = ?`,
		projectID, storeName, storeTransactionID)
	return scanSubscriptionRow(row)
}

// ListUserSubscriptions returns a user's subscriptions, newest first.
func (s *Store) ListUserSubscriptions(ctx context.Context, projectID, userID string) ([]Subscription, error) {
	rows, err := s.db.QueryContext(ctx,
		selectSubscription+` WHERE project_id = ? AND user_id = ? ORDER BY created_at DESC, id DESC`,
		projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("list user subscriptions: %w", err)
	}
	defer rows.Close()
	var out []Subscription
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sub)
	}
	return out, rows.Err()
}

// ListSubscriptionsForReconciliation returns access-granting subscriptions
// (active, trialing, in_grace_period, in_billing_retry) whose current_period_end
// has passed cutoff, across every project, oldest-expiry first and capped at
// limit. The reconciliation sweep re-reads these from the store API to catch
// renewals or lapses a missed webhook would otherwise leave stale.
func (s *Store) ListSubscriptionsForReconciliation(ctx context.Context, cutoff time.Time, limit int) ([]Subscription, error) {
	rows, err := s.db.QueryContext(ctx,
		selectSubscription+` WHERE status IN (?, ?, ?, ?)
		        AND current_period_end IS NOT NULL AND current_period_end < ?
		  ORDER BY current_period_end LIMIT ?`,
		SubscriptionStatusActive, SubscriptionStatusTrialing, SubscriptionStatusInGracePeriod,
		SubscriptionStatusInBillingRetry, formatTime(cutoff), limit)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions for reconciliation: %w", err)
	}
	defer rows.Close()
	var out []Subscription
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sub)
	}
	return out, rows.Err()
}

const selectSubscription = `SELECT id, project_id, user_id, store, COALESCE(product_id, ''),
	store_transaction_id, subscription_id, status, current_period_end, auto_renew, environment,
	raw_state, created_at, updated_at FROM subscriptions`

func scanSubscriptionRow(row *sql.Row) (Subscription, error) {
	sub, err := scanSubscription(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Subscription{}, ErrNotFound
	}
	return sub, err
}

func scanSubscription(row rowScanner) (Subscription, error) {
	var sub Subscription
	var currentPeriodEnd sql.NullString
	var createdAt, updatedAt string
	if err := row.Scan(&sub.ID, &sub.ProjectID, &sub.UserID, &sub.Store, &sub.ProductID,
		&sub.StoreTransactionID, &sub.SubscriptionID, &sub.Status, &currentPeriodEnd, &sub.AutoRenew,
		&sub.Environment, &sub.RawState, &createdAt, &updatedAt); err != nil {
		return Subscription{}, err
	}
	var err error
	if sub.CurrentPeriodEnd, err = parseNullTime(currentPeriodEnd); err != nil {
		return Subscription{}, fmt.Errorf("parse subscription current_period_end: %w", err)
	}
	if sub.CreatedAt, err = parseTime(createdAt); err != nil {
		return Subscription{}, fmt.Errorf("parse subscription created_at: %w", err)
	}
	if sub.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return Subscription{}, fmt.Errorf("parse subscription updated_at: %w", err)
	}
	return sub, nil
}

// --- Subscription grants --------------------------------------------------

func (s *Store) CreateSubscriptionGrant(ctx context.Context, g SubscriptionGrant) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO subscription_grants (id, project_id, user_id, entitlement_id, expires_at,
		        reason, granted_by, created_at, revoked_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		g.ID, g.ProjectID, g.UserID, g.EntitlementID, formatNullTime(g.ExpiresAt),
		g.Reason, g.GrantedBy, formatTime(g.CreatedAt), formatNullTime(g.RevokedAt))
	if err != nil {
		return fmt.Errorf("create subscription grant: %w", err)
	}
	return nil
}

func (s *Store) GetSubscriptionGrant(ctx context.Context, projectID, id string) (SubscriptionGrant, error) {
	row := s.db.QueryRowContext(ctx, selectGrant+` WHERE project_id = ? AND id = ?`, projectID, id)
	g, err := scanGrant(row)
	if errors.Is(err, sql.ErrNoRows) {
		return SubscriptionGrant{}, ErrNotFound
	}
	return g, err
}

// ListUserGrants returns all of a user's grants (active, expired and revoked),
// newest first — the admin history view.
func (s *Store) ListUserGrants(ctx context.Context, projectID, userID string) ([]SubscriptionGrant, error) {
	return s.queryGrants(ctx,
		selectGrant+` WHERE project_id = ? AND user_id = ? ORDER BY created_at DESC, id DESC`,
		projectID, userID)
}

// ListActiveUserGrants returns a user's grants that are neither revoked nor
// expired at now — the rows the entitlement engine unions with store state.
func (s *Store) ListActiveUserGrants(ctx context.Context, projectID, userID string, now time.Time) ([]SubscriptionGrant, error) {
	return s.queryGrants(ctx,
		selectGrant+` WHERE project_id = ? AND user_id = ? AND revoked_at IS NULL
		        AND (expires_at IS NULL OR expires_at > ?)
		  ORDER BY created_at DESC, id DESC`,
		projectID, userID, timeBound(now))
}

// RevokeSubscriptionGrant marks a grant revoked; revoking an absent grant, or
// one already revoked, returns ErrNotFound.
func (s *Store) RevokeSubscriptionGrant(ctx context.Context, projectID, id string, now time.Time) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE subscription_grants SET revoked_at = ?
		  WHERE project_id = ? AND id = ? AND revoked_at IS NULL`,
		formatTime(now), projectID, id)
	if err != nil {
		return fmt.Errorf("revoke subscription grant: %w", err)
	}
	return requireRow(res)
}

func (s *Store) queryGrants(ctx context.Context, query string, args ...any) ([]SubscriptionGrant, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list subscription grants: %w", err)
	}
	defer rows.Close()
	var out []SubscriptionGrant
	for rows.Next() {
		g, err := scanGrant(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

const selectGrant = `SELECT id, project_id, user_id, entitlement_id, expires_at, reason,
	granted_by, created_at, revoked_at FROM subscription_grants`

func scanGrant(row rowScanner) (SubscriptionGrant, error) {
	var g SubscriptionGrant
	var expiresAt, revokedAt sql.NullString
	var createdAt string
	if err := row.Scan(&g.ID, &g.ProjectID, &g.UserID, &g.EntitlementID, &expiresAt, &g.Reason,
		&g.GrantedBy, &createdAt, &revokedAt); err != nil {
		return SubscriptionGrant{}, err
	}
	var err error
	if g.ExpiresAt, err = parseNullTime(expiresAt); err != nil {
		return SubscriptionGrant{}, fmt.Errorf("parse grant expires_at: %w", err)
	}
	if g.RevokedAt, err = parseNullTime(revokedAt); err != nil {
		return SubscriptionGrant{}, fmt.Errorf("parse grant revoked_at: %w", err)
	}
	if g.CreatedAt, err = parseTime(createdAt); err != nil {
		return SubscriptionGrant{}, fmt.Errorf("parse grant created_at: %w", err)
	}
	return g, nil
}

// --- Store notifications --------------------------------------------------

// InsertStoreNotificationIfNew inserts the notification and reports whether it
// was new. A replay (same project_id, store, notification_id) is a no-op that
// returns false — the idempotency guard for webhook delivery.
func (s *Store) InsertStoreNotificationIfNew(ctx context.Context, n StoreNotification) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO store_notifications (id, project_id, store, notification_id, type, subtype,
		        raw_payload, processed_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (project_id, store, notification_id) DO NOTHING`,
		n.ID, n.ProjectID, n.Store, n.NotificationID, n.Type, n.Subtype,
		n.RawPayload, formatNullTime(n.ProcessedAt), formatTime(n.CreatedAt))
	if err != nil {
		return false, fmt.Errorf("insert store notification: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("insert store notification: %w", err)
	}
	return affected > 0, nil
}

// GetStoreNotification returns a recorded notification by its store identity, or
// ErrNotFound. The caller inspects ProcessedAt to decide whether a redelivery is
// a genuine replay (already applied) or a row left unprocessed by an earlier
// transient failure that must be re-processed.
func (s *Store) GetStoreNotification(ctx context.Context, projectID, storeName, notificationID string) (StoreNotification, error) {
	var n StoreNotification
	var processedAt sql.NullString
	var createdAt string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, store, notification_id, type, subtype, raw_payload, processed_at, created_at
		   FROM store_notifications WHERE project_id = ? AND store = ? AND notification_id = ?`,
		projectID, storeName, notificationID).Scan(
		&n.ID, &n.ProjectID, &n.Store, &n.NotificationID, &n.Type, &n.Subtype, &n.RawPayload,
		&processedAt, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return StoreNotification{}, ErrNotFound
	}
	if err != nil {
		return StoreNotification{}, fmt.Errorf("get store notification: %w", err)
	}
	if n.ProcessedAt, err = parseNullTime(processedAt); err != nil {
		return StoreNotification{}, fmt.Errorf("parse store notification processed_at: %w", err)
	}
	if n.CreatedAt, err = parseTime(createdAt); err != nil {
		return StoreNotification{}, fmt.Errorf("parse store notification created_at: %w", err)
	}
	return n, nil
}

// MarkStoreNotificationProcessed stamps processed_at once the notification has
// been applied.
func (s *Store) MarkStoreNotificationProcessed(ctx context.Context, projectID, id string, now time.Time) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE store_notifications SET processed_at = ? WHERE project_id = ? AND id = ?`,
		formatTime(now), projectID, id)
	if err != nil {
		return fmt.Errorf("mark store notification processed: %w", err)
	}
	return requireRow(res)
}

// --- Billing credentials --------------------------------------------------

// UpsertBillingCredentials writes a project's store credentials. For each *Enc
// secret, a nil slice keeps the stored ciphertext, an empty slice clears it,
// and a non-empty slice replaces it (write-only secret semantics). Non-secret
// fields are always overwritten.
func (s *Store) UpsertBillingCredentials(ctx context.Context, c BillingCredentials) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO billing_credentials (project_id, apple_iap_key_id, apple_iap_issuer_id,
		        apple_iap_key_enc, apple_bundle_id, apple_app_apple_id, apple_notification_secret_enc,
		        apple_notification_url, google_service_account_enc, google_package_name, google_pubsub_topic,
		        google_rtdn_secret_enc, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (project_id) DO UPDATE SET
		        apple_iap_key_id = excluded.apple_iap_key_id,
		        apple_iap_issuer_id = excluded.apple_iap_issuer_id,
		        apple_iap_key_enc = COALESCE(excluded.apple_iap_key_enc, billing_credentials.apple_iap_key_enc),
		        apple_bundle_id = excluded.apple_bundle_id,
		        apple_app_apple_id = excluded.apple_app_apple_id,
		        apple_notification_secret_enc = COALESCE(excluded.apple_notification_secret_enc, billing_credentials.apple_notification_secret_enc),
		        apple_notification_url = CASE WHEN excluded.apple_notification_url = '' THEN billing_credentials.apple_notification_url ELSE excluded.apple_notification_url END,
		        google_service_account_enc = COALESCE(excluded.google_service_account_enc, billing_credentials.google_service_account_enc),
		        google_package_name = excluded.google_package_name,
		        google_pubsub_topic = excluded.google_pubsub_topic,
		        google_rtdn_secret_enc = COALESCE(excluded.google_rtdn_secret_enc, billing_credentials.google_rtdn_secret_enc),
		        updated_at = excluded.updated_at`,
		c.ProjectID, c.AppleIAPKeyID, c.AppleIAPIssuerID, nullBytes(c.AppleIAPKeyEnc), c.AppleBundleID,
		c.AppleAppAppleID, nullBytes(c.AppleNotificationSecretEnc), c.AppleNotificationURL, nullBytes(c.GoogleServiceAccountEnc),
		c.GooglePackageName, c.GooglePubsubTopic, nullBytes(c.GoogleRTDNSecretEnc),
		formatTime(c.CreatedAt), formatTime(c.UpdatedAt))
	if err != nil {
		return fmt.Errorf("upsert billing credentials: %w", err)
	}
	return nil
}

// GetBillingCredentials returns a project's store credentials, or ErrNotFound
// when none are configured.
func (s *Store) GetBillingCredentials(ctx context.Context, projectID string) (BillingCredentials, error) {
	var c BillingCredentials
	var createdAt, updatedAt string
	err := s.db.QueryRowContext(ctx,
		`SELECT project_id, apple_iap_key_id, apple_iap_issuer_id, apple_iap_key_enc, apple_bundle_id,
		        apple_app_apple_id, apple_notification_secret_enc, apple_notification_url, google_service_account_enc,
		        google_package_name, google_pubsub_topic, google_rtdn_secret_enc, created_at, updated_at
		   FROM billing_credentials WHERE project_id = ?`, projectID).Scan(
		&c.ProjectID, &c.AppleIAPKeyID, &c.AppleIAPIssuerID, &c.AppleIAPKeyEnc, &c.AppleBundleID,
		&c.AppleAppAppleID, &c.AppleNotificationSecretEnc, &c.AppleNotificationURL, &c.GoogleServiceAccountEnc,
		&c.GooglePackageName, &c.GooglePubsubTopic, &c.GoogleRTDNSecretEnc, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return BillingCredentials{}, ErrNotFound
	}
	if err != nil {
		return BillingCredentials{}, fmt.Errorf("get billing credentials: %w", err)
	}
	if c.CreatedAt, err = parseTime(createdAt); err != nil {
		return BillingCredentials{}, fmt.Errorf("parse billing credentials created_at: %w", err)
	}
	if c.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return BillingCredentials{}, fmt.Errorf("parse billing credentials updated_at: %w", err)
	}
	return c, nil
}

// --- Subscription events --------------------------------------------------

func (s *Store) InsertSubscriptionEvent(ctx context.Context, e SubscriptionEvent) error {
	return s.InsertSubscriptionEvents(ctx, []SubscriptionEvent{e})
}

// InsertSubscriptionEvents writes a batch of revenue events in one transaction.
func (s *Store) InsertSubscriptionEvents(ctx context.Context, events []SubscriptionEvent) error {
	if len(events) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("insert subscription events: %w", err)
	}
	defer tx.Rollback()
	for _, e := range events {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO subscription_events (id, project_id, type, user_id, product_id, store,
			        price_amount_micros, currency, environment, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			e.ID, e.ProjectID, e.Type, nullString(e.UserID), nullString(e.ProductID), e.Store,
			e.PriceAmountMicros, e.Currency, e.Environment, formatTime(e.CreatedAt)); err != nil {
			return fmt.Errorf("insert subscription event %s: %w", e.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("insert subscription events: %w", err)
	}
	return nil
}

// nullBytes maps a nil slice to a SQL NULL (keep-existing under COALESCE) while
// preserving an empty non-nil slice as an empty BLOB (clear).
func nullBytes(b []byte) any {
	if b == nil {
		return nil
	}
	return b
}
