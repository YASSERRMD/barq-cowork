package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/barq-cowork/barq-cowork/internal/domain"
)

// ProviderProfileStore implements persistence for domain.ProviderProfile.
type ProviderProfileStore struct{ db *sql.DB }

// NewProviderProfileStore returns a new ProviderProfileStore.
func NewProviderProfileStore(db *sql.DB) *ProviderProfileStore {
	return &ProviderProfileStore{db: db}
}

const providerProfileCols = `id, name, provider_name, base_url, api_key_env,
                              model, timeout_sec, is_default, created_at, updated_at`

func scanProviderProfile(row interface{ Scan(...any) error }) (*domain.ProviderProfile, error) {
	var p domain.ProviderProfile
	var createdAt, updatedAt string
	var isDefault int
	if err := row.Scan(
		&p.ID, &p.Name, &p.ProviderName, &p.BaseURL, &p.APIKeyEnv,
		&p.Model, &p.TimeoutSec, &isDefault, &createdAt, &updatedAt,
	); err != nil {
		return nil, err
	}
	p.IsDefault = isDefault == 1
	p.CreatedAt = parseTime(createdAt)
	p.UpdatedAt = parseTime(updatedAt)
	return &p, nil
}

// Create inserts a new provider profile.
func (s *ProviderProfileStore) Create(ctx context.Context, p *domain.ProviderProfile) error {
	isDefault := 0
	if p.IsDefault {
		isDefault = 1
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO provider_profiles
		 (id, name, provider_name, base_url, api_key_env, model, timeout_sec, is_default, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Name, p.ProviderName, p.BaseURL, p.APIKeyEnv,
		p.Model, p.TimeoutSec, isDefault,
		formatTime(p.CreatedAt), formatTime(p.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("provider profile create: %w", err)
	}
	return nil
}

// GetByID retrieves a profile by ID.
func (s *ProviderProfileStore) GetByID(ctx context.Context, id string) (*domain.ProviderProfile, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+providerProfileCols+` FROM provider_profiles WHERE id=?`, id)
	p, err := scanProviderProfile(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return p, err
}

// List returns all provider profiles ordered by name.
func (s *ProviderProfileStore) List(ctx context.Context) ([]*domain.ProviderProfile, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+providerProfileCols+` FROM provider_profiles ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("provider profile list: %w", err)
	}
	defer rows.Close()

	var out []*domain.ProviderProfile
	for rows.Next() {
		p, err := scanProviderProfile(rows)
		if err != nil {
			return nil, fmt.Errorf("provider profile scan: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Update replaces all mutable fields of a provider profile.
func (s *ProviderProfileStore) Update(ctx context.Context, p *domain.ProviderProfile) error {
	isDefault := 0
	if p.IsDefault {
		isDefault = 1
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE provider_profiles
		 SET name=?, provider_name=?, base_url=?, api_key_env=?, model=?,
		     timeout_sec=?, is_default=?, updated_at=?
		 WHERE id=?`,
		p.Name, p.ProviderName, p.BaseURL, p.APIKeyEnv, p.Model,
		p.TimeoutSec, isDefault, formatTime(p.UpdatedAt), p.ID,
	)
	if err != nil {
		return fmt.Errorf("provider profile update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Delete removes a profile by ID.
func (s *ProviderProfileStore) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM provider_profiles WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("provider profile delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
