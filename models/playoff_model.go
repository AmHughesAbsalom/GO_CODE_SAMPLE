package models

import "github.com/google/uuid"

type PlayoffsModel struct {
	Operation       string     `db:"operation" json:"operation"`
	PlayoffsId      uuid.UUID  `db:"playoffs_id" json:"playoffsId"`
	FixtureRound    *int       `db:"fixture_round" json:"fixtureRound"`
	GameCount       *string    `db:"game_count" json:"gameCount"`
	GameRound       string     `db:"game_round" json:"gameRound"`
	HomeTeamId      *uuid.UUID `db:"home_team_id" json:"homeTeamId"`
	HomeTeamName    *string    `db:"home_team_name" json:"homeTeamName"`
	PlayersInHomeId uuid.UUID  `db:"players_in_home_id" json:"playersInHomeId"`
	AwayTeamId      *uuid.UUID `db:"away_team_id" json:"awayTeamId"`
	AwayTeamName    *string    `db:"away_team_name" json:"awayTeamName"`
	PlayersInAwayId uuid.UUID  `db:"players_in_away_id" json:"playersInAwayId"`
	Season          string     `db:"season" json:"season"`
	Winner          *uuid.UUID `db:"winner" json:"winner"`
	HomeTeamURL     *string    `db:"home_team_url" json:"homeTeamURL"`
	AwayTeamURL     *string    `db:"away_team_url" json:"awayTeamURL"`
}
type PlayoffsModelRes struct {
	Operation       string    `db:"operation" json:"operation"`
	PlayoffsId      uuid.UUID `db:"playoffs_id" json:"playoffsId"`
	FixtureRound    int       `db:"fixture_round" json:"fixtureRound"`
	GameCount       string    `db:"game_count" json:"gameCount"`
	GameRound       string    `db:"game_round" json:"gameRound"`
	HomeTeamId      uuid.UUID `db:"home_team_id" json:"homeTeamId"`
	HomeTeamName    string    `db:"home_team_name" json:"homeTeamName"`
	PlayersInHomeId uuid.UUID `db:"players_in_home_id" json:"playersInHomeId"`
	AwayTeamId      uuid.UUID `db:"away_team_id" json:"awayTeamId"`
	AwayTeamName    string    `db:"away_team_name" json:"awayTeamName"`
	PlayersInAwayId uuid.UUID `db:"players_in_away_id" json:"playersInAwayId"`
	Season          string    `db:"season" json:"season"`
	Winner          uuid.UUID `db:"winner" json:"winner"`
	HomeTeamURL     string    `db:"home_team_url" json:"homeTeamURL"`
	AwayTeamURL     string    `db:"away_team_url" json:"awayTeamURL"`
}
