package models

import "github.com/google/uuid"

type StandingsModel struct {
	Operation     string     `json:"operation"`
	StandingsId   uuid.UUID  `db:"standings_id" json:"standingsId"`
	TeamId        *uuid.UUID `db:"team_id" json:"teamId"`
	Position      int        `db:"position" json:"position"`
	TeamName      string     `db:"team_name" json:"teamName"`
	Acronym       string     `db:"acronym" json:"acronym"`
	TeamPicUrl    *string    `db:"team_pic_url" json:"teamPicUrl"`
	Gp            int        `db:"gp" json:"gp"`
	W             int        `db:"w" json:"w"`
	L             int        `db:"l" json:"l"`
	WinPercentage float64    `db:"win_percentage" json:"winPercentage"`
	Gf            int        `db:"gf" json:"gf"`
	Pts           int        `db:"pts" json:"pts"`
	Conference    string     `db:"conference" json:"conference"`
	Season        string     `db:"season" json:"season"`
}
