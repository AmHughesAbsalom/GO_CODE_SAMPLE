package dbconnection

import (
	"fmt"

	"os"

	"AmHughesAbsalom/GO_CODE_SAMPLE.git/queries"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type DBConnection struct {
	*queries.PlayoffsDBConnection
}

func NewDBConnection() (*DBConnection, *sqlx.DB, error) {

	godotenv.Load()

	connectionString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		os.Getenv("USER_NAME"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
		os.Getenv("SSL_MODE"),
	)
	db, connErr := sqlx.Open("postgres", connectionString)
	if connErr != nil {
		return nil, &sqlx.DB{}, fmt.Errorf("failed to connect the database!...: %w", connErr)
	}

	if err := db.Ping(); err != nil {
		return nil, &sqlx.DB{}, fmt.Errorf("database connection failed!: %w", err)
	}

	return &DBConnection{
		PlayoffsDBConnection: &queries.PlayoffsDBConnection{DB: db},
	}, db, nil
}
