package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/178inaba/today-duty-bot/entity"
	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

type DutyHistoryRepository struct {
	db *sqlx.DB
}

func NewDutyHistoryRepository(db *sqlx.DB) *DutyHistoryRepository {
	return &DutyHistoryRepository{db: db}
}

func (r *DutyHistoryRepository) Create(ctx context.Context, memberID int, assignedOn time.Time) error {
	query, args, err := sq.
		Insert("duty_histories").
		Columns(
			"member_id",
			"assigned_on",
			"is_skip",
		).
		Values(
			memberID,
			assignedOn,
			false,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("to sql: %w", err)
	}

	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("exec: %w", err)
	}

	return nil
}

func (r *DutyHistoryRepository) Skip(ctx context.Context, id int) error {
	query, args, err := sq.
		Update("duty_histories").
		Set("is_skip", true).
		Where(sq.And{
			sq.Eq{"id": id},
			sq.Eq{"deleted_at": nil},
		}).
		OrderBy("id DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return fmt.Errorf("to sql: %w", err)
	}

	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("exec: %w", err)
	}

	return nil
}

func (r *DutyHistoryRepository) Delete(ctx context.Context, id int) error {
	query, args, err := sq.
		Update("duty_histories").
		Set("deleted_at", time.Now()).
		Where(sq.And{
			sq.Eq{"id": id},
			sq.Eq{"deleted_at": nil},
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("to sql: %w", err)
	}

	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("exec: %w", err)
	}

	return nil
}

func (r *DutyHistoryRepository) GetSkipped(ctx context.Context) (*entity.DutyHistory, error) {
	b := sq.
		Select(
			"id",
			"member_id",
			"assigned_on",
			"is_skip",
		).
		From("duty_histories").
		Where(sq.And{
			sq.Eq{"is_skip": true},
			sq.Eq{"deleted_at": nil},
		}).
		OrderBy("id").
		Limit(1)

	query, args, err := b.ToSql()
	if err != nil {
		return nil, fmt.Errorf("to sql: %w", err)
	}

	var dh entity.DutyHistory
	if err := r.db.GetContext(ctx, &dh, query, args...); errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("get: %w", err)
	}

	return &dh, nil
}

func (r *DutyHistoryRepository) GetLatestDutyMember(ctx context.Context) (*entity.DutyHistory, error) {
	b := sq.
		Select(
			"id",
			"member_id",
			"assigned_on",
			"is_skip",
		).
		From("duty_histories").
		Where(sq.And{
			sq.Eq{"is_skip": false},
			sq.Eq{"deleted_at": nil},
		}).
		OrderBy("id DESC").
		Limit(1)

	query, args, err := b.ToSql()
	if err != nil {
		return nil, fmt.Errorf("to sql: %w", err)
	}

	var dh entity.DutyHistory
	if err := r.db.GetContext(ctx, &dh, query, args...); errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("get: %w", err)
	}

	return &dh, nil
}
