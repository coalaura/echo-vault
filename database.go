package main

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func connectToDatabase() error {
	db, err := sql.Open("sqlite3", "./echo.db")
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS echos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		hash TEXT NOT NULL,
		name TEXT NOT NULL,
		extension TEXT NOT NULL,
		upload_size INTEGER NOT NULL,
		timestamp INTEGER NOT NULL
	)`)
	if err != nil {
		return err
	}

	database = db

	return nil
}

func (e *Echo) Create() error {
	if e.Hash == "" {
		hash, err := findFreeHash()
		if err != nil {
			return nil
		}

		e.Hash = hash
	}

	if e.Timestamp == 0 {
		e.Timestamp = time.Now().Unix()
	}

	_, err := database.Exec("INSERT INTO echos (hash, name, extension, upload_size, timestamp) VALUES (?, ?, ?, ?, ?)", e.Hash, e.Name, e.Extension, e.UploadSize, e.Timestamp)
	if err != nil {
		return err
	}

	return nil
}

func (e *Echo) Exists() bool {
	var count int

	err := database.QueryRow("SELECT COUNT(id) FROM echos WHERE hash = ?", e.Hash).Scan(&count)
	if err != nil {
		return false
	}

	return count > 0
}

func (e *Echo) Delete() error {
	_, err := database.Exec("DELETE FROM echos WHERE hash = ?", e.Hash)
	if err != nil {
		return err
	}

	return nil
}
