package domain

import (
	"time"

	"github.com/google/uuid"
)

// TeamMemberStatus enumerates membership states. The owner of a team is its CreatedBy.
type TeamMemberStatus string

const (
	TeamMemberActive  TeamMemberStatus = "active"
	TeamMemberExpired TeamMemberStatus = "expired"
	TeamMemberBlock   TeamMemberStatus = "block"
)

func ValidTeamMemberStatus(s string) bool {
	switch TeamMemberStatus(s) {
	case TeamMemberActive, TeamMemberExpired, TeamMemberBlock:
		return true
	}
	return false
}

// TeamMaster is a team. Membership rows live in TeamDetail.
type TeamMaster struct {
	Base
	Name    string       `json:"name"`
	Members []TeamDetail `json:"members,omitempty"`
}

func (TeamMaster) TableName() string { return "team_masters" }

// TeamDetail is a membership of a user within a team. User is populated by the
// repository via a join; it is not a stored column.
type TeamDetail struct {
	Base
	IDTeamMaster uuid.UUID        `json:"idTeamMaster"`
	IDUser       uuid.UUID        `json:"idUser"`
	ExpiredAt    *time.Time       `json:"expiredAt,omitempty"`
	Status       TeamMemberStatus `json:"status"`
	User         *User            `json:"user,omitempty"`
}

func (TeamDetail) TableName() string { return "team_details" }
