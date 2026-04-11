package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/barq-cowork/barq-cowork/internal/domain"
)

// SkillStore implements domain.SkillRepository on SQLite.
type SkillStore struct{ db *sql.DB }

// NewSkillStore returns a new SkillStore.
func NewSkillStore(db *sql.DB) *SkillStore { return &SkillStore{db: db} }

const skillCols = `id, name, kind, description, output_mime_type, output_file_ext,
                   prompt_template, built_in, enabled, tags, input_mime_types,
                   created_at, updated_at`

func scanSkill(row interface{ Scan(...any) error }) (*domain.SkillSpec, error) {
	var s domain.SkillSpec
	var builtIn, enabled int
	var tags, inputMimes, createdAt, updatedAt string

	if err := row.Scan(
		&s.ID, &s.Name, &s.Kind, &s.Description,
		&s.OutputMimeType, &s.OutputFileExt, &s.PromptTemplate,
		&builtIn, &enabled, &tags, &inputMimes,
		&createdAt, &updatedAt,
	); err != nil {
		return nil, err
	}
	s.BuiltIn = builtIn == 1
	s.Enabled = enabled == 1
	if tags != "" {
		s.Tags = strings.Split(tags, ",")
	}
	if inputMimes != "" {
		s.InputMimeTypes = strings.Split(inputMimes, ",")
	}
	return &s, nil
}

func (s *SkillStore) Create(ctx context.Context, sk *domain.SkillSpec) error {
	now := formatTime(time.Now().UTC())
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO skills (id,name,kind,description,output_mime_type,output_file_ext,
		 prompt_template,built_in,enabled,tags,input_mime_types,created_at,updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		sk.ID, sk.Name, sk.Kind, sk.Description, sk.OutputMimeType, sk.OutputFileExt,
		sk.PromptTemplate, boolInt(sk.BuiltIn), boolInt(sk.Enabled),
		strings.Join(sk.Tags, ","), strings.Join(sk.InputMimeTypes, ","),
		now, now,
	)
	if err != nil {
		return fmt.Errorf("skill create: %w", err)
	}
	return nil
}

func (s *SkillStore) GetByID(ctx context.Context, id string) (*domain.SkillSpec, bool) {
	row := s.db.QueryRowContext(ctx, `SELECT `+skillCols+` FROM skills WHERE id=?`, id)
	sk, err := scanSkill(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false
	}
	if err != nil {
		return nil, false
	}
	return sk, true
}

func (s *SkillStore) List(ctx context.Context) ([]*domain.SkillSpec, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+skillCols+` FROM skills ORDER BY built_in DESC, name ASC`)
	if err != nil {
		return nil, fmt.Errorf("skill list: %w", err)
	}
	defer rows.Close()
	var out []*domain.SkillSpec
	for rows.Next() {
		sk, err := scanSkill(rows)
		if err != nil {
			return nil, fmt.Errorf("skill scan: %w", err)
		}
		out = append(out, sk)
	}
	return out, rows.Err()
}

func (s *SkillStore) Update(ctx context.Context, sk *domain.SkillSpec) error {
	now := formatTime(time.Now().UTC())
	res, err := s.db.ExecContext(ctx,
		`UPDATE skills SET name=?,kind=?,description=?,output_mime_type=?,output_file_ext=?,
		 prompt_template=?,enabled=?,tags=?,input_mime_types=?,updated_at=? WHERE id=?`,
		sk.Name, sk.Kind, sk.Description, sk.OutputMimeType, sk.OutputFileExt,
		sk.PromptTemplate, boolInt(sk.Enabled),
		strings.Join(sk.Tags, ","), strings.Join(sk.InputMimeTypes, ","),
		now, sk.ID,
	)
	if err != nil {
		return fmt.Errorf("skill update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *SkillStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM skills WHERE id=?`, id)
	return err
}

func boolInt(b bool) int {
	if b { return 1 }
	return 0
}
