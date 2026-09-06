package database

import (
	"os"

	"github.com/Protarium-Network/mario-sonic-rio-2016-nex/globals"
)

// initPostgres creates the schema this server needs on first run. Every
// statement is idempotent (CREATE ... IF NOT EXISTS), so it is safe to run
// on every start.
func initPostgres() {
	mustExec := func(label, query string) {
		if _, err := Postgres.Exec(query); err != nil {
			globals.Logger.Criticalf("%s: %s", label, err.Error())
			os.Exit(1)
		}
	}

	mustExec("rio2016_rankings", `CREATE TABLE IF NOT EXISTS rio2016_rankings (
		owner_pid   bigint,
		unique_id   bigint,
		category    bigint,
		score       bigint,
		order_by    smallint,
		update_mode smallint,
		groups      bytea,
		param       bigint,
		updated_at  bigint,
		PRIMARY KEY (unique_id, owner_pid, category)
	)`)

	mustExec("rio2016_ranking_categories", `CREATE TABLE IF NOT EXISTS rio2016_ranking_categories (
		category   bigint PRIMARY KEY,
		order_by   smallint NOT NULL CHECK (order_by IN (0, 1)),
		created_at bigint NOT NULL
	)`)

	mustExec("rio2016_common_datas", `CREATE TABLE IF NOT EXISTS rio2016_common_datas (
		unique_id   bigint,
		owner_pid   bigint,
		common_data bytea,
		updated_at  bigint,
		PRIMARY KEY (unique_id, owner_pid)
	)`)

	mustExec("rio2016 indexes", `
		CREATE INDEX IF NOT EXISTS rio2016_rankings_category_score_idx
			ON rio2016_rankings (category, score, updated_at);
		CREATE INDEX IF NOT EXISTS rio2016_rankings_owner_category_idx
			ON rio2016_rankings (owner_pid, category);
		CREATE INDEX IF NOT EXISTS rio2016_common_datas_owner_updated_idx
			ON rio2016_common_datas (owner_pid, updated_at DESC)
	`)

	// DataStore backing store for the score-attachment (ghost/photo) flow.
	mustExec("datastore schema", `CREATE SCHEMA IF NOT EXISTS datastore`)

	mustExec("datastore sequence", `CREATE SEQUENCE IF NOT EXISTS datastore.object_data_id_seq
		INCREMENT 1 MINVALUE 1 MAXVALUE 281474976710656 START 1 CACHE 1`)

	mustExec("datastore.objects", `CREATE TABLE IF NOT EXISTS datastore.objects (
		data_id                      bigint NOT NULL DEFAULT nextval('datastore.object_data_id_seq') PRIMARY KEY,
		upload_completed             boolean NOT NULL DEFAULT FALSE,
		deleted                      boolean NOT NULL DEFAULT FALSE,
		owner                        bigint,
		size                         int,
		name                         text,
		data_type                    int,
		meta_binary                  bytea,
		permission                   int,
		permission_recipients        int[],
		delete_permission            int,
		delete_permission_recipients int[],
		flag                         int,
		period                       int,
		refer_data_id                bigint,
		tags                         text[],
		persistence_slot_id          int,
		extra_data                   text[],
		access_password              bigint NOT NULL DEFAULT 0,
		update_password              bigint NOT NULL DEFAULT 0,
		creation_date                timestamp,
		update_date                  timestamp
	)`)

	globals.Logger.Success("Postgres schema ready")
}
