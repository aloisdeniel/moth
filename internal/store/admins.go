package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrNotFound is returned when a row does not exist.
var ErrNotFound = errors.New("not found")

// Admin is an operator of the moth instance.
type Admin struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (s *Store) CreateAdmin(ctx context.Context, a Admin) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO admins (id, email, password_hash, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)`,
		a.ID, a.Email, a.PasswordHash, formatTime(a.CreatedAt), formatTime(a.UpdatedAt))
	if err != nil {
		return fmt.Errorf("create admin: %w", err)
	}
	return nil
}

func (s *Store) UpsertAdmin(ctx context.Context, a Admin) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO admins (id, email, password_hash, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT (email) DO UPDATE SET
		   password_hash = excluded.password_hash,
		   updated_at    = excluded.updated_at`,
		a.ID, a.Email, a.PasswordHash, formatTime(a.CreatedAt), formatTime(a.UpdatedAt))
	if err != nil {
		return fmt.Errorf("upsert admin: %w", err)
	}
	return nil
}

func (s *Store) GetAdmin(ctx context.Context, id string) (Admin, error) {
	return s.scanAdmin(s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, created_at, updated_at
		 FROM admins WHERE id = ?`, id))
}

func (s *Store) GetAdminByEmail(ctx context.Context, email string) (Admin, error) {
	return s.scanAdmin(s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, created_at, updated_at
		 FROM admins WHERE email = ?`, email))
}

// ListAdmins returns every operator account, oldest first.
func (s *Store) ListAdmins(ctx context.Context) ([]Admin, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, email, password_hash, created_at, updated_at
		 FROM admins ORDER BY created_at, id`)
	if err != nil {
		return nil, fmt.Errorf("list admins: %w", err)
	}
	defer rows.Close()

	var admins []Admin
	for rows.Next() {
		var a Admin
		var createdAt, updatedAt string
		if err := rows.Scan(&a.ID, &a.Email, &a.PasswordHash, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan admin: %w", err)
		}
		if a.CreatedAt, err = parseTime(createdAt); err != nil {
			return nil, fmt.Errorf("parse admin created_at: %w", err)
		}
		if a.UpdatedAt, err = parseTime(updatedAt); err != nil {
			return nil, fmt.Errorf("parse admin updated_at: %w", err)
		}
		admins = append(admins, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list admins: %w", err)
	}
	return admins, nil
}

// UpdateAdminPassword replaces the admin's password hash.
func (s *Store) UpdateAdminPassword(ctx context.Context, id, passwordHash string, now time.Time) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE admins SET password_hash = ?, updated_at = ? WHERE id = ?`,
		passwordHash, formatTime(now), id)
	if err != nil {
		return fmt.Errorf("update admin password: %w", err)
	}
	return requireRow(res)
}

func (s *Store) CountAdmins(ctx context.Context) (int, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM admins`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count admins: %w", err)
	}
	return n, nil
}

func (s *Store) scanAdmin(row *sql.Row) (Admin, error) {
	var a Admin
	var createdAt, updatedAt string
	err := row.Scan(&a.ID, &a.Email, &a.PasswordHash, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Admin{}, ErrNotFound
	}
	if err != nil {
		return Admin{}, fmt.Errorf("scan admin: %w", err)
	}
	if a.CreatedAt, err = parseTime(createdAt); err != nil {
		return Admin{}, fmt.Errorf("parse admin created_at: %w", err)
	}
	if a.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return Admin{}, fmt.Errorf("parse admin updated_at: %w", err)
	}
	return a, nil
}
