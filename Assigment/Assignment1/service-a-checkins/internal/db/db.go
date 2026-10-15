package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"google.golang.org/protobuf/types/known/timestamppb"

	checkinsv1 "github.com/sangay/web303/proto/checkins/v1"
)

var ErrNotFound = errors.New("check-in not found")

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

// Create inserts a new check-in.
func (s *Store) Create(ctx context.Context, c *checkinsv1.CheckIn) (*checkinsv1.CheckIn, error) {
	if c.Id == "" {
		c.Id = uuid.NewString()
	}
	var checkInAt time.Time
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO checkins (id, member_id, notes)
		 VALUES ($1,$2,$3)
		 RETURNING check_in_at`,
		c.Id, c.MemberId, nullableString(c.Notes),
	).Scan(&checkInAt)
	if err != nil {
		return nil, err
	}
	c.CheckInAt = timestamppb.New(checkInAt)
	return c, nil
}

// Get fetches a check-in by ID.
func (s *Store) Get(ctx context.Context, id string) (*checkinsv1.CheckIn, error) {
	var (
		c         checkinsv1.CheckIn
		checkInAt time.Time
		notes     sql.NullString
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT id, member_id, check_in_at, notes FROM checkins WHERE id=$1`, id,
	).Scan(&c.Id, &c.MemberId, &checkInAt, &notes)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	c.CheckInAt = timestamppb.New(checkInAt)
	if notes.Valid {
		c.Notes = notes.String
	}
	return &c, nil
}

// Update modifies the notes on an existing check-in.
func (s *Store) Update(ctx context.Context, id, notes string) (*checkinsv1.CheckIn, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE checkins SET notes=$1, updated_at=NOW() WHERE id=$2`,
		nullableString(notes), id,
	)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrNotFound
	}
	return s.Get(ctx, id)
}

// Delete removes a check-in.
func (s *Store) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM checkins WHERE id=$1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// List returns check-ins, optionally filtered by member_id.
func (s *Store) List(ctx context.Context, memberID string, pageSize int32, offset int) ([]*checkinsv1.CheckIn, error) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	var (
		rows *sql.Rows
		err  error
	)
	if memberID != "" {
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, member_id, check_in_at, notes
			 FROM checkins WHERE member_id=$1
			 ORDER BY check_in_at DESC LIMIT $2 OFFSET $3`,
			memberID, pageSize, offset,
		)
	} else {
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, member_id, check_in_at, notes
			 FROM checkins
			 ORDER BY check_in_at DESC LIMIT $1 OFFSET $2`,
			pageSize, offset,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*checkinsv1.CheckIn
	for rows.Next() {
		var (
			c         checkinsv1.CheckIn
			checkInAt time.Time
			notes     sql.NullString
		)
		if err := rows.Scan(&c.Id, &c.MemberId, &checkInAt, &notes); err != nil {
			return nil, err
		}
		c.CheckInAt = timestamppb.New(checkInAt)
		if notes.Valid {
			c.Notes = notes.String
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

// nullableString converts an empty string to NULL for optional columns.
func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
