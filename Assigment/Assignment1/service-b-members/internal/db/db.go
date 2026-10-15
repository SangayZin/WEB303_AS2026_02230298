package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"google.golang.org/protobuf/types/known/timestamppb"

	membersv1 "github.com/sangay/web303/proto/members/v1"
)

var ErrNotFound = errors.New("member not found")

type Store struct{ db *sql.DB }

func New(dsn string) (*Store, error) {
	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, err
	}
	return &Store{db: sqlDB}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Create(ctx context.Context, m *membersv1.Member) (*membersv1.Member, error) {
	if m.Id == "" {
		m.Id = uuid.NewString()
	}
	var joined time.Time
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO members (id, name, email, phone, plan)
		 VALUES ($1,$2,$3,$4,$5)
		 RETURNING joined_at`,
		m.Id, m.Name, m.Email, m.Phone, m.Plan,
	).Scan(&joined)
	if err != nil {
		return nil, err
	}
	m.JoinedAt = timestamppb.New(joined)
	return m, nil
}

func (s *Store) Get(ctx context.Context, id string) (*membersv1.Member, error) {
	var (
		m      membersv1.Member
		joined time.Time
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, email, phone, plan, joined_at FROM members WHERE id=$1`, id,
	).Scan(&m.Id, &m.Name, &m.Email, &m.Phone, &m.Plan, &joined)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	m.JoinedAt = timestamppb.New(joined)
	return &m, nil
}

func (s *Store) Update(ctx context.Context, m *membersv1.Member) (*membersv1.Member, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE members SET name=$1,email=$2,phone=$3,plan=$4,updated_at=NOW() WHERE id=$5`,
		m.Name, m.Email, m.Phone, m.Plan, m.Id,
	)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrNotFound
	}
	return s.Get(ctx, m.Id)
}

func (s *Store) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM members WHERE id=$1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) List(ctx context.Context, pageSize int32, offset int) ([]*membersv1.Member, error) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, email, phone, plan, joined_at
		 FROM members ORDER BY joined_at DESC LIMIT $1 OFFSET $2`,
		pageSize, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*membersv1.Member
	for rows.Next() {
		var (
			m      membersv1.Member
			joined time.Time
		)
		if err := rows.Scan(&m.Id, &m.Name, &m.Email, &m.Phone, &m.Plan, &joined); err != nil {
			return nil, err
		}
		m.JoinedAt = timestamppb.New(joined)
		out = append(out, &m)
	}
	return out, rows.Err()
}
