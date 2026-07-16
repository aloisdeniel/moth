package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Product-store sync statuses (the product_store_sync.status column). The
// sync/diff layer (milestone 12) records one of these per (product, store)
// after reconciling moth's catalog into App Store Connect / Google Play.
const (
	// ProductSyncPending is the implied status of a product never pushed to a
	// store (no product_store_sync row exists).
	ProductSyncPending = "pending"
	// ProductSyncInSync means the store SKU matches moth's desired state.
	ProductSyncInSync = "in_sync"
	// ProductSyncDrift means the store SKU exists but differs from moth (e.g. a
	// price changed in moth but not yet pushed, or edited in the console).
	ProductSyncDrift = "drift"
	// ProductSyncError means the last reconcile failed; Error holds the reason.
	ProductSyncError = "error"
)

// DefaultOffering is the offering tag every project has by default — the
// products carrying it (ordered by sort_order) are the paywall's default
// listing. An "offering" is not a separate object: it is the set of products
// sharing this tag (see migration 0013).
const DefaultOffering = "default"

// ProductStoreSync is the per-product, per-store catalog reconciliation record.
// It tracks the sync lifecycle around a store SKU (which lives on
// products.{apple,google}_product_id), not the SKU itself: what moth last
// pushed, when, the store-side revision observed, and whether they now agree.
type ProductStoreSync struct {
	ProductID      string
	Store          string
	Status         string
	StoreProductID string
	Revision       string
	Error          string
	SyncedAt       *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// UpsertProductStoreSync records a reconcile outcome for one (product, store),
// keyed on that pair. CreatedAt is used only on first insert; on conflict the
// mutable fields are overwritten and created_at is preserved.
func (s *Store) UpsertProductStoreSync(ctx context.Context, r ProductStoreSync) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO product_store_sync (product_id, store, status, store_product_id, revision,
		        error, synced_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (product_id, store) DO UPDATE SET
		        status = excluded.status,
		        store_product_id = excluded.store_product_id,
		        revision = excluded.revision,
		        error = excluded.error,
		        synced_at = excluded.synced_at,
		        updated_at = excluded.updated_at`,
		r.ProductID, r.Store, r.Status, r.StoreProductID, r.Revision, r.Error,
		formatNullTime(r.SyncedAt), formatTime(r.CreatedAt), formatTime(r.UpdatedAt))
	if err != nil {
		return fmt.Errorf("upsert product store sync: %w", err)
	}
	return nil
}

// GetProductStoreSync returns the sync record for one (product, store), or
// ErrNotFound when the product has never been synced to that store.
func (s *Store) GetProductStoreSync(ctx context.Context, productID, storeName string) (ProductStoreSync, error) {
	row := s.db.QueryRowContext(ctx, selectProductStoreSync+
		` WHERE product_id = ? AND store = ?`, productID, storeName)
	r, err := scanProductStoreSync(row)
	if errors.Is(err, sql.ErrNoRows) {
		return ProductStoreSync{}, ErrNotFound
	}
	return r, err
}

// ListProductStoreSyncs returns every sync record for a project's products, for
// the given store (storeName == "" returns all stores), ordered by product id
// then store. Products with no record are simply absent (status pending); the
// status view joins these against ListProducts to fill the gaps.
func (s *Store) ListProductStoreSyncs(ctx context.Context, projectID, storeName string) ([]ProductStoreSync, error) {
	query := `SELECT s.product_id, s.store, s.status, s.store_product_id, s.revision,
		s.error, s.synced_at, s.created_at, s.updated_at
		FROM product_store_sync s
		JOIN products p ON p.id = s.product_id WHERE p.project_id = ?`
	args := []any{projectID}
	if storeName != "" {
		query += ` AND s.store = ?`
		args = append(args, storeName)
	}
	query += ` ORDER BY s.product_id, s.store`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list product store syncs: %w", err)
	}
	defer rows.Close()
	var out []ProductStoreSync
	for rows.Next() {
		r, err := scanProductStoreSync(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

const selectProductStoreSync = `SELECT product_id, store, status, store_product_id, revision,
	error, synced_at, created_at, updated_at FROM product_store_sync`

func scanProductStoreSync(row rowScanner) (ProductStoreSync, error) {
	var r ProductStoreSync
	var syncedAt sql.NullString
	var createdAt, updatedAt string
	if err := row.Scan(&r.ProductID, &r.Store, &r.Status, &r.StoreProductID, &r.Revision,
		&r.Error, &syncedAt, &createdAt, &updatedAt); err != nil {
		return ProductStoreSync{}, err
	}
	var err error
	if r.SyncedAt, err = parseNullTime(syncedAt); err != nil {
		return ProductStoreSync{}, fmt.Errorf("parse product store sync synced_at: %w", err)
	}
	if r.CreatedAt, err = parseTime(createdAt); err != nil {
		return ProductStoreSync{}, fmt.Errorf("parse product store sync created_at: %w", err)
	}
	if r.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return ProductStoreSync{}, fmt.Errorf("parse product store sync updated_at: %w", err)
	}
	return r, nil
}

// SetProductSortOrders rewrites the sort_order of several products in one
// transaction — the primitive behind offering reordering (an offering is the
// products sharing an `offering` tag, ordered by sort_order). Each id must
// belong to the project; an id that matches no row makes the whole call fail
// with ErrNotFound so a reorder never silently drops a product.
func (s *Store) SetProductSortOrders(ctx context.Context, projectID string, orders map[string]int) error {
	if len(orders) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("set product sort orders: %w", err)
	}
	defer tx.Rollback()
	now := formatTime(time.Now().UTC())
	for id, order := range orders {
		res, err := tx.ExecContext(ctx,
			`UPDATE products SET sort_order = ?, updated_at = ? WHERE project_id = ? AND id = ?`,
			order, now, projectID, id)
		if err != nil {
			return fmt.Errorf("set product sort order: %w", err)
		}
		if err := requireRow(res); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit set product sort orders: %w", err)
	}
	return nil
}
