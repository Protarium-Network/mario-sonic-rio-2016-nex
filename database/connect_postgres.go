package database

import (
	"database/sql"
	"os"

	_ "github.com/lib/pq"

	"github.com/Protarium-Network/mario-sonic-rio-2016-nex/globals"
)

var Postgres *sql.DB

func ConnectPostgres() {
	var err error

	Postgres, err = sql.Open("postgres", os.Getenv("PN_RIO2016_POSTGRES_URI"))
	if err != nil {
		globals.Logger.Critical(err.Error())
		os.Exit(1)
	}

	if err = Postgres.Ping(); err != nil {
		globals.Logger.Critical(err.Error())
		os.Exit(1)
	}

	globals.Logger.Success("Connected to Postgres!")

	initPostgres()
}
