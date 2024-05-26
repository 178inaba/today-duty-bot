package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/178inaba/today-duty-bot/entity"
	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

type MemberRepository struct {
	db *sqlx.DB
}

func NewMemberRepository(db *sqlx.DB) *MemberRepository {
	return &MemberRepository{db: db}
}

func (r *MemberRepository) Get(ctx context.Context, id int) (*entity.Member, error) {
	b := sq.
		Select(
			"id",
			"name",
			"slack_id",
		).
		From("members").
		Where(sq.And{
			sq.Eq{"id": id},
			sq.Eq{"deleted_at": nil},
		})

	query, args, err := b.ToSql()
	if err != nil {
		return nil, fmt.Errorf("to sql: %w", err)
	}

	var m entity.Member
	if err := r.db.GetContext(ctx, &m, query, args...); errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("get: %w", err)
	}

	return &m, nil
}

func (r *MemberRepository) GetNext(ctx context.Context, id int) (*entity.Member, error) {
	b := sq.
		Select(
			"id",
			"name",
			"slack_id",
		).
		From("members").
		Where(sq.And{
			sq.Gt{"id": id},
			sq.Eq{"deleted_at": nil},
		}).
		OrderBy("id").
		Limit(1)

	query, args, err := b.ToSql()
	if err != nil {
		return nil, fmt.Errorf("to sql: %w", err)
	}

	var m entity.Member
	if err := r.db.GetContext(ctx, &m, query, args...); errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("get: %w", err)
	}

	return &m, nil
}
