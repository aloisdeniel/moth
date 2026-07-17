package store

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/aloisdeniel/moth/internal/paywall"
	"github.com/aloisdeniel/moth/internal/theme"
)

// parseLegacyCopyJSON decodes a pre-0019 JSON copy-override document (a
// locale → key → value map). BACKFILL ONLY: the frozen legacy shape, kept
// solely for backfillProtoConfigs; live paths use parseCopyOverrides.
func parseLegacyCopyJSON(raw []byte) (CopyOverrides, error) {
	o := CopyOverrides{}
	if err := json.Unmarshal(raw, &o); err != nil {
		return nil, fmt.Errorf("parse legacy copy overrides: %w", err)
	}
	return o, nil
}

// backfillProtoConfigs is the one-time JSON→protobuf conversion that pairs
// with migration 0019: for every row whose *_pb BLOB column is still empty
// while the legacy TEXT column holds a JSON document, it parses the legacy
// JSON (with the frozen legacy parsers), re-encodes it as the
// moth.projectconfig.v1 storage message and writes the BLOB, clearing the TEXT
// column in the same statement. It runs on every startup right after
// migrations apply and is idempotent: once converted (or on a fresh
// database) no row matches and the whole pass is a no-op.
//
// Each table is converted in its own transaction, so a crash mid-way leaves
// every already-converted table done and the rest untouched for the next
// start. A legacy document this binary cannot parse is left in place (TEXT
// kept, BLOB empty — read paths then render defaults, exactly as they did
// for an unparseable JSON document) and counted as skipped rather than
// failing startup; the data is never dropped.
func (s *Store) backfillProtoConfigs(ctx context.Context) error {
	type job struct {
		table     string // table to convert
		legacyCol string // frozen TEXT column holding the legacy JSON
		pbCol     string // BLOB column receiving the proto document
		convert   func(legacy []byte) ([]byte, error)
	}
	themeConv := func(raw []byte) ([]byte, error) {
		t, err := theme.ParseLegacyJSON(raw)
		if err != nil {
			return nil, err
		}
		return theme.Encode(t)
	}
	paywallConv := func(raw []byte) ([]byte, error) {
		c, err := paywall.ParseLegacyJSON(raw)
		if err != nil {
			return nil, err
		}
		return paywall.Encode(c)
	}
	copyConv := func(raw []byte) ([]byte, error) {
		o, err := parseLegacyCopyJSON(raw)
		if err != nil {
			return nil, err
		}
		enc, err := encodeCopyOverrides(o)
		if err != nil {
			return nil, err
		}
		if len(enc) == 0 {
			// A legacy document holding only empty locales/values encodes to
			// the default; a zero-length BLOB already means exactly that.
			return []byte{}, nil
		}
		return enc, nil
	}
	jobs := []job{
		{"projects", "theme", "theme_pb", themeConv},
		{"theme_revisions", "theme", "theme_pb", themeConv},
		{"projects", "paywall", "paywall_pb", paywallConv},
		{"paywall_revisions", "paywall", "paywall_pb", paywallConv},
		{"projects", "copy", "copy_pb", copyConv},
		{"copy_revisions", "copy", "copy_pb", copyConv},
	}

	var converted, skipped int
	for _, j := range jobs {
		c, sk, err := s.backfillTable(ctx, j.table, j.legacyCol, j.pbCol, j.convert)
		if err != nil {
			return fmt.Errorf("backfill %s.%s: %w", j.table, j.pbCol, err)
		}
		converted += c
		skipped += sk
	}
	if converted > 0 || skipped > 0 {
		slog.Info("backfilled project config storage from JSON to protobuf",
			"converted", converted, "skipped_unparseable", skipped)
	}
	return nil
}

// backfillTable converts one (table, legacy TEXT column, proto BLOB column)
// triple in a single transaction and reports how many rows were converted
// and how many were skipped as unparseable.
func (s *Store) backfillTable(ctx context.Context, table, legacyCol, pbCol string, convert func([]byte) ([]byte, error)) (converted, skipped int, err error) {
	// Table and column names come from the fixed job list above, never from
	// input, so string assembly is safe here.
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx,
		`SELECT rowid, `+legacyCol+` FROM `+table+
			` WHERE length(`+pbCol+`) = 0 AND `+legacyCol+` != ''`)
	if err != nil {
		return 0, 0, err
	}
	type pending struct {
		rowid int64
		pb    []byte
	}
	var updates []pending
	var skippedIDs []int64
	for rows.Next() {
		var rowid int64
		var legacy string
		if err := rows.Scan(&rowid, &legacy); err != nil {
			rows.Close()
			return 0, 0, err
		}
		pb, err := convert([]byte(legacy))
		if err != nil {
			// Unparseable legacy document: keep the JSON where it is so
			// nothing is lost; read paths fall back to defaults just as they
			// did before the storage change.
			skippedIDs = append(skippedIDs, rowid)
			continue
		}
		updates = append(updates, pending{rowid: rowid, pb: pb})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, 0, err
	}
	rows.Close()

	for _, u := range updates {
		if _, err := tx.ExecContext(ctx,
			`UPDATE `+table+` SET `+pbCol+` = ?, `+legacyCol+` = '' WHERE rowid = ?`,
			u.pb, u.rowid); err != nil {
			return 0, 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}
	return len(updates), len(skippedIDs), nil
}
