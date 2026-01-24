package queries

import (
	"database/sql"
	"errors"
	"testing"

	"AmHughesAbsalom/GO_CODE_SAMPLE.git/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type PlayoffsTestSuite struct {
	suite.Suite
	db   *sqlx.DB
	mock sqlmock.Sqlmock
	conn *PlayoffsDBConnection
}

// RUNS BEFORE EACH TEST
func (suite *PlayoffsTestSuite) SetupTest() {
	mockDB, mock, err := sqlmock.New()
	require.NoError(suite.T(), err)

	suite.db = sqlx.NewDb(mockDB, "sqlmock")
	suite.mock = mock
	suite.conn = &PlayoffsDBConnection{DB: suite.db}
}

// runs after each test
func (suite *PlayoffsTestSuite) TearDownTest() {
	suite.db.Close()
}

// TESTING CREATE PLAYOFFS FUNCTIONALITY
func (suite *PlayoffsTestSuite) TestCreatePlayoffs_SeasonAlreadyExists() {
	season := "2023-2024"
	conferences := []string{"East", "West"}
	limit := 8

	suite.mock.ExpectBegin()
	suite.mock.ExpectQuery(`SELECT COUNT\(\*\) AS count FROM playoffs WHERE season = \$1`).
		WithArgs(season).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	suite.mock.ExpectRollback()

	err := suite.conn.CreatePlayoffs(conferences, season, limit)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "this season already exists")
	assert.NoError(suite.T(), suite.mock.ExpectationsWereMet())
}

// TESTING INVALID NUMBER OF CONFERENCES
func (suite *PlayoffsTestSuite) TestCreatePlayoffs_InvalidNumberOfConferences() {
	season := "2023-2024"
	conferences := []string{"East", "West", "North"}
	limit := 8

	suite.mock.ExpectBegin()
	suite.mock.ExpectQuery(`SELECT COUNT\(\*\) AS count FROM playoffs WHERE season = \$1`).
		WithArgs(season).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	suite.mock.ExpectRollback()

	err := suite.conn.CreatePlayoffs(conferences, season, limit)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid number of conferences")
	assert.NoError(suite.T(), suite.mock.ExpectationsWereMet())
}

func (suite *PlayoffsTestSuite) TestCreatePlayoffs_OneConference_Success() {
	season := "2023-2024"
	conferences := []string{"Main"}
	limit := 4

	teamID1 := uuid.New()
	teamID2 := uuid.New()
	teamID3 := uuid.New()
	teamID4 := uuid.New()

	suite.mock.ExpectBegin()
	suite.mock.ExpectQuery(`SELECT COUNT\(\*\) AS count FROM playoffs WHERE season = \$1`).
		WithArgs(season).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// Mock standings query
	rows := sqlmock.NewRows([]string{
		"team_id", "team_name", "team_pic_url", "conference", "season", "pts", "position",
	}).
		AddRow(teamID1, "Team1", "url1", "Main", season, 100, 1).
		AddRow(teamID2, "Team2", "url2", "Main", season, 90, 2).
		AddRow(teamID3, "Team3", "url3", "Main", season, 80, 3).
		AddRow(teamID4, "Team4", "url4", "Main", season, 70, 4)

	suite.mock.ExpectQuery(`SELECT \*, RANK\(\)`).
		WithArgs("Main", season, limit).
		WillReturnRows(rows)

	// Round 1: 2 matchups, each with 3 games (best-of-3)
	// 2 matchups × 3 games = 6 inserts with full team details (13 args each)
	for i := 0; i < 6; i++ {
		suite.mock.ExpectExec(`INSERT INTO playoffs`).
			WithArgs(
				sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
				sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
				sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
				season,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}

	// Round 2 (Finals): 1 matchup

	suite.mock.ExpectExec(`INSERT INTO playoffs`).
		WithArgs(
			sqlmock.AnyArg(), sqlmock.AnyArg(), "FINAL", "1", sqlmock.AnyArg(), sqlmock.AnyArg(),
			season,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	suite.mock.ExpectCommit()

	err := suite.conn.CreatePlayoffs(conferences, season, limit)

	assert.NoError(suite.T(), err)
	assert.NoError(suite.T(), suite.mock.ExpectationsWereMet())
}

// TestCreatePlayoffs_TwoConferences_InsufficientTeams tests insufficient teams scenario
func (suite *PlayoffsTestSuite) TestCreatePlayoffs_TwoConferences_InsufficientTeams() {
	season := "2023-2024"
	conferences := []string{"East", "West"}
	limit := 8

	teamID1 := uuid.New()

	suite.mock.ExpectBegin()
	suite.mock.ExpectQuery(`SELECT COUNT\(\*\) AS count FROM playoffs WHERE season = \$1`).
		WithArgs(season).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// Mock insufficient teams for East
	rows := sqlmock.NewRows([]string{
		"team_id", "team_name", "team_pic_url", "conference", "season", "pts", "position",
	}).AddRow(teamID1, "Team1", "url1", "East", season, 100, 1)

	suite.mock.ExpectQuery(`SELECT \*, RANK\(\)`).
		WithArgs("East", season, limit).
		WillReturnRows(rows)

	// Mock insufficient teams for West (empty result)
	westRows := sqlmock.NewRows([]string{
		"team_id", "team_name", "team_pic_url", "conference", "season", "pts", "position",
	})

	suite.mock.ExpectQuery(`SELECT \*, RANK\(\)`).
		WithArgs("West", season, limit).
		WillReturnRows(westRows)

	suite.mock.ExpectRollback()

	err := suite.conn.CreatePlayoffs(conferences, season, limit)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "has less qualified teams")
	assert.NoError(suite.T(), suite.mock.ExpectationsWereMet())
}

// TestListPlayoffs_Success tests successful listing of playoffs
func (suite *PlayoffsTestSuite) TestListPlayoffs_Success() {
	season := "2023-2024"
	playoffsID := uuid.New()
	teamID1 := uuid.New()
	teamID2 := uuid.New()

	// Mock rounds query
	roundsRows := sqlmock.NewRows([]string{"fixture_round"}).
		AddRow(1).
		AddRow(2)

	suite.mock.ExpectQuery(`SELECT fixture_round FROM playoffs`).
		WithArgs(season).
		WillReturnRows(roundsRows)

	// Mock game count for round 1
	countRows1 := sqlmock.NewRows([]string{"fixture_round", "game_count"}).
		AddRow(1, "1")

	suite.mock.ExpectQuery(`SELECT fixture_round, game_count`).
		WithArgs(season, 1).
		WillReturnRows(countRows1)

	// Mock playoffs for round 1, game 1
	playoffsRows1 := sqlmock.NewRows([]string{
		"playoffs_id", "fixture_round", "game_count", "game_round",
		"home_team_id", "home_team_name", "home_team_url", "players_in_home_id",
		"away_team_id", "away_team_name", "away_team_url", "players_in_away_id",
		"season", "winner",
	}).AddRow(
		playoffsID, 1, "1", "1",
		teamID1, "Team1", "url1", uuid.New(),
		teamID2, "Team2", "url2", uuid.New(),
		season, nil,
	)

	suite.mock.ExpectQuery(`SELECT \* FROM playoffs WHERE season`).
		WithArgs(season, 1, "1").
		WillReturnRows(playoffsRows1)

	// Mock game count for round 2
	countRows2 := sqlmock.NewRows([]string{"fixture_round", "game_count"}).
		AddRow(2, "FINAL")

	suite.mock.ExpectQuery(`SELECT fixture_round, game_count`).
		WithArgs(season, 2).
		WillReturnRows(countRows2)

	// Mock playoffs for round 2 final
	playoffsRows2 := sqlmock.NewRows([]string{
		"playoffs_id", "fixture_round", "game_count", "game_round",
		"home_team_id", "home_team_name", "home_team_url", "players_in_home_id",
		"away_team_id", "away_team_name", "away_team_url", "players_in_away_id",
		"season", "winner",
	}).AddRow(
		playoffsID, 2, "FINAL", "1",
		teamID1, "Team1", "url1", uuid.New(),
		uuid.Nil, "", "", uuid.New(),
		season, nil,
	)

	suite.mock.ExpectQuery(`SELECT \* FROM playoffs WHERE season`).
		WithArgs(season, 2, "FINAL").
		WillReturnRows(playoffsRows2)

	result, err := suite.conn.ListPlayoffs(season)

	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 2)
	assert.NoError(suite.T(), suite.mock.ExpectationsWereMet())
}

// TestListPlayoffs_NoRows tests when no playoffs exist
func (suite *PlayoffsTestSuite) TestListPlayoffs_NoRows() {
	season := "2023-2024"

	suite.mock.ExpectQuery(`SELECT fixture_round FROM playoffs`).
		WithArgs(season).
		WillReturnError(sql.ErrNoRows)

	result, err := suite.conn.ListPlayoffs(season)

	assert.NoError(suite.T(), err)
	assert.Empty(suite.T(), result)
	assert.NoError(suite.T(), suite.mock.ExpectationsWereMet())
}

// TestUpdatePlayoffs_Success tests successful playoffs update
func (suite *PlayoffsTestSuite) TestUpdatePlayoffs_Success() {
	playoffsID := uuid.New()
	homeTeamID := uuid.New()
	awayTeamID := uuid.New()
	season := "2023-2024"

	playoffs := PlayoffsModelReqQuery{
		PlayoffsId:   playoffsID,
		FixtureRound: 1,
		GameCount:    "1",
		GameRound:    "1",
		HomeTeamId:   homeTeamID,
		HomeTeamName: "Team1",
		HomeTeamURL:  "url1",
		AwayTeamId:   awayTeamID,
		AwayTeamName: "Team2",
		AwayTeamURL:  "url2",
		Season:       season,
		Winner:       homeTeamID,
	}

	// Beginning transaction
	suite.mock.ExpectBegin()

	// 1. UPDATING winner
	suite.mock.ExpectExec(`UPDATE playoffs SET winner = \$1 WHERE playoffs_id = \$2`).
		WithArgs(homeTeamID, playoffsID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 2. SELECTING home winner (returns 2 rows)
	suite.mock.ExpectQuery(`SELECT \* FROM playoffs WHERE winner = \$1 AND fixture_round = \$2 AND game_count = \$3`).
		WithArgs(homeTeamID, 1, "1").
		WillReturnRows(sqlmock.NewRows([]string{"playoffs_id", "winner", "fixture_round", "game_count", "home_team_id", "home_team_name", "home_team_url", "away_team_id", "away_team_name", "away_team_url", "season"}).
			AddRow(playoffsID, homeTeamID, 1, "1", homeTeamID, "Team1", "url1", awayTeamID, "Team2", "url2", season).
			AddRow(uuid.New(), homeTeamID, 1, "2", homeTeamID, "Team1", "url1", uuid.New(), "Team3", "url3", season))

	// 3. SELECTING away winner (returns 0 rows)
	suite.mock.ExpectQuery(`SELECT \* FROM playoffs WHERE winner = \$1 AND fixture_round = \$2 AND game_count = \$3`).
		WithArgs(awayTeamID, 1, "1").
		WillReturnRows(sqlmock.NewRows([]string{"playoffs_id"}))

	// 4. SELECTING current round play counts (returns 4 rows for non-finals scenario)
	suite.mock.ExpectQuery(`SELECT fixture_round, game_count FROM playoffs WHERE season = \$1 AND fixture_round = \$2 GROUP BY fixture_round, game_count ORDER BY`).
		WithArgs(season, 1).
		WillReturnRows(sqlmock.NewRows([]string{"fixture_round", "game_count"}).
			AddRow(1, "1").
			AddRow(1, "2").
			AddRow(1, "3").
			AddRow(1, "4"))

	// 5. SELECTing next round play counts
	suite.mock.ExpectQuery(`SELECT fixture_round, game_count FROM playoffs WHERE season = \$1 AND fixture_round = \$2 GROUP BY fixture_round, game_count ORDER BY`).
		WithArgs(season, 2).
		WillReturnRows(sqlmock.NewRows([]string{"fixture_round", "game_count"}).
			AddRow(2, "1").
			AddRow(2, "2"))

	// 6. UPDATING next round home team
	suite.mock.ExpectExec(`UPDATE playoffs SET\s+home_team_id = \$1, home_team_name = \$2, home_team_url = \$3 WHERE season = \$4 AND fixture_round = \$5 AND game_count = \$6`).
		WithArgs(homeTeamID, "Team1", "url1", season, 2, "1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// COMMITTING TRANSACTION
	suite.mock.ExpectCommit()

	err := suite.conn.UpdatePlayoffs(playoffsID, playoffs)

	assert.NoError(suite.T(), err)
	assert.NoError(suite.T(), suite.mock.ExpectationsWereMet())
}

// TestUpdatePlayoffs_NoRowsAffected tests when no rows are updated
func (suite *PlayoffsTestSuite) TestUpdatePlayoffs_NoRowsAffected() {
	playoffsID := uuid.New()
	homeTeamID := uuid.New()
	season := "2023-2024"

	playoffs := PlayoffsModelReqQuery{
		PlayoffsId:   playoffsID,
		FixtureRound: 1,
		GameCount:    "1",
		Winner:       homeTeamID,
		Season:       season,
	}

	suite.mock.ExpectBegin()
	suite.mock.ExpectExec(`UPDATE playoffs SET winner = \$1 WHERE playoffs_id = \$2`).
		WithArgs(homeTeamID, playoffsID).
		WillReturnResult(sqlmock.NewResult(0, 0))
	suite.mock.ExpectRollback()

	err := suite.conn.UpdatePlayoffs(playoffsID, playoffs)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "failed to update the requested row")
	assert.NoError(suite.T(), suite.mock.ExpectationsWereMet())
}

// TestUpdatePlayoffsToNull_Success tests successful null update
func (suite *PlayoffsTestSuite) TestUpdatePlayoffsToNull_Success() {
	playoffsID := uuid.New()
	teamID := uuid.New()
	season := "2023-2024"
	round := 1

	suite.mock.ExpectBegin()

	suite.mock.ExpectExec(`UPDATE playoffs SET winner = \$1 WHERE playoffs_id = \$2`).
		WithArgs(nil, playoffsID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	homeWinnerRows := sqlmock.NewRows([]string{"winner"})
	suite.mock.ExpectQuery(`SELECT winner FROM playoffs WHERE winner = \$1`).
		WithArgs(teamID, season, round).
		WillReturnRows(homeWinnerRows)

	awayWinnerRows := sqlmock.NewRows([]string{"winner"})
	suite.mock.ExpectQuery(`SELECT winner FROM playoffs WHERE winner = \$1`).
		WithArgs(teamID, season, round).
		WillReturnRows(awayWinnerRows)

	suite.mock.ExpectCommit()

	err := suite.conn.UpdatePlayoffsToNull(playoffsID, round, teamID, season)

	assert.NoError(suite.T(), err)
	assert.NoError(suite.T(), suite.mock.ExpectationsWereMet())
}

// TestUpdatePlayoffsToNull_WithNextRoundUpdate tests null update with next round changes
func (suite *PlayoffsTestSuite) TestUpdatePlayoffsToNull_WithNextRoundUpdate() {
	playoffsID := uuid.New()
	teamID := uuid.New()
	season := "2023-2024"
	round := 1

	suite.mock.ExpectBegin()

	suite.mock.ExpectExec(`UPDATE playoffs SET winner = \$1 WHERE playoffs_id = \$2`).
		WithArgs(nil, playoffsID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// First SELECT (home)
	suite.mock.ExpectQuery(
		`SELECT winner FROM playoffs WHERE winner = \$1 AND season = \$2 AND fixture_round = \$3`,
	).
		WithArgs(teamID, season, round).
		WillReturnRows(sqlmock.NewRows([]string{"winner"}).AddRow(teamID))

	// Second SELECT (away)
	suite.mock.ExpectQuery(
		`SELECT winner FROM playoffs WHERE winner = \$1 AND season = \$2 AND fixture_round = \$3`,
	).
		WithArgs(teamID, season, round).
		WillReturnRows(sqlmock.NewRows([]string{"winner"})) // empty result set

	suite.mock.ExpectExec(`UPDATE playoffs SET home_team_id`).
		WithArgs(nil, nil, nil, nil, teamID, round+1, season).
		WillReturnResult(sqlmock.NewResult(1, 1))

	suite.mock.ExpectCommit()

	err := suite.conn.UpdatePlayoffsToNull(playoffsID, round, teamID, season)

	assert.NoError(suite.T(), err)
	assert.NoError(suite.T(), suite.mock.ExpectationsWereMet())
}

// TestDeletePlayoffs_Success tests successful deletion
func (suite *PlayoffsTestSuite) TestDeletePlayoffs_Success() {
	season := "2023-2024"

	suite.mock.ExpectExec(`DELETE FROM playoffs WHERE season = \$1`).
		WithArgs(season).
		WillReturnResult(sqlmock.NewResult(0, 5))

	err := suite.conn.DeletePlayoffs(season)

	assert.NoError(suite.T(), err)
	assert.NoError(suite.T(), suite.mock.ExpectationsWereMet())
}

// TestDeletePlayoffs_NoRowsDeleted tests deletion when no records exist
func (suite *PlayoffsTestSuite) TestDeletePlayoffs_NoRowsDeleted() {
	season := "2023-2024"

	suite.mock.ExpectExec(`DELETE FROM playoffs WHERE season = \$1`).
		WithArgs(season).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := suite.conn.DeletePlayoffs(season)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "could not delete the requested records")
	assert.NoError(suite.T(), suite.mock.ExpectationsWereMet())
}

// TestDeletePlayoffs_DatabaseError tests deletion with database error
func (suite *PlayoffsTestSuite) TestDeletePlayoffs_DatabaseError() {
	season := "2023-2024"

	suite.mock.ExpectExec(`DELETE FROM playoffs WHERE season = \$1`).
		WithArgs(season).
		WillReturnError(errors.New("database error"))

	err := suite.conn.DeletePlayoffs(season)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "database error")
	assert.NoError(suite.T(), suite.mock.ExpectationsWereMet())
}

// TESTS THE reverseTeam HELPER FUNCTION
func TestReverseTeam(t *testing.T) {
	team1 := models.StandingsModel{TeamName: "Team1"}
	team2 := models.StandingsModel{TeamName: "Team2"}
	team3 := models.StandingsModel{TeamName: "Team3"}

	input := []models.StandingsModel{team1, team2, team3}
	result := reverseTeam(input)

	assert.Equal(t, "Team3", result[0].TeamName)
	assert.Equal(t, "Team2", result[1].TeamName)
	assert.Equal(t, "Team1", result[2].TeamName)

	// Ensuring original slice is not modified
	assert.Equal(t, "Team1", input[0].TeamName)
	assert.Equal(t, "Team2", input[1].TeamName)
	assert.Equal(t, "Team3", input[2].TeamName)
}

// runs the test suite
func TestPlayoffsTestSuite(t *testing.T) {
	suite.Run(t, new(PlayoffsTestSuite))
}
