package main

import "database/sql"

type SQLiteColumn struct {
	Name       string
	Definition string
}

type SQLiteTableBuilder struct {
	db    *sql.DB
	table string

	columns map[string]bool
}

func NewTableBuilder(db *sql.DB, table string) *SQLiteTableBuilder {
	return &SQLiteTableBuilder{
		db:    db,
		table: table,
	}
}

func (b *SQLiteTableBuilder) Create() error {
	_, err := b.db.Exec(`CREATE TABLE IF NOT EXISTS echos (
		id INTEGER PRIMARY KEY AUTOINCREMENT
	)`)
	if err != nil {
		return err
	}

	rows, err := b.db.Query("PRAGMA table_info(echos)")
	if err != nil {
		return err
	}

	b.columns = make(map[string]bool)

	for rows.Next() {
		var _ignore interface{}
		var name string

		err = rows.Scan(&_ignore, &name, &_ignore, &_ignore, &_ignore, &_ignore)
		if err != nil {
			return err
		}

		b.columns[name] = true
	}

	return nil
}

func (b *SQLiteTableBuilder) AddColumn(col SQLiteColumn) error {
	if b.columns[col.Name] {
		return nil
	}

	_, err := b.db.Exec("ALTER TABLE echos ADD COLUMN " + col.Name + " " + col.Definition)
	if err != nil {
		return err
	}

	b.columns[col.Name] = true

	return nil
}

func (b *SQLiteTableBuilder) AddColumns(cols []SQLiteColumn) error {
	for _, col := range cols {
		err := b.AddColumn(col)
		if err != nil {
			return err
		}
	}

	return nil
}
