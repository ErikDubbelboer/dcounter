package main

import (
	_ "github.com/mattn/go-sqlite3"
)

/*func create() {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS counters (
		  id    TEXT PRIMARY KEY,
			amount FLOAT
		)
	`); err != nil {
		panic(err)
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS changes (
		  id         TEXT,
			replica_id INTEGER,
			diff       FLOAT,
			PRIMARY KEY (id, replica_id)
		)
	`); err != nil {
		panic(err)
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS replicas (
		  id   INTEGER PRIMARY KEY,
			host TEXT
		)
	`); err != nil {
		panic(err)
	}
}

func load() {
	if rows, err := db.Query(`
		SELECT id, amount
		FROM counters
	`); err != nil {
		panic(err)
	} else {
		for rows.Next() {
			var id string
			var amount float64

			if err := rows.Scan(&id, &amount); err != nil {
				panic(err)
			}

			counters[id] = amount
		}
	}

	if rows, err := db.Query(`
		SELECT id, replica_id, diff
		FROM changes
	`); err != nil {
		panic(err)
	} else {
		for rows.Next() {
			var id string
			var replica uint32
			var diff float64

			if err := rows.Scan(&id, &replica, &diff); err != nil {
				panic(err)
			}

			if _, ok := changes[replica]; !ok {
				changes[replica] = make(map[string]float64, 0)
			}

			changes[replica][id] = diff
		}
	}

	if rows, err := db.Query(`
		SELECT id, host
		FROM replicas
	`); err != nil {
		panic(err)
	} else {
		for rows.Next() {
			var id uint32
			var host string

			if err := rows.Scan(&id, &host); err != nil {
				panic(err)
			}

			replicas[host] = id
		}
	}
}*/
