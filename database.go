package main

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

type EchoDatabase struct {
	*sql.DB
}

func connectToDatabase() error {
	db, err := sql.Open("sqlite", "./echo.db")
	if err != nil {
		return err
	}

	b := NewTableBuilder(db, "echos")

	err = b.Create()
	if err != nil {
		return err
	}

	err = b.AddColumns([]SQLiteColumn{
		{"hash", "TEXT NOT NULL"},
		{"name", "TEXT NOT NULL"},
		{"extension", "TEXT NOT NULL"},
		{"upload_size", "INTEGER NOT NULL"},
		{"timestamp", "INTEGER NOT NULL"},
	})
	if err != nil {
		return err
	}

	database = &EchoDatabase{db}

	return nil
}

func (d *EchoDatabase) Exists(hash string) (bool, error) {
	var count int

	err := d.QueryRow("SELECT COUNT(id) FROM echos WHERE hash = ?", hash).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (d *EchoDatabase) Find(hash string) (*Echo, error) {
	var e Echo

	err := d.QueryRow("SELECT id, hash, name, extension, upload_size, timestamp FROM echos WHERE hash = ? LIMIT 1", hash).Scan(&e.ID, &e.Hash, &e.Name, &e.Extension, &e.UploadSize, &e.Timestamp)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	return &e, nil
}

func (d *EchoDatabase) FindAll(offset, limit int) ([]Echo, error) {
	rows, err := database.Query("SELECT id, hash, name, extension, upload_size, timestamp FROM echos ORDER BY timestamp DESC LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, err
	}

	var echos []Echo

	for rows.Next() {
		var e Echo

		err := rows.Scan(&e.ID, &e.Hash, &e.Name, &e.Extension, &e.UploadSize, &e.Timestamp)
		if err != nil {
			return nil, err
		}

		echos = append(echos, e)
	}

	return echos, nil
}

func (d *EchoDatabase) Create(echo *Echo) error {
	err := echo.Fill()
	if err != nil {
		return err
	}

	_, err = database.Exec("INSERT INTO echos (hash, name, extension, upload_size, timestamp) VALUES (?, ?, ?, ?, ?)", echo.Hash, echo.Name, echo.Extension, echo.UploadSize, echo.Timestamp)
	if err != nil {
		return err
	}

	return nil
}

func (d *EchoDatabase) Delete(hash string) error {
	_, err := database.Exec("DELETE FROM echos WHERE hash = ?", hash)
	if err != nil {
		return err
	}

	return nil
}

func (d *EchoDatabase) SetExtension(hash, extension string) error {
	_, err := database.Exec("UPDATE echos SET extension = ? WHERE hash = ?", extension, hash)
	if err != nil {
		return err
	}

	return nil
}
