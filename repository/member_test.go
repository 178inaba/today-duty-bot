package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/178inaba/today-duty-bot/entity"
	sq "github.com/Masterminds/squirrel"
)

func TestMemberRepository_GetNext(t *testing.T) {
	ctx := context.Background()
	r := NewMemberRepository(testDB)

	count := 5
	members := make([]*entity.Member, count)
	for i := 0; i < count; i++ {
		members[i] = &entity.Member{
			Name:    fmt.Sprintf("Test Name: %d", i+1),
			SlackID: fmt.Sprintf("test_slack_id_%d", i+1),
		}
	}
	if err := r.bulkCreate(ctx, members); err != nil {
		t.Fatalf("Should not be fail: %v.", err)
	}
	t.Cleanup(func() {
		if _, err := testDB.ExecContext(ctx, "TRUNCATE members"); err != nil {
			t.Fatalf("Fail cleanup: %v.", err)
		}
	})

	gotMember, err := r.GetNext(ctx, 1)
	if err != nil {
		t.Fatalf("Should not be fail: %v.", err)
	}
	if got, want := gotMember.ID, 2; got != want {
		t.Fatalf("ID is %d, but want %d.", got, want)
	}
	if got, want := gotMember.Name, "Test Name: 2"; got != want {
		t.Fatalf("Name is %q, but want %q.", got, want)
	}
	if got, want := gotMember.SlackID, "test_slack_id_2"; got != want {
		t.Fatalf("SlackID is %q, but want %q.", got, want)
	}
}

// For test use only.
func (r *MemberRepository) bulkCreate(ctx context.Context, members []*entity.Member) error {
	ib := sq.
		Insert("members").
		Columns(
			"name",
			"slack_id",
		)

	for _, m := range members {
		ib = ib.Values(
			m.Name,
			m.SlackID,
		)
	}

	query, args, err := ib.ToSql()
	if err != nil {
		return fmt.Errorf("to sql: %w", err)
	}

	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("exec: %w", err)
	}

	return nil
}
