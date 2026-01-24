package queries

import (
	"errors"
	"fmt"
	"log"
	"slices"

	"AmHughesAbsalom/GO_CODE_SAMPLE.git/models"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type PlayoffsDBConnection struct {
	*sqlx.DB
}

type Playoffs interface {
	CreatePlayoffs(conferences []string, season string, limit int) error
	ListPlayoffs(season string) ([]models.PlayoffsModel, error)
	UpdatePlayoffs(playoffsId uuid.UUID, playoffs models.PlayoffsModel) error
	DeletePlayoffs(season string) error
}

type seasonCount struct {
	count int `db:"count"`
}

func (p *PlayoffsDBConnection) CreatePlayoffs(conferences []string, season string, limit int) error {
	seasonCount := seasonCount{}
	query :=
		`
		SELECT COUNT(*) AS count FROM playoffs WHERE season = $1
		`
	tx, errTx := p.DB.Beginx()
	if errTx != nil {
		log.Println("error creating playoffs tx: ", errTx.Error())
		return errTx
	}
	defer func() {
		_ = tx.Rollback()

	}()

	err := tx.Get(&seasonCount.count, query, season)
	if err != nil {
		log.Println("error counting playoffs records: ", err.Error())
		return err
	}

	if seasonCount.count >= 1 {
		errC := errors.New("Cannot create the requested Playoffs of season " + season + ", this season already exists!")
		return errC
	}
	if len(conferences) != 1 && len(conferences) != 2 && len(conferences) != 4 && len(conferences) != 8 {
		errL := errors.New("invalid number of conferences for Playoffs generator. valid numbers: (1, 2, 4, 8)")
		return errL
	}

	// EXPECTED NUMBER OF CONFERENCES IS 1, 2, 4, OR 8 FOR THIS LEAGUE STRUCTURE SINCE
	// THE ANTICIPATED NUMBER OF TEAMS FOR THE BRACKET IS 16, 32, 64, OR 128 RESPECTIVELY.
	// THE NUMBER OF TEAMS PER CONFERENCE IS DERIVED FROM THE LIMIT PARAMETER.
	// E.G. IF THERE ARE 2 CONFERENCES AND LIMIT IS 8, THEN EACH CONFERENCE SHOULD HAVE 8 TEAMS,

	switch len(conferences) {

	case 1:
		var allTeams []models.StandingsModel
		var homeTeams []models.StandingsModel
		var awayTeams []models.StandingsModel

		query :=
			`
			SELECT *, 
			RANK() OVER(PARTITION BY conference ORDER BY pts desc) AS position 
			FROM standings
			WHERE conference = $1 AND season = $2
			LIMIT $3
		`
		errHt := tx.Select(&allTeams, query, conferences[0], season, limit)
		if errHt != nil {
			log.Println("error SELECTING allTeams; CASE = 1 ERROR: ", errHt)
			return errHt
		}
		partitionLimit := limit / 2
		if len(allTeams) >= partitionLimit {
			teams := allTeams[:partitionLimit]
			homeTeams = append(homeTeams, teams...)
		} else {
			teams := allTeams
			homeTeams = append(homeTeams, teams...)
		}

		if len(allTeams) >= partitionLimit*2 {
			teams := allTeams[partitionLimit : partitionLimit*2]
			awayTeams = append(awayTeams, teams...)
		} else if len(allTeams) > partitionLimit {
			teams := allTeams[partitionLimit:]
			awayTeams = append(awayTeams, teams...)
		}

		if len(homeTeams) < partitionLimit {
			err := errors.New(conferences[0] + "has less qualified teams of" + fmt.Sprint(len(homeTeams)) + " teams than the required number of " + fmt.Sprint(partitionLimit) + "teams")
			return err
		}
		if len(awayTeams) < partitionLimit {
			err := errors.New(conferences[0] + "has less qualified teams of" + fmt.Sprint(len(homeTeams)) + " teams than the required number of " + fmt.Sprint(partitionLimit) + "teams")
			return err
		}
		if len(homeTeams) != len(awayTeams) {
			err := errors.New("Invalid number of teams in the conference " + conferences[0] + ". The number of teams must be even to create home and away teams for the playoffs.")
			return err
		}
		reversedAwayTeams := reverseTeam(awayTeams)
		playoffsQuery :=
			`
		INSERT INTO playoffs 
		(
		playoffs_id, 
		fixture_round, 
		game_count, 
		game_round, 
		home_team_id, 
		home_team_name, 
		home_team_url, 
		players_in_home_id, 
		away_team_id, 
		away_team_name, 
		away_team_url, 
		players_in_away_id, 
		season)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		`
		// Insert AS THE FINAL if limit is 1
		if partitionLimit == 1 {
			playoffsId := uuid.New()
			playersInHomeId := uuid.New()
			playersInAwayId := uuid.New()
			h := homeTeams[0]
			a := awayTeams[0]
			_, err := tx.Exec(
				playoffsQuery,
				playoffsId,
				0000,
				"FINAL",
				"1",
				h.TeamId,
				h.TeamName,
				h.TeamPicUrl,
				playersInHomeId,
				a.TeamId,
				a.TeamName,
				a.TeamPicUrl,
				playersInAwayId,
				season,
			)
			if err != nil {
				log.Println("failed to insert FINAL: CASE 1: ", err.Error())
				return err
			}
		}

		count := len(homeTeams)
		fixtureRound := 1
		for len(homeTeams) > 1 {
			//
			for i := 0; i < len(homeTeams); i++ {
				// INSERTING THE BEST OF 3 GAMES FOR EVERY FIXTURE ROUND
				for inner := 0; inner < 3; inner++ {
					if fixtureRound == 1 {
						_, err := tx.Exec(
							playoffsQuery,
							uuid.New(),
							fixtureRound,
							i+1,
							inner+1,
							homeTeams[i].TeamId,
							homeTeams[i].TeamName,
							homeTeams[i].TeamPicUrl,
							uuid.New(),
							reversedAwayTeams[i].TeamId,
							reversedAwayTeams[i].TeamName,
							reversedAwayTeams[i].TeamPicUrl,
							uuid.New(),
							season,
						)
						if err != nil {
							log.Println("failed to INSERT playoffs records: ERROR CASE = 1, INNER LOOP: ", err.Error())
							return err
						}
					} else {
						playoffsIdNextRound := uuid.New()
						playersInHomeIdNextRound := uuid.New()
						playersInAwayIdNextRound := uuid.New()
						playoffsQueryNextRound :=
							`
								INSERT INTO playoffs 
								(playoffs_id, fixture_round, game_count, game_round,  players_in_home_id,  players_in_away_id, season)
								VALUES($1, $2, $3, $4, $5, $6, $7)
								`
						_, err := tx.Exec(
							playoffsQueryNextRound,
							playoffsIdNextRound,
							fixtureRound,
							i+1+count,
							inner+1,
							playersInHomeIdNextRound,
							playersInAwayIdNextRound,
							season,
						)
						if err != nil {
							log.Println("failed to INSERT playoffs records: ERROR CASE = 1, INNER LOOP fixtureRound > 1: ", err.Error())
							return err
						}
						// NEXT ROUND COUNT INCREMENT
						if inner == 2 && i == len(homeTeams)-1 {
							count = count + len(homeTeams)
						}

					}
				}
			}

			// REDUCING THE NUMBER OF TEAMS BY HALF FOR THE NEXT FIXTURE ROUND
			homeTeams = homeTeams[:len(homeTeams)/2]
			reversedAwayTeams = reversedAwayTeams[:len(reversedAwayTeams)/2]
			fixtureRound++

		}
		playoffsQueryFinal :=
			`
					INSERT INTO playoffs 
					(playoffs_id, fixture_round, game_count, game_round,  players_in_home_id,  players_in_away_id, season)
					VALUES($1, $2, $3, $4, $5, $6, $7)
					`
		_, err := tx.Exec(
			playoffsQueryFinal,
			uuid.New(),
			fixtureRound,
			"FINAL",
			"1",
			uuid.New(),
			uuid.New(),
			season,
		)
		if err != nil {
			log.Println("failed to INSERT playoffs records: ERROR CASE = 1, FINAL: ", err.Error())
			return err
		}

	case 2:
		var homeTeams []models.StandingsModel
		var awayTeams []models.StandingsModel
		query :=
			`
				SELECT *, 
				RANK() OVER(PARTITION BY conference ORDER BY pts desc) AS position 
				FROM standings
				WHERE conference = $1 AND season = $2
				LIMIT $3
			`
		errHt := tx.Select(&homeTeams, query, conferences[0], season, limit)
		if errHt != nil {
			log.Println("error SELECTING homeTeams; CASE = 2 ERROR: ", errHt)
			return errHt
		}
		errAt := tx.Select(&awayTeams, query, conferences[1], season, limit)
		if errAt != nil {
			log.Println("error SELECTING awayTeams; CASE = 2 ERROR: ", errAt)
			return errAt
		}
		if len(homeTeams) < limit {
			err := errors.New(conferences[0] + "has less qualified teams of" + fmt.Sprint(len(homeTeams)) + " teams than the required number of " + fmt.Sprint(limit) + "teams")
			return err
		}
		if len(awayTeams) < limit {
			err := errors.New(conferences[0] + "has less qualified teams of" + fmt.Sprint(len(homeTeams)) + " teams than the required number of " + fmt.Sprint(limit) + "teams")
			return err
		}
		if len(homeTeams) != len(awayTeams) {
			err := errors.New("Ivalid number of teams in the conferences " + conferences[0] + " and " + conferences[1] + ". The number of teams in both conferences must be equal to create home and away teams for the playoffs.")
			return err
		}
		reversedAwayTeams := reverseTeam(awayTeams)
		playoffsQuery :=
			`
			INSERT INTO playoffs 
			(
			playoffs_id, 
			fixture_round, 
			game_count, 
			game_round, 
			home_team_id, 
			home_team_name, 
			home_team_url, 
			players_in_home_id, 
			away_team_id, 
			away_team_name, 
			away_team_url, 
			players_in_away_id, 
			season)
			VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			`
		if limit == 1 {
			playoffsId := uuid.New()
			playersInHomeId := uuid.New()
			playersInAwayId := uuid.New()
			h := homeTeams[0]
			a := awayTeams[0]
			_, err := tx.Exec(
				playoffsQuery,
				playoffsId,
				0000,
				"FINAL",
				"1",
				h.TeamId,
				h.TeamName,
				h.TeamPicUrl,
				playersInHomeId,
				a.TeamId,
				a.TeamName,
				a.TeamPicUrl,
				playersInAwayId,
				season,
			)
			if err != nil {
				log.Println("failed to insert FINAL: CASE 2: ", err.Error())
				return err
			}
		}

		count := len(homeTeams)
		fixtureRound := 1
		for len(homeTeams) > 1 {
			for i := 0; i < len(homeTeams); i++ {
				for inner := 0; inner < 3; inner++ {
					if fixtureRound == 1 {
						_, err := tx.Exec(
							playoffsQuery,
							uuid.New(),
							fixtureRound,
							i+1,
							inner+1,
							homeTeams[i].TeamId,
							homeTeams[i].TeamName,
							homeTeams[i].TeamPicUrl,
							uuid.New(),
							reversedAwayTeams[i].TeamId,
							reversedAwayTeams[i].TeamName,
							reversedAwayTeams[i].TeamPicUrl,
							uuid.New(),
							season,
						)
						if err != nil {
							log.Println("failed to INSERT playoffs records: ERROR CASE = 2, INNER LOOP: ", err.Error())
							return err
						}
					} else {
						playoffsIdNextRound := uuid.New()
						playersInHomeIdNextRound := uuid.New()
						playersInAwayIdNextRound := uuid.New()
						playoffsQueryNextRound :=
							`
									INSERT INTO playoffs 
									(playoffs_id, fixture_round, game_count, game_round,  players_in_home_id,  players_in_away_id, season)
									VALUES($1, $2, $3, $4, $5, $6, $7)
									`
						_, err := tx.Exec(
							playoffsQueryNextRound,
							playoffsIdNextRound,
							fixtureRound,
							i+1+count,
							inner+1,
							playersInHomeIdNextRound,
							playersInAwayIdNextRound,
							season,
						)
						if err != nil {
							log.Println("failed to INSERT playoffs records: ERROR CASE = 2, INNER LOOP fixtureRound > 1: ", err.Error())
							return err
						}
						if inner == 2 && i == len(homeTeams)-1 {
							count = count + len(homeTeams)
						}

					}
				}
			}

			homeTeams = homeTeams[:len(homeTeams)/2]
			reversedAwayTeams = reversedAwayTeams[:len(reversedAwayTeams)/2]
			fixtureRound++

		}
		playoffsQueryFinal :=
			`
						INSERT INTO playoffs 
						(playoffs_id, fixture_round, game_count, game_round,  players_in_home_id,  players_in_away_id, season)
						VALUES($1, $2, $3, $4, $5, $6, $7)
						`
		_, err := tx.Exec(
			playoffsQueryFinal,
			uuid.New(),
			fixtureRound,
			"FINAL",
			"1",
			uuid.New(),
			uuid.New(),
			season,
		)
		if err != nil {
			log.Println("failed to INSERT playoffs records: ERROR CASE =2, FINAL: ", err.Error())
			return err
		}

	case 4:
		var homeTeams1 []models.StandingsModel
		var awayTeams1 []models.StandingsModel
		var homeTeams2 []models.StandingsModel
		var awayTeams2 []models.StandingsModel
		var pairedteams [][]models.StandingsModel
		query :=
			`
				SELECT *, 
				RANK() OVER(PARTITION BY conference ORDER BY pts desc) AS position 
				FROM standings
				WHERE conference = $1 AND season = $2 
				LIMIT $3
			`
		errHt1 := tx.Select(&homeTeams1, query, conferences[0], season, limit)
		if errHt1 != nil {
			log.Println("error SELECTING homeTeams1; CASE = 4 ERROR: ", errHt1)
			return errHt1
		}
		if len(homeTeams1) < limit {
			err := errors.New(conferences[0] + "has less qualified teams of" + fmt.Sprint(len(homeTeams1)) + " teams than the required number of " + fmt.Sprint(limit) + "teams")
			return err
		}
		errAt1 := tx.Select(&awayTeams1, query, conferences[1], season, limit)
		if errAt1 != nil {
			fmt.Println("error SELECTING awayTeams1; CASE = 4 ERROR: ", errAt1)
			return errAt1
		}
		errHt2 := tx.Select(&homeTeams2, query, conferences[2], season, limit)
		if errHt2 != nil {
			log.Println("error SELECTING homeTeams2; CASE = 4 ERROR: ", errHt2)
			return errHt2
		}
		if len(homeTeams2) < limit {
			err := errors.New(conferences[1] + "has less qualified teams of" + fmt.Sprint(len(homeTeams2)) + " teams than the required number of " + fmt.Sprint(limit) + "teams")
			return err
		}
		if len(awayTeams1) < limit {
			err := errors.New(conferences[2] + "has less qualified teams of" + fmt.Sprint(len(awayTeams1)) + " teams than the required number of " + fmt.Sprint(limit) + "teams")
			return err
		}
		errAt2 := tx.Select(&awayTeams2, query, conferences[3], season, limit)
		if errAt2 != nil {
			log.Println("error SELECTING awayTeams1; CASE = 4 ERROR: ", errAt2)
			return errAt2
		}
		if len(awayTeams2) < limit {
			err := errors.New(conferences[3] + "has less qualified teams of" + fmt.Sprint(len(awayTeams2)) + " teams than the required number of " + fmt.Sprint(limit) + "teams")
			return err
		}
		if len(homeTeams1) != len(awayTeams1) {
			err := errors.New("The number of teams in all conferences must be equal. " + conferences[1] + "has " + fmt.Sprint(len(awayTeams1)) + " qualified teams which is not the same as the other conferences.")
			return err
		}
		if len(homeTeams1) != len(homeTeams2) {
			err := errors.New("The number of teams in all conferences must be equal. " + conferences[2] + "has " + fmt.Sprint(len(homeTeams2)) + " qualified teams which is not the same as the other conferences.")
			return err
		}
		if len(homeTeams1) != len(awayTeams2) {
			err := errors.New("The number of teams in all conferences must be equal. " + conferences[3] + "has " + fmt.Sprint(len(homeTeams2)) + " qualified teams which is not the same as the other conferences.")
			return err
		}

		reversedAwayTeam1 := reverseTeam(awayTeams1)
		reversedAwayTeam2 := reverseTeam(awayTeams2)
		for i := 0; i < len(homeTeams1); i++ {
			pairedteams = append(pairedteams, []models.StandingsModel{homeTeams1[i], reversedAwayTeam1[i]}, []models.StandingsModel{homeTeams2[i], reversedAwayTeam2[i]})
		}
		count := len(pairedteams)
		playoffsQuery :=
			`
			INSERT INTO playoffs 
			(
			playoffs_id, 
			fixture_round, 
			game_count, 
			game_round, 
			home_team_id, 
			home_team_name, 
			home_team_url, 
			players_in_home_id, 
			away_team_id, 
			away_team_name, 
			away_team_url, 
			players_in_away_id, 
			season)
			VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			`
		round := 1
		for len(pairedteams) > 1 {
			for i := 0; i < len(pairedteams); i++ {
				for inner := 0; inner < 3; inner++ {
					if round == 1 {
						_, err := tx.Exec(
							playoffsQuery,
							uuid.New(),
							round,
							i+1,
							inner+1,
							pairedteams[i][0].TeamId,
							pairedteams[i][0].TeamName,
							pairedteams[i][0].TeamPicUrl,
							uuid.New(),
							pairedteams[i][1].TeamId,
							pairedteams[i][1].TeamName,
							pairedteams[i][1].TeamPicUrl,
							uuid.New(),
							season,
						)
						if err != nil {
							log.Println("failed to INSERT playoffs records: ERROR CASE = 4, INNER LOOP: ", err.Error())
							return err
						}
					} else {

						playoffsQueryNextRound :=
							`
									INSERT INTO playoffs 
									(playoffs_id, fixture_round, game_count, game_round,  players_in_home_id,  players_in_away_id, season)
									VALUES($1, $2, $3, $4, $5, $6, $7)
									`
						_, err := tx.Exec(
							playoffsQueryNextRound,
							uuid.New(),
							round,
							i+1+count,
							inner+1,
							uuid.New(),
							uuid.New(),
							season,
						)
						if err != nil {
							log.Println("failed to INSERT playoffs records: ERROR CASE = 4, INNER LOOP NEXT ROUND: ", err.Error())
							return err
						}
					}
				}
			}
			pairedteams = pairedteams[:len(pairedteams)/2]
			round++

		}

		playoffsQueryFinal :=
			`
									INSERT INTO playoffs 
									(playoffs_id, fixture_round, game_count, game_round,  players_in_home_id,  players_in_away_id, season)
									VALUES($1, $2, $3, $4, $5, $6, $7)
									`
		_, err := tx.Exec(
			playoffsQueryFinal,
			uuid.New(),
			round,
			"FINAL",
			"1",
			uuid.New(),
			uuid.New(),
			season,
		)
		if err != nil {
			log.Println("failed to INSERT playoffs records: ERROR CASE = 4, FINAL: ", err.Error())
			return err
		}

	case 8:
		var homeTeams1 []models.StandingsModel
		var awayTeams1 []models.StandingsModel
		var homeTeams2 []models.StandingsModel
		var awayTeams2 []models.StandingsModel
		var homeTeams3 []models.StandingsModel
		var awayTeams3 []models.StandingsModel
		var homeTeams4 []models.StandingsModel
		var awayTeams4 []models.StandingsModel
		var pairedteams [][]models.StandingsModel
		query :=
			`
				SELECT *, 
				RANK() OVER(PARTITION BY conference ORDER BY pts desc) AS position 
				FROM standings
				WHERE conference = $1 AND season = $2
				LIMIT $3
			`
		errHt1 := tx.Select(&homeTeams1, query, conferences[0], season, limit)
		if errHt1 != nil {
			log.Println("error SELECTING homeTeams1; CASE = 8 ERROR: ", errHt1)
			return errHt1
		}
		if len(homeTeams1) < limit {
			err := errors.New(conferences[0] + "has less qualified teams of" + fmt.Sprint(len(homeTeams1)) + " teams than the required number of " + fmt.Sprint(limit) + "teams")
			return err
		}
		errAt1 := tx.Select(&awayTeams1, query, conferences[1], season, limit)
		if errAt1 != nil {
			log.Println("error SELECTING awayTeams1; CASE = 8 ERROR: ", errAt1)
			return errAt1
		}
		if len(awayTeams1) < limit {
			err := errors.New(conferences[1] + "has less qualified teams of" + fmt.Sprint(len(awayTeams1)) + " teams than the required number of " + fmt.Sprint(limit) + "teams")
			return err
		}
		errHt2 := tx.Select(&homeTeams2, query, conferences[2], season, limit)
		if errHt2 != nil {
			log.Println("error SELECTING homeTeams2; CASE = 8 ERROR: ", errHt2)
			return errHt2
		}
		if len(homeTeams2) < limit {
			err := errors.New(conferences[2] + "has less qualified teams of" + fmt.Sprint(len(homeTeams2)) + " teams than the required number of " + fmt.Sprint(limit) + "teams")
			return err
		}
		errAt2 := tx.Select(&awayTeams2, query, conferences[3], season, limit)
		if errAt2 != nil {
			log.Println("error SELECTING awayTeams1; CASE = 8 ERROR: ", errAt2)
			return errAt2
		}
		if len(awayTeams2) < limit {
			err := errors.New(conferences[3] + "has less qualified teams of" + fmt.Sprint(len(awayTeams2)) + " teams than the required number of " + fmt.Sprint(limit) + "teams")
			return err
		}
		errHt3 := tx.Select(&homeTeams3, query, conferences[4], season, limit)
		if errHt3 != nil {
			log.Println("error SELECTING homeTeams2; CASE = 8 ERROR: ", errHt3)
			return errHt3
		}
		if len(homeTeams3) < limit {
			err := errors.New(conferences[4] + "has less qualified teams of" + fmt.Sprint(len(homeTeams3)) + " teams than the required number of " + fmt.Sprint(limit) + "teams")
			return err
		}
		errAt3 := tx.Select(&awayTeams3, query, conferences[5], season, limit)
		if errAt3 != nil {
			log.Println("error SELECTING awayTeams1; CASE = 8 ERROR: ", errAt3)
			return errAt3
		}
		if len(awayTeams3) < limit {
			err := errors.New(conferences[5] + "has less qualified teams of" + fmt.Sprint(len(awayTeams3)) + " teams than the required number of " + fmt.Sprint(limit) + "teams")
			return err
		}
		errHt4 := tx.Select(&homeTeams4, query, conferences[6], season, limit)
		if errHt4 != nil {
			log.Println("error SELECTING homeTeams2; CASE = 8 ERROR: ", errHt4)
			return errHt4
		}
		if len(homeTeams4) < limit {
			err := errors.New(conferences[6] + "has less qualified teams of" + fmt.Sprint(len(homeTeams4)) + " teams than the required number of " + fmt.Sprint(limit) + "teams")
			return err
		}
		errAt4 := tx.Select(&awayTeams4, query, conferences[7], season, limit)
		if errAt4 != nil {
			log.Println("error SELECTING awayTeams1; CASE = 8 ERROR: ", errAt4)
			return errAt4
		}
		if len(awayTeams4) < limit {
			err := errors.New(conferences[7] + "has less qualified teams of" + fmt.Sprint(len(awayTeams4)) + " teams than the required number of " + fmt.Sprint(limit) + "teams")
			return err
		}

		if len(homeTeams1) != len(awayTeams1) {
			err := errors.New("The number of teams in all conferences must be equal. " + conferences[1] + "has " + fmt.Sprint(len(awayTeams1)) + " qualified teams which is not the same as the other conferences.")
			return err
		}
		if len(homeTeams1) != len(homeTeams2) {
			err := errors.New("The number of teams in all conferences must be equal. " + conferences[2] + "has " + fmt.Sprint(len(homeTeams2)) + " qualified teams which is not the same as the other conferences.")
			return err
		}
		if len(homeTeams1) != len(awayTeams2) {
			err := errors.New("The number of teams in all conferences must be equal. " + conferences[3] + "has " + fmt.Sprint(len(awayTeams2)) + " qualified teams which is not the same as the other conferences.")
			return err
		}
		if len(homeTeams1) != len(homeTeams3) {
			err := errors.New("The number of teams in all conferences must be equal. " + conferences[4] + "has " + fmt.Sprint(len(homeTeams3)) + " qualified teams which is not the same as the other conferences.")
			return err
		}
		if len(homeTeams1) != len(awayTeams3) {
			err := errors.New("The number of teams in all conferences must be equal. " + conferences[5] + "has " + fmt.Sprint(len(awayTeams3)) + " qualified teams which is not the same as the other conferences.")
			return err
		}
		if len(homeTeams1) != len(homeTeams4) {
			err := errors.New("The number of teams in all conferences must be equal. " + conferences[6] + "has " + fmt.Sprint(len(homeTeams4)) + " qualified teams which is not the same as the other conferences.")
			return err
		}
		if len(homeTeams1) != len(awayTeams4) {
			err := errors.New("The number of teams in all conferences must be equal. " + conferences[7] + "has " + fmt.Sprint(len(awayTeams4)) + " qualified teams which is not the same as the other conferences.")
			return err
		}
		reversedAwayTeam1 := reverseTeam(awayTeams1)
		reversedAwayTeam2 := reverseTeam(awayTeams2)
		reversedAwayTeam3 := reverseTeam(awayTeams3)
		reversedAwayTeam4 := reverseTeam(awayTeams4)
		for i := 0; i < len(homeTeams1); i++ {
			pairedteams = append(pairedteams, []models.StandingsModel{homeTeams1[i], reversedAwayTeam1[i]}, []models.StandingsModel{homeTeams2[i], reversedAwayTeam2[i]}, []models.StandingsModel{homeTeams3[i], reversedAwayTeam3[i]}, []models.StandingsModel{homeTeams4[i], reversedAwayTeam4[i]})
		}
		count := len(pairedteams)
		playoffsQuery :=
			`
			INSERT INTO playoffs 
			(
			playoffs_id, 
			fixture_round, 
			game_count, 
			game_round, 
			home_team_id, 
			home_team_name, 
			home_team_url, 
			players_in_home_id, 
			away_team_id, 
			away_team_name, 
			away_team_url, 
			players_in_away_id, 
			season)
			VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			`
		round := 1
		for len(pairedteams) > 1 {
			for i := 0; i < len(pairedteams); i++ {
				for inner := 0; inner < 3; inner++ {
					if round == 1 {
						_, err := tx.Exec(
							playoffsQuery,
							uuid.New(),
							round,
							i+1,
							inner+1,
							pairedteams[i][0].TeamId,
							pairedteams[i][0].TeamName,
							pairedteams[i][0].TeamPicUrl,
							uuid.New(),
							pairedteams[i][1].TeamId,
							pairedteams[i][1].TeamName,
							pairedteams[i][1].TeamPicUrl,
							uuid.New(),
							season,
						)
						if err != nil {
							log.Println("failed to INSERT playoffs records: ERROR CASE = 8, INNER LOOP: ", err.Error())
							return err
						}
					} else {

						playoffsQueryNextRound :=
							`
									INSERT INTO playoffs 
									(playoffs_id, fixture_round, game_count, game_round,  players_in_home_id,  players_in_away_id, season)
									VALUES($1, $2, $3, $4, $5, $6, $7)
									`
						_, err := tx.Exec(
							playoffsQueryNextRound,
							uuid.New(),
							round,
							i+1+count,
							inner+1,
							uuid.New(),
							uuid.New(),
							season,
						)
						if err != nil {
							log.Println("failed to INSERT playoffs records: ERROR CASE = 8, INNER LOOP NEXT ROUND: ", err.Error())
							return err
						}
					}
				}
			}
			pairedteams = pairedteams[:len(pairedteams)/2]
			round++

		}

		playoffsQueryFinal :=
			`
									INSERT INTO playoffs 
									(playoffs_id, fixture_round, game_count, game_round,  players_in_home_id,  players_in_away_id, season)
									VALUES($1, $2, $3, $4, $5, $6, $7)
									`
		_, err := tx.Exec(
			playoffsQueryFinal,
			uuid.New(),
			round,
			"FINAL",
			"1",
			uuid.New(),
			uuid.New(),
			season,
		)
		if err != nil {
			log.Println("failed to INSERT playoffs records: ERROR CASE = 8, FINAL: ", err.Error())
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// REVERSING THE ORDER OF TEAMS FOR PAIRING
func reverseTeam(in []models.StandingsModel) []models.StandingsModel {
	out := append([]models.StandingsModel(nil), in...)
	slices.Reverse(out)
	return out
}

type rounds struct {
	FixtureRound int `db:"fixture_round"`
}
type playCount struct {
	FixtureRound int    `db:"fixture_round"`
	GameCount    string `db:"game_count"`
}

func (p *PlayoffsDBConnection) ListPlayoffs(season string) ([][][]models.PlayoffsModel, error) {
	var playCount []playCount
	var playoffsInner []models.PlayoffsModel
	var rounds []rounds
	queryCount :=
		`
	SELECT fixture_round FROM playoffs WHERE season = $1 GROUP BY fixture_round ORDER BY fixture_round ASC
	`
	errC := p.DB.Select(&rounds, queryCount, season)
	if errC != nil {
		log.Println("error counting fixture_round in playoffs: ", string(errC.Error()))
		if errC.Error() == "sql: no rows in result set" {
			return [][][]models.PlayoffsModel{}, nil
		}
		return [][][]models.PlayoffsModel{}, errC
	}

	query :=
		`
		SELECT fixture_round, game_count
		FROM playoffs 
		WHERE season = $1
		AND fixture_round = $2
		GROUP BY fixture_round, game_count
		ORDER BY fixture_round, 
 		 CASE
    		WHEN game_count ~ '^\d+$' THEN CAST(game_count AS integer)
    	 ELSE NULL
  		 END ASC,
  		game_count ASC
		`
	queryInner :=
		`
			SELECT * FROM playoffs WHERE season = $1 AND fixture_round = $2 AND game_count = $3
		`
	roundsList := make([][][]models.PlayoffsModel, len(rounds))
	for i := 0; i < len(rounds); i++ {
		err := p.DB.Select(&playCount, query, season, rounds[i].FixtureRound)
		if err != nil {
			if err.Error() == "sql: no rows in result set" {
				return [][][]models.PlayoffsModel{}, nil
			}
			log.Println(err.Error())
			return [][][]models.PlayoffsModel{}, err
		}
		if len(playCount) == 1 {
			roundsList[i] = make([][]models.PlayoffsModel, len(playCount))
		} else {
			roundsList[i] = make([][]models.PlayoffsModel, len(playCount))
		}

		for inner := 0; inner < len(roundsList[i]); inner++ {
			err := p.DB.Select(&playoffsInner, queryInner, season, rounds[i].FixtureRound, playCount[inner].GameCount)
			if err != nil {
				if err.Error() == "sql: no rows in result set" {
					return [][][]models.PlayoffsModel{}, nil
				}
				log.Println(err.Error())
				return [][][]models.PlayoffsModel{}, err
			}
			roundsList[i][inner] = append(roundsList[i][inner], playoffsInner...)

		}

	}
	return roundsList, nil
}

type PlayoffsModelReqQuery struct {
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
type WinnerRes struct {
	Winner uuid.UUID `db:"winner"`
}

func (p *PlayoffsDBConnection) UpdatePlayoffsToNull(playoffsId uuid.UUID, round int, teamId uuid.UUID, season string) error {
	playoffsListWinnerHome := []WinnerRes{}
	playoffsListWinnerAway := []WinnerRes{}
	query :=
		`
	UPDATE playoffs
	SET winner = $1
	WHERE playoffs_id = $2
	`
	querySelectHomeTeam :=
		`
	SELECT winner
	FROM playoffs
	WHERE winner = $1
	AND season = $2
	AND fixture_round = $3
	`
	querySelectAwayTeam :=
		`
	SELECT winner
	FROM playoffs
	WHERE winner = $1
	AND season = $2
	AND fixture_round = $3
	`
	queryUpdateNextRoundHome :=
		`
	UPDATE playoffs
	SET home_team_id = $1, home_team_name = $2, home_team_url = $3, winner = $4
	WHERE home_team_id = $5
	AND fixture_round = $6 
	AND season = $7
	`
	queryUpdateNextRoundAway :=
		`
	UPDATE playoffs
	SET away_team_id = $1, away_team_name = $2, away_team_url = $3, winner = $4
	WHERE away_team_id = $5
	AND fixture_round = $6
	AND season = $7 
	`
	tx, errTx := p.DB.Beginx()
	if errTx != nil {
		return errTx
	}

	defer func() {
		_ = tx.Rollback()
	}()

	sqlRow, err := tx.Exec(query, nil, playoffsId)
	if err != nil {
		return err
	}
	row, errR := sqlRow.RowsAffected()
	if errR != nil {
		return errR
	}
	if row == 0 {
		return errors.New("could not update the requested record")
	}
	errSHome := tx.Select(&playoffsListWinnerHome, querySelectHomeTeam, teamId, season, round)
	if errSHome != nil {
		return errSHome
	}
	errSAway := tx.Select(&playoffsListWinnerAway, querySelectAwayTeam, teamId, season, round)
	if errSAway != nil {
		return errSAway
	}
	if len(playoffsListWinnerHome) < 2 {
		if len(playoffsListWinnerHome) != 0 {
			if playoffsListWinnerHome[0].Winner == teamId {
				_, errUh := tx.Exec(queryUpdateNextRoundHome, nil, nil, nil, nil, teamId, round+1, season)
				if errUh != nil {
					return errUh
				}

			}
		}

	}
	if len(playoffsListWinnerAway) < 2 {
		if len(playoffsListWinnerAway) != 0 {
			if playoffsListWinnerAway[0].Winner == teamId {
				_, errUa := tx.Exec(queryUpdateNextRoundAway, nil, nil, nil, nil, teamId, round+1, season)
				if errUa != nil {
					return errUa
				}

			}
		}
	}
	errC := tx.Commit()
	if errC != nil {
		return errC
	}
	return nil
}

func (p *PlayoffsDBConnection) UpdatePlayoffs(playoffsId uuid.UUID, playoffs PlayoffsModelReqQuery) error {
	var playCountInit []playCount
	var playCountNextRound []playCount
	var playoffsWinnerHome []models.PlayoffsModel
	var playoffsWinnerAway []models.PlayoffsModel
	queryCount :=
		`
		SELECT fixture_round, game_count
		FROM playoffs
		WHERE season = $1
		AND fixture_round = $2
		GROUP BY fixture_round, game_count
		ORDER BY fixture_round,
		 CASE
			WHEN game_count ~ '^\d+$' THEN CAST(game_count AS integer)
		 ELSE NULL
		 END ASC,
		game_count ASC
		`
	query :=
		`
	UPDATE playoffs
	SET winner = $1
	WHERE playoffs_id = $2
	`
	queryWinner :=
		`
	SELECT * 
	FROM playoffs
	WHERE winner = $1
	AND fixture_round = $2
	AND game_count = $3
	`
	queryUpdateNextRoundHome :=
		`
	UPDATE playoffs
	SET
	home_team_id = $1, home_team_name = $2,
	home_team_url = $3
	WHERE season = $4
	AND fixture_round = $5
	AND game_count = $6
	`
	queryUpdateNextRoundAway :=
		`
	UPDATE playoffs
	SET
	away_team_id = $1, away_team_name = $2,
	away_team_url = $3
	WHERE season = $4
	AND fixture_round = $5
	AND game_count = $6
	`
	tx, errTx := p.DB.Beginx()
	if errTx != nil {
		return errTx
	}
	defer func() {
		_ = tx.Rollback()
	}()
	sqlRow, errU := tx.Exec(query, playoffs.Winner, playoffsId)
	if errU != nil {
		return errU
	}
	row, errR := sqlRow.RowsAffected()
	if errR != nil {
		return errR
	}
	if row == 0 {
		return errors.New("failed to update the requested row")
	}
	errSH := tx.Select(&playoffsWinnerHome, queryWinner, playoffs.HomeTeamId, playoffs.FixtureRound, playoffs.GameCount)
	if errSH != nil {
		return errSH
	}

	errSA := tx.Select(&playoffsWinnerAway, queryWinner, playoffs.AwayTeamId, playoffs.FixtureRound, playoffs.GameCount)
	if errSA != nil {
		return errSA
	}

	// CONDITION IF THE LIST OF WINNER HAS TWO IDS OF TEAM IN THE HOME SIDE WHICH IS THE WINNING TEAM
	if len(playoffsWinnerHome) == 2 {
		errCount := tx.Select(&playCountInit, queryCount, playoffs.Season, playoffs.FixtureRound)
		if errCount != nil {
			return errCount
		}
		var rowList []PlayoffsModelReqQuery
		var newList [][]PlayoffsModelReqQuery
		var newListFinal [][][]PlayoffsModelReqQuery

		pla := PlayoffsModelReqQuery{}
		for _, p := range playCountInit {
			// MAKES A NEW LIST CALLED (rowList) WHERE EVERY ITEM WILL BE EMPTY EXCEPT THE ITEM WHERE (GameCount) ARE THE SAME
			if p.GameCount != playoffs.GameCount {
				rowList = append(rowList, pla)
			} else {
				rowList = append(rowList, playoffs)
			}
		}
		for i := 0; i < len(rowList); i += 2 {
			newList = append(newList, rowList[i:i+2])
		}
		if len(newList) >= 2 {
			for i := 0; i < len(newList); i += 2 {
				newListFinal = append(newListFinal, newList[i:i+2])
			}
		}

		// ENSURING THAT THIS IS THE FINAL GAME WHERE THE REMAINING GAMES IS  ONLY 2 TEAMS
		if len(rowList) == 2 {
			if rowList[0].HomeTeamId == playoffs.Winner && rowList[0].HomeTeamId.String() != "00000000-0000-0000-0000-000000000000" {
				sqlRow, errUpdateFinal := tx.Exec(
					queryUpdateNextRoundHome,
					rowList[0].HomeTeamId,
					rowList[0].HomeTeamName,
					rowList[0].HomeTeamURL,
					rowList[0].Season,
					rowList[0].FixtureRound+1,
					"FINAL")
				if errUpdateFinal != nil {
					return errUpdateFinal
				}
				row, errU := sqlRow.RowsAffected()
				if errU != nil {
					return errU
				}
				if row == 0 {
					return errors.New("failed to update the requested record, record does not exists")
				}
			}

			if rowList[1].HomeTeamId == playoffs.Winner && rowList[1].HomeTeamId.String() != "00000000-0000-0000-0000-000000000000" {
				sqlRow, errUpdateFinal := tx.Exec(
					queryUpdateNextRoundAway,
					rowList[1].HomeTeamId,
					rowList[1].HomeTeamName,
					rowList[1].HomeTeamURL,
					rowList[1].Season,
					rowList[1].FixtureRound+1,
					"FINAL")
				if errUpdateFinal != nil {
					return errUpdateFinal
				}
				row, errU := sqlRow.RowsAffected()
				if errU != nil {
					return errU
				}
				if row == 0 {
					return errors.New("failed to update the requested record, record does not exists")
				}
			}

			// NOW EXECUTING THE NEXT ROUND SINCE IT IS NOT THE FINALS
		} else {
			errCountNext := tx.Select(&playCountNextRound, queryCount, playoffs.Season, playoffs.FixtureRound+1)
			if errCountNext != nil {
				return errCountNext
			}

			var playCountNextRoundFinal [][]playCount
			for i := 0; i < len(playCountNextRound); i += 2 {
				playCountNextRoundFinal = append(playCountNextRoundFinal, playCountNextRound[i:i+2])
			}

			// UPDATE IF THE LAST ROUND IS NOT 0 MEANING IT IS NOT THE FINAL ROUND
			if len(playCountNextRound) != 0 {
				for index := range newListFinal {
					if newListFinal[index][0][0].HomeTeamId == playoffs.Winner && newListFinal[index][0][0].HomeTeamId.String() != "00000000-0000-0000-0000-000000000000" {
						sqlRow, errUpdateNextRound := tx.Exec(
							queryUpdateNextRoundHome,
							newListFinal[index][0][0].HomeTeamId,
							newListFinal[index][0][0].HomeTeamName,
							newListFinal[index][0][0].HomeTeamURL,
							newListFinal[index][0][0].Season,
							playCountNextRound[index].FixtureRound,
							playCountNextRoundFinal[index][0].GameCount,
						)
						if errUpdateNextRound != nil {
							return errUpdateNextRound
						}
						row, errU := sqlRow.RowsAffected()
						// fmt.Printf("rows affected %d", row)
						if errU != nil {
							return errU
						}
						if row == 0 {
							return errors.New("failed to update the requested record, record does not exists")
						}
					} else if newListFinal[index][0][1].HomeTeamId == playoffs.Winner && newListFinal[index][0][1].HomeTeamId.String() != "00000000-0000-0000-0000-000000000000" {
						sqlRow, errUpdateNextRound := tx.Exec(
							queryUpdateNextRoundAway,
							newListFinal[index][0][1].HomeTeamId,
							newListFinal[index][0][1].HomeTeamName,
							newListFinal[index][0][1].HomeTeamURL,
							newListFinal[index][0][1].Season,
							playCountNextRound[index].FixtureRound,
							playCountNextRoundFinal[index][0].GameCount)
						if errUpdateNextRound != nil {
							return errUpdateNextRound
						}
						row, errU := sqlRow.RowsAffected()
						// fmt.Printf("rows affected %d", row)
						if errU != nil {
							return errU
						}
						if row == 0 {
							return errors.New("failed to update the requested record, record does not exists")
						}
					} else if newListFinal[index][1][0].HomeTeamId == playoffs.Winner && newListFinal[index][1][0].HomeTeamId.String() != "00000000-0000-0000-0000-000000000000" {
						sqlRow, errUpdateNextRound := tx.Exec(
							queryUpdateNextRoundHome,
							newListFinal[index][1][0].HomeTeamId,
							newListFinal[index][1][0].HomeTeamName,
							newListFinal[index][1][0].HomeTeamURL,
							newListFinal[index][1][0].Season,
							playCountNextRound[index].FixtureRound,
							playCountNextRoundFinal[index][1].GameCount)
						if errUpdateNextRound != nil {
							return errUpdateNextRound
						}
						row, errU := sqlRow.RowsAffected()
						if errU != nil {
							return errU
						}
						if row == 0 {
							return errors.New("failed to update the requested record, record does not exists")
						}
					} else if newListFinal[index][1][1].HomeTeamId == playoffs.Winner && newListFinal[index][1][1].HomeTeamId.String() != "00000000-0000-0000-0000-000000000000" {
						sqlRow, errUpdateNextRound := tx.Exec(
							queryUpdateNextRoundAway,
							newListFinal[index][1][1].HomeTeamId,
							newListFinal[index][1][1].HomeTeamName,
							newListFinal[index][1][1].HomeTeamURL,
							newListFinal[index][1][1].Season,
							playCountNextRound[index].FixtureRound,
							playCountNextRoundFinal[index][1].GameCount,
						)
						if errUpdateNextRound != nil {
							return errUpdateNextRound
						}
						row, errU := sqlRow.RowsAffected()
						if errU != nil {
							return errU
						}
						if row == 0 {
							return errors.New("failed to update the requested record, record does not exists")
						}
					}

				}

			}
		}

		// CONDITION IF THE LIST OF WINNER HAS TWO OR MORE IDS OF TEAM IN THE AWAY SIDE
	} else if len(playoffsWinnerAway) == 2 {
		errCount := tx.Select(&playCountInit, queryCount, playoffs.Season, playoffs.FixtureRound)
		if errCount != nil {
			return errCount
		}

		var rowList []PlayoffsModelReqQuery
		var newList [][]PlayoffsModelReqQuery
		var newListFinal [][][]PlayoffsModelReqQuery

		pla := PlayoffsModelReqQuery{}
		for _, p := range playCountInit {
			if p.GameCount != playoffs.GameCount {
				rowList = append(rowList, pla)
			} else {
				rowList = append(rowList, playoffs)
			}
		}
		for i := 0; i < len(rowList); i += 2 {
			newList = append(newList, rowList[i:i+2])
		}
		if len(newList) >= 2 {
			for i := 0; i < len(newList); i += 2 {
				newListFinal = append(newListFinal, newList[i:i+2])
			}
		}
		if len(rowList) == 2 {
			if rowList[0].AwayTeamId == playoffs.Winner && rowList[0].AwayTeamId.String() != "00000000-0000-0000-0000-000000000000" {
				sqlRow, errUpdateFinal := tx.Exec(
					queryUpdateNextRoundHome,
					rowList[0].AwayTeamId,
					rowList[0].AwayTeamName,
					rowList[0].AwayTeamURL,
					rowList[0].Season,
					rowList[0].FixtureRound+1, "FINAL")
				if errUpdateFinal != nil {
					return errUpdateFinal
				}
				row, errU := sqlRow.RowsAffected()
				if errU != nil {
					return errU
				}
				if row == 0 {
					return errors.New("failed to update the requested record, record does not exists")
				}
			}

			if rowList[1].AwayTeamId == playoffs.Winner && rowList[1].AwayTeamId.String() != "00000000-0000-0000-0000-000000000000" {
				sqlRow, errUpdateFinal := tx.Exec(
					queryUpdateNextRoundAway,
					rowList[1].AwayTeamId,
					rowList[1].AwayTeamName,
					rowList[1].AwayTeamURL,
					rowList[1].Season,
					rowList[1].FixtureRound+1, "FINAL")
				if errUpdateFinal != nil {
					return errUpdateFinal
				}
				row, errU := sqlRow.RowsAffected()
				if errU != nil {
					return errU
				}
				if row == 0 {
					return errors.New("failed to update the requested record, record does not exists")
				}
			}

		} else {

			errCountNext := tx.Select(&playCountNextRound, queryCount, playoffs.Season, playoffs.FixtureRound+1)
			if errCountNext != nil {
				return errCountNext
			}

			var playCountNextRoundFinal [][]playCount
			for i := 0; i < len(playCountNextRound); i += 2 {
				playCountNextRoundFinal = append(playCountNextRoundFinal, playCountNextRound[i:i+2])
			}

			for index := range newListFinal {
				if newListFinal[index][0][0].AwayTeamId == playoffs.Winner && newListFinal[index][0][0].AwayTeamId.String() != "00000000-0000-0000-0000-000000000000" {
					sqlRow, errUpdateNextRound := tx.Exec(
						queryUpdateNextRoundHome,
						newListFinal[index][0][0].AwayTeamId,
						newListFinal[index][0][0].AwayTeamName,
						newListFinal[index][0][0].AwayTeamURL,
						newListFinal[index][0][0].Season,
						playCountNextRound[index].FixtureRound,
						playCountNextRoundFinal[index][0].GameCount)

					if errUpdateNextRound != nil {
						return errUpdateNextRound
					}
					row, errU := sqlRow.RowsAffected()
					if errU != nil {
						return errU
					}
					if row == 0 {
						return errors.New("failed to update the requested record, record does not exists")
					}
				} else if newListFinal[index][0][1].AwayTeamId == playoffs.Winner && newListFinal[index][0][1].AwayTeamId.String() != "00000000-0000-0000-0000-000000000000" {
					sqlRow, errUpdateNextRound := tx.Exec(
						queryUpdateNextRoundAway,
						newListFinal[index][0][1].AwayTeamId,
						newListFinal[index][0][1].AwayTeamName,
						newListFinal[index][0][1].AwayTeamURL,
						newListFinal[index][0][1].Season,
						playCountNextRound[index].FixtureRound,
						playCountNextRoundFinal[index][0].GameCount)

					if errUpdateNextRound != nil {
						return errUpdateNextRound
					}
					row, errU := sqlRow.RowsAffected()
					if errU != nil {
						return errU
					}
					if row == 0 {
						return errors.New("failed to update the requested record, record does not exists")
					}
				} else if newListFinal[index][1][0].AwayTeamId == playoffs.Winner && newListFinal[index][1][0].AwayTeamId.String() != "00000000-0000-0000-0000-000000000000" {
					sqlRow, errUpdateNextRound := tx.Exec(
						queryUpdateNextRoundHome,
						newListFinal[index][1][0].AwayTeamId,
						newListFinal[index][1][0].AwayTeamName,
						newListFinal[index][1][0].AwayTeamURL,
						newListFinal[index][1][0].Season,
						playCountNextRound[index].FixtureRound,
						playCountNextRoundFinal[index][1].GameCount)

					if errUpdateNextRound != nil {
						return errUpdateNextRound
					}

					row, errU := sqlRow.RowsAffected()
					if errU != nil {
						return errU
					}
					if row == 0 {
						return errors.New("failed to update the requested record, record does not exists")
					}
				} else if newListFinal[index][1][1].AwayTeamId == playoffs.Winner && newListFinal[index][1][1].AwayTeamId.String() != "00000000-0000-0000-0000-000000000000" {
					sqlRow, errUpdateNextRound := tx.Exec(
						queryUpdateNextRoundAway,
						newListFinal[index][1][1].AwayTeamId,
						newListFinal[index][1][1].AwayTeamName,
						newListFinal[index][1][1].AwayTeamURL,
						newListFinal[index][1][1].Season,
						playCountNextRound[index].FixtureRound,
						playCountNextRoundFinal[index][1].GameCount)
					if errUpdateNextRound != nil {
						return errUpdateNextRound
					}
					row, errU := sqlRow.RowsAffected()
					if errU != nil {
						return errU
					}
					if row == 0 {
						return errors.New("failed to update the requested record, record does not exists")
					}
				}

			}
		}

	}
	errC := tx.Commit()
	if errC != nil {
		log.Println("failed to commit playoffs tx: ", errC.Error())
		return errC
	}

	return nil
}

func (p *PlayoffsDBConnection) DeletePlayoffs(season string) error {
	query :=
		`
	DELETE FROM playoffs WHERE season = $1
	`
	sqlRow, err := p.Exec(query, season)
	if err != nil {
		return err
	}
	row, _ := sqlRow.RowsAffected()
	if row == 0 {
		return errors.New("could not delete the requested records. Records of season" + season + " do not exists")
	}
	return nil
}
