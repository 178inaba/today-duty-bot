package entity

import "time"

type DutyHistory struct {
	ID         int       `db:"id"`
	MemberID   int       `db:"member_id"`
	AssignedOn time.Time `db:"assigned_on"`
	IsSkip     bool      `db:"is_skip"`
}

type Member struct {
	ID      int    `db:"id"`
	Name    string `db:"name"`
	SlackID string `db:"slack_id"`
}
