// Package store is the SQLite persistence layer: connection setup,
// embedded migrations, and hand-written queries behind small per-domain
// interfaces (no ORM).
package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// AdminStore persists instance operators.
type AdminStore interface {
	CreateAdmin(ctx context.Context, a Admin) error
	// UpsertAdmin creates the admin or, when the email already exists,
	// resets its password hash. Used by `moth admin create`.
	UpsertAdmin(ctx context.Context, a Admin) error
	GetAdmin(ctx context.Context, id string) (Admin, error)
	GetAdminByEmail(ctx context.Context, email string) (Admin, error)
	ListAdmins(ctx context.Context) ([]Admin, error)
	UpdateAdminPassword(ctx context.Context, id, passwordHash string, now time.Time) error
	CountAdmins(ctx context.Context) (int, error)
}

// AdminInviteStore persists pending operator invitations.
type AdminInviteStore interface {
	CreateAdminInvite(ctx context.Context, inv AdminInvite) error
	GetAdminInviteByTokenHash(ctx context.Context, tokenHash string) (AdminInvite, error)
	ListAdminInvites(ctx context.Context) ([]AdminInvite, error)
	DeleteAdminInvite(ctx context.Context, id string) error
}

// SessionStore persists admin browser sessions (cookie tokens are stored
// hashed).
type SessionStore interface {
	CreateSession(ctx context.Context, s AdminSession) error
	GetSession(ctx context.Context, tokenHash string) (AdminSession, error)
	DeleteSession(ctx context.Context, tokenHash string) error
	DeleteAdminSessionsExcept(ctx context.Context, adminID, keepTokenHash string) error
	DeleteExpiredSessions(ctx context.Context, now time.Time) error
}

// PersonalAccessTokenStore persists admin CLI/API credentials
// (moth_pat_... values are stored hashed).
type PersonalAccessTokenStore interface {
	CreatePAT(ctx context.Context, t PersonalAccessToken) error
	GetPATByHash(ctx context.Context, tokenHash string) (PersonalAccessToken, error)
	ListPATs(ctx context.Context, adminID string) ([]PersonalAccessToken, error)
	TouchPAT(ctx context.Context, id string, at time.Time) error
	RevokePAT(ctx context.Context, adminID, id string, now time.Time) error
	DeleteExpiredPATs(ctx context.Context, now time.Time) error
}

// InstanceSettingStore persists instance-wide admin-edited settings.
type InstanceSettingStore interface {
	GetInstanceSetting(ctx context.Context, key string) (string, error)
	SetInstanceSetting(ctx context.Context, key, value string, now time.Time) error
	DeleteInstanceSetting(ctx context.Context, key string) error
}

// InstanceSecretStore persists instance-wide secrets (e.g. the SMTP relay
// password) as ciphertext encrypted by the caller under the master key.
type InstanceSecretStore interface {
	SetInstanceSecret(ctx context.Context, key string, secretEnc []byte, now time.Time) error
	GetInstanceSecret(ctx context.Context, key string) ([]byte, error)
	DeleteInstanceSecret(ctx context.Context, key string) error
}

// RateLimitStore persists restart-surviving rate-limit buckets shared by the
// gRPC interceptor and HTTP middleware.
type RateLimitStore interface {
	// TakeRateLimit atomically records n hits against key's fixed window and
	// reports whether the bucket stays within limit.
	TakeRateLimit(ctx context.Context, key string, n, limit int, window time.Duration, now time.Time) (RateLimitResult, error)
	DeleteStaleRateLimits(ctx context.Context, cutoff time.Time) (int64, error)
}

// AuditStore persists the append-only admin/security audit log.
type AuditStore interface {
	AppendAudit(ctx context.Context, e AuditEntry) error
	ListAudit(ctx context.Context, filter AuditFilter) ([]AuditEntry, error)
}

// UserMigrationStore bulk-reads and bulk-writes users for project data
// export/import (migration on and off moth).
type UserMigrationStore interface {
	ExportUsers(ctx context.Context, projectID string) ([]UserExport, error)
	ImportUsers(ctx context.Context, projectID string, users []UserImport, now time.Time) (ImportResult, error)
}

// ProjectStore persists projects and their signing keys.
type ProjectStore interface {
	// CreateProject inserts the project and its first signing key in one
	// transaction: a project must never exist without a keypair.
	CreateProject(ctx context.Context, p Project, k ProjectKey) error
	GetProject(ctx context.Context, id string) (Project, error)
	GetProjectBySlug(ctx context.Context, slug string) (Project, error)
	GetProjectByPublishableKey(ctx context.Context, key string) (Project, error)
	GetProjectBySecretKeyHash(ctx context.Context, keyHash string) (Project, error)
	ListProjects(ctx context.Context) ([]Project, error)
	UpdateProject(ctx context.Context, p Project) error
	UpdateProjectSecretKey(ctx context.Context, id, secretKeyHash string, now time.Time) error
	// ResetProjectSigningKey retires all keys, installs k and revokes the
	// project's refresh tokens in one transaction.
	ResetProjectSigningKey(ctx context.Context, projectID string, k ProjectKey, now time.Time) error
	DeleteProject(ctx context.Context, id string) error
	SlugExists(ctx context.Context, slug string) (bool, error)
	ListActiveProjectKeys(ctx context.Context, projectID string) ([]ProjectKey, error)
	// RotateSigningKey installs a new active key and moves the current one to
	// a grace period (kept in the JWKS until graceUntil); refresh tokens are
	// preserved, unlike ResetProjectSigningKey.
	RotateSigningKey(ctx context.Context, projectID string, k ProjectKey, graceUntil, now time.Time) error
	// ListActiveAndGraceKeys returns the keys the project JWKS must publish:
	// the active key plus grace keys not yet expired at now.
	ListActiveAndGraceKeys(ctx context.Context, projectID string, now time.Time) ([]ProjectKey, error)
	// PruneExpiredKeys deletes grace keys whose grace ended by now.
	PruneExpiredKeys(ctx context.Context, now time.Time) (int64, error)
}

// UserStore persists a project's end users and their provider identities.
type UserStore interface {
	CreateUser(ctx context.Context, u User, identities ...Identity) error
	CreateIdentity(ctx context.Context, id Identity) error
	GetIdentity(ctx context.Context, projectID, provider, subject string) (Identity, error)
	ListUserIdentities(ctx context.Context, projectID, userID string) ([]Identity, error)
	DeleteUserIdentities(ctx context.Context, projectID, userID, provider string) error
	SetIdentityAppleRefreshToken(ctx context.Context, projectID, id string, tokenEnc []byte) error
	SetIdentityProviderEmail(ctx context.Context, projectID, id, email string) error
	GetUser(ctx context.Context, projectID, id string) (User, error)
	GetUserByEmail(ctx context.Context, projectID, email string) (User, error)
	ListUsers(ctx context.Context, projectID string) ([]User, error)
	ListUsersPage(ctx context.Context, projectID string, page UserPage) ([]User, error)
	CountUsers(ctx context.Context, projectID, query string) (int, error)
	CountUsersByProject(ctx context.Context) (map[string]int, error)
	ListIdentitiesForUsers(ctx context.Context, projectID string, userIDs []string) (map[string][]Identity, error)
	UpdateUser(ctx context.Context, u User) error
	SetUserLastLogin(ctx context.Context, projectID, id string, at time.Time) error
	SetUserPasswordHash(ctx context.Context, projectID, id, hash, algo string, at time.Time) error
	DeleteUser(ctx context.Context, projectID, id string) error
}

// RefreshTokenStore persists rotating refresh tokens.
type RefreshTokenStore interface {
	CreateRefreshToken(ctx context.Context, rt RefreshToken) error
	GetRefreshToken(ctx context.Context, projectID, tokenHash string) (RefreshToken, error)
	ListActiveUserRefreshTokens(ctx context.Context, projectID, userID string, now time.Time) ([]RefreshToken, error)
	RotateRefreshToken(ctx context.Context, oldID string, rotatedAt time.Time, successor RefreshToken) error
	RevokeRefreshToken(ctx context.Context, projectID, id string, now time.Time) error
	RevokeRefreshTokenFamily(ctx context.Context, projectID, familyID string, now time.Time) error
	RevokeUserRefreshTokens(ctx context.Context, projectID, userID string, now time.Time) error
}

// EmailTokenStore persists single-use verification/reset/change tokens.
type EmailTokenStore interface {
	CreateEmailToken(ctx context.Context, et EmailToken) error
	GetEmailToken(ctx context.Context, projectID, tokenHash string) (EmailToken, error)
	ConsumeEmailToken(ctx context.Context, projectID, id string, now time.Time) error
	DeleteUserEmailTokens(ctx context.Context, projectID, userID, purpose string) error
}

// OAuthTokenStore persists the single-use artifacts of the web-redirect
// OAuth fallback (hashed state values and one-time exchange codes).
type OAuthTokenStore interface {
	CreateOAuthToken(ctx context.Context, ot OAuthToken) error
	ConsumeOAuthToken(ctx context.Context, projectID, purpose, tokenHash string, now time.Time) (OAuthToken, error)
	DeleteExpiredOAuthTokens(ctx context.Context, now time.Time) error
}

// ProviderSecretStore persists per-project provider secrets (Apple .p8
// private key, Google web client secret) as ciphertext encrypted by the
// caller under the master key.
type ProviderSecretStore interface {
	SetProviderSecret(ctx context.Context, projectID, name string, secretEnc []byte, now time.Time) error
	GetProviderSecret(ctx context.Context, projectID, name string) ([]byte, error)
	DeleteProviderSecret(ctx context.Context, projectID, name string) error
}

// ThemeStore persists per-project design-system themes and their revision
// history (raw internal/theme JSON documents).
type ThemeStore interface {
	SetProjectTheme(ctx context.Context, rev ThemeRevision, prevRevisionID string) error
	ClearProjectTheme(ctx context.Context, projectID string, now time.Time) error
	GetThemeRevision(ctx context.Context, projectID, revisionID string) (ThemeRevision, error)
	ListThemeRevisions(ctx context.Context, projectID string, limit int) ([]ThemeRevision, error)
}

// PaywallStore persists per-project paywall configs and their revision
// history (raw internal/paywall JSON documents). Mirrors ThemeStore.
type PaywallStore interface {
	SetProjectPaywall(ctx context.Context, rev PaywallRevision, prevRevisionID string) error
	ClearProjectPaywall(ctx context.Context, projectID string, now time.Time) error
	GetPaywallRevision(ctx context.Context, projectID, revisionID string) (PaywallRevision, error)
	ListPaywallRevisions(ctx context.Context, projectID string, limit int) ([]PaywallRevision, error)
}

// EventStore records and reads the raw analytics event stream.
type EventStore interface {
	InsertEvent(ctx context.Context, e Event) error
	// InsertEvents writes a batch in one transaction (the async event
	// writer flushes through this).
	InsertEvents(ctx context.Context, events []Event) error
	ListRecentEvents(ctx context.Context, projectID string, limit int) ([]Event, error)
	// DeleteEventsBefore prunes events older than cutoff (the project's
	// analytics retention window) and reports how many were removed.
	DeleteEventsBefore(ctx context.Context, projectID string, cutoff time.Time) (int64, error)
}

// StatsStore persists the daily analytics rollups and the runs of the job
// that produces them. Aggregation windows are UTC instants; the caller
// converts the project's local day with DayWindow.
type StatsStore interface {
	AggregateDailyStats(ctx context.Context, projectID, date string, from, to time.Time) (DailyStats, error)
	// UpsertDailyStats replaces the (project, date) row; re-rolling a day
	// is idempotent.
	UpsertDailyStats(ctx context.Context, ds DailyStats) error
	GetDailyStats(ctx context.Context, projectID, fromDate, toDate string) ([]DailyStats, error)
	// LatestDailyStatsDate returns the newest rolled-up date of a project,
	// "" when none.
	LatestDailyStatsDate(ctx context.Context, projectID string) (string, error)
	InsertRollupRun(ctx context.Context, r RollupRun) error
	ListRollupRuns(ctx context.Context, limit int) ([]RollupRun, error)
}

// SubscriptionStatsStore persists the milestone-14 subscription revenue
// rollups (monthly, per currency) and the per-tier slices. Aggregation windows
// are UTC instants; the caller converts the project's local month with
// MonthWindow. Dashboards read these rows, never the raw subscription_events.
type SubscriptionStatsStore interface {
	// AggregateSubscription computes the (project, month) rollup from the raw
	// events in [from, to); includeSandbox=false excludes sandbox events. It
	// also returns the currency-agnostic distinct active-subscriber counts
	// (per month and per tier) that must not be summed from the per-currency
	// rows.
	AggregateSubscription(ctx context.Context, projectID, period string, from, to time.Time, includeSandbox bool) ([]SubscriptionStats, []SubscriptionTierStats, []SubscriptionPeriodActive, error)
	// UpsertSubscriptionStats replaces all rows for (project, period); re-rolling
	// a month is idempotent.
	UpsertSubscriptionStats(ctx context.Context, projectID, period string, stats []SubscriptionStats, tiers []SubscriptionTierStats, periodActive []SubscriptionPeriodActive) error
	GetSubscriptionStats(ctx context.Context, projectID, fromPeriod, toPeriod string) ([]SubscriptionStats, error)
	GetSubscriptionTierStats(ctx context.Context, projectID, fromPeriod, toPeriod string) ([]SubscriptionTierStats, error)
	// GetSubscriptionPeriodActive returns the currency-agnostic distinct
	// active-subscriber counts per (period, product_id); product_id "" is the
	// all-products month total.
	GetSubscriptionPeriodActive(ctx context.Context, projectID, fromPeriod, toPeriod string) ([]SubscriptionPeriodActive, error)
	// LatestSubscriptionStatsPeriod returns the newest rolled-up month, "" when none.
	LatestSubscriptionStatsPeriod(ctx context.Context, projectID string) (string, error)
	// DeleteSubscriptionEventsBefore prunes raw events older than cutoff.
	DeleteSubscriptionEventsBefore(ctx context.Context, projectID string, cutoff time.Time) (int64, error)
}

// EntitlementStore persists a project's named capability definitions.
type EntitlementStore interface {
	CreateEntitlement(ctx context.Context, e Entitlement) error
	GetEntitlement(ctx context.Context, projectID, id string) (Entitlement, error)
	GetEntitlementByIdentifier(ctx context.Context, projectID, identifier string) (Entitlement, error)
	ListEntitlements(ctx context.Context, projectID string) ([]Entitlement, error)
	UpdateEntitlement(ctx context.Context, e Entitlement) error
	DeleteEntitlement(ctx context.Context, projectID, id string) error
}

// ProductStore persists a project's subscription tiers and the entitlements
// each grants.
type ProductStore interface {
	CreateProduct(ctx context.Context, p Product) error
	GetProduct(ctx context.Context, projectID, id string) (Product, error)
	ListProducts(ctx context.Context, projectID string) ([]Product, error)
	UpdateProduct(ctx context.Context, p Product) error
	DeleteProduct(ctx context.Context, projectID, id string) error
}

// SubscriptionStore persists moth's mirror of store subscriptions. The store
// returns rows and active grants; entitlement derivation lives above it.
type SubscriptionStore interface {
	// UpsertSubscription inserts or updates by store identity (project_id,
	// store, store_transaction_id) and returns the stored row.
	UpsertSubscription(ctx context.Context, sub Subscription) (Subscription, error)
	GetSubscription(ctx context.Context, projectID, id string) (Subscription, error)
	GetSubscriptionByStoreID(ctx context.Context, projectID, store, storeTransactionID string) (Subscription, error)
	ListUserSubscriptions(ctx context.Context, projectID, userID string) ([]Subscription, error)
	// ListSubscriptionsForReconciliation returns access-granting subscriptions
	// whose current_period_end is before cutoff, across all projects — the rows
	// a reconciliation sweep re-reads to catch missed store notifications.
	ListSubscriptionsForReconciliation(ctx context.Context, cutoff time.Time, limit int) ([]Subscription, error)
}

// SubscriptionGrantStore persists manual/promotional entitlement grants.
type SubscriptionGrantStore interface {
	CreateSubscriptionGrant(ctx context.Context, g SubscriptionGrant) error
	GetSubscriptionGrant(ctx context.Context, projectID, id string) (SubscriptionGrant, error)
	ListUserGrants(ctx context.Context, projectID, userID string) ([]SubscriptionGrant, error)
	// ListActiveUserGrants returns grants neither revoked nor expired at now.
	ListActiveUserGrants(ctx context.Context, projectID, userID string, now time.Time) ([]SubscriptionGrant, error)
	RevokeSubscriptionGrant(ctx context.Context, projectID, id string, now time.Time) error
}

// StoreNotificationStore persists raw store notifications for idempotency and
// audit.
type StoreNotificationStore interface {
	// InsertStoreNotificationIfNew reports whether the notification was new
	// (false on a deduped replay).
	InsertStoreNotificationIfNew(ctx context.Context, n StoreNotification) (bool, error)
	// GetStoreNotification returns a recorded notification by its store
	// identity (project_id, store, notification_id), or ErrNotFound. Callers
	// use ProcessedAt to distinguish a still-unprocessed row (re-processable
	// after a transient failure) from a genuinely applied replay.
	GetStoreNotification(ctx context.Context, projectID, storeName, notificationID string) (StoreNotification, error)
	MarkStoreNotificationProcessed(ctx context.Context, projectID, id string, now time.Time) error
}

// BillingCredentialStore persists per-project store API credentials. Secret
// fields are AES-GCM ciphertext under the master key (encryption in the handler
// layer); a nil *Enc slice on upsert keeps the stored value.
type BillingCredentialStore interface {
	UpsertBillingCredentials(ctx context.Context, c BillingCredentials) error
	GetBillingCredentials(ctx context.Context, projectID string) (BillingCredentials, error)
}

// SubscriptionEventStore records the raw revenue event stream (milestone 14
// rolls it up).
type SubscriptionEventStore interface {
	InsertSubscriptionEvent(ctx context.Context, e SubscriptionEvent) error
	InsertSubscriptionEvents(ctx context.Context, events []SubscriptionEvent) error
}

// ProductStoreSyncStore persists the per-product, per-store catalog
// reconciliation bookkeeping the milestone-12 sync/diff layer reads and writes,
// plus the offering-reorder primitive (products share an `offering` tag ordered
// by sort_order; there is no separate offering table).
type ProductStoreSyncStore interface {
	UpsertProductStoreSync(ctx context.Context, r ProductStoreSync) error
	GetProductStoreSync(ctx context.Context, productID, store string) (ProductStoreSync, error)
	// ListProductStoreSyncs returns a project's sync records for the given
	// store (store == "" returns all stores).
	ListProductStoreSyncs(ctx context.Context, projectID, store string) ([]ProductStoreSync, error)
	// SetProductSortOrders rewrites several products' sort_order atomically.
	SetProductSortOrders(ctx context.Context, projectID string, orders map[string]int) error
}

// Store implements every per-domain store interface on SQLite.
type Store struct {
	db *sql.DB
}

var (
	_ AdminStore               = (*Store)(nil)
	_ AdminInviteStore         = (*Store)(nil)
	_ SessionStore             = (*Store)(nil)
	_ PersonalAccessTokenStore = (*Store)(nil)
	_ InstanceSettingStore     = (*Store)(nil)
	_ InstanceSecretStore      = (*Store)(nil)
	_ RateLimitStore           = (*Store)(nil)
	_ AuditStore               = (*Store)(nil)
	_ UserMigrationStore       = (*Store)(nil)
	_ ProjectStore             = (*Store)(nil)
	_ UserStore                = (*Store)(nil)
	_ RefreshTokenStore        = (*Store)(nil)
	_ EmailTokenStore          = (*Store)(nil)
	_ OAuthTokenStore          = (*Store)(nil)
	_ ProviderSecretStore      = (*Store)(nil)
	_ ThemeStore               = (*Store)(nil)
	_ PaywallStore             = (*Store)(nil)
	_ EventStore               = (*Store)(nil)
	_ StatsStore               = (*Store)(nil)
	_ SubscriptionStatsStore   = (*Store)(nil)
	_ EntitlementStore         = (*Store)(nil)
	_ ProductStore             = (*Store)(nil)
	_ SubscriptionStore        = (*Store)(nil)
	_ SubscriptionGrantStore   = (*Store)(nil)
	_ StoreNotificationStore   = (*Store)(nil)
	_ BillingCredentialStore   = (*Store)(nil)
	_ SubscriptionEventStore   = (*Store)(nil)
	_ ProductStoreSyncStore    = (*Store)(nil)
)

// Open opens (creating if needed) the SQLite database at path with WAL
// mode, foreign keys and a busy timeout.
func Open(path string) (*Store, error) {
	dsn := "file:" + path +
		"?_pragma=journal_mode(WAL)" +
		"&_pragma=foreign_keys(1)" +
		"&_pragma=busy_timeout(5000)" +
		"&_pragma=synchronous(NORMAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return &Store{db: db}, nil
}

// Close closes the underlying database.
func (s *Store) Close() error { return s.db.Close() }

// Migrate applies all embedded migrations that are not yet recorded in
// schema_migrations. It is idempotent and runs on every startup.
func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		version, err := migrationVersion(name)
		if err != nil {
			return err
		}
		var applied bool
		if err := s.db.QueryRowContext(ctx,
			`SELECT COUNT(*) > 0 FROM schema_migrations WHERE version = ?`, version,
		).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", name, err)
		}
		if applied {
			continue
		}
		sqlText, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx, string(sqlText)); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`,
			version, formatTime(time.Now()),
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}
	}
	return nil
}

// migrationVersion extracts the numeric prefix of "0001_init.sql".
func migrationVersion(name string) (int, error) {
	prefix, _, ok := strings.Cut(name, "_")
	if !ok {
		return 0, fmt.Errorf("migration %s: name must be NNNN_description.sql", name)
	}
	v, err := strconv.Atoi(prefix)
	if err != nil {
		return 0, fmt.Errorf("migration %s: invalid version prefix: %w", name, err)
	}
	return v, nil
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, s)
}
