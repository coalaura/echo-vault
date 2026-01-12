package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "modernc.org/sqlite"
)

const (
	DatabasePath    = "echo.db"
	VerifyChunkSize = 1024
)

type EchoDatabase struct {
	*sql.DB
}

func ConnectToDatabase() (*EchoDatabase, error) {
	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", DatabasePath)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(16)
	db.SetMaxIdleConns(16)
	db.SetConnMaxLifetime(time.Hour)

	b := NewTableBuilder(db, "echos")

	err = b.Create()
	if err != nil {
		return nil, err
	}

	err = b.AddColumns([]SQLiteColumn{
		{"hash", "TEXT NOT NULL"},
		{"name", "TEXT NOT NULL"},
		{"extension", "TEXT NOT NULL"},
		{"size", "INTEGER NOT NULL DEFAULT 0"},
		{"upload_size", "INTEGER NOT NULL DEFAULT 0"},
		{"timestamp", "INTEGER NOT NULL DEFAULT 0"},

		{"categories", "TEXT NULL"},
		{"tags", "TEXT NULL"},
		{"caption", "TEXT NULL"},
		{"text", "TEXT NULL"},
		{"safety", "TEXT NULL"},
	})
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_echos_hash ON echos(hash)")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_echos_timestamp ON echos(timestamp)")
	if err != nil {
		return nil, err
	}

	return &EchoDatabase{db}, nil
}

func (d *EchoDatabase) Exists(hash string) (bool, error) {
	var exists bool

	err := d.QueryRow("SELECT EXISTS(SELECT 1 FROM echos WHERE hash = ?)", hash).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (d *EchoDatabase) Find(ctx context.Context, hash string) (*Echo, error) {
	var (
		e       Echo
		caption sql.NullString
		safety  sql.NullString
	)

	err := d.QueryRowContext(ctx, "SELECT id, hash, name, extension, size, upload_size, timestamp, caption, safety FROM echos WHERE hash = ? LIMIT 1", hash).Scan(&e.ID, &e.Hash, &e.Name, &e.Extension, &e.Size, &e.UploadSize, &e.Timestamp, &caption, &safety)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	e.Caption = caption.String
	e.Safety = safety.String

	return &e, nil
}

func (d *EchoDatabase) FindAll(ctx context.Context, offset, limit int) ([]Echo, error) {
	rows, err := d.QueryContext(ctx, "SELECT id, hash, name, extension, size, upload_size, timestamp, caption, safety FROM echos ORDER BY timestamp DESC LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var echos []Echo

	for rows.Next() {
		var (
			e       Echo
			caption sql.NullString
			safety  sql.NullString
		)

		err := rows.Scan(&e.ID, &e.Hash, &e.Name, &e.Extension, &e.Size, &e.UploadSize, &e.Timestamp, &caption, &safety)
		if err != nil {
			return nil, err
		}

		e.Caption = caption.String
		e.Safety = safety.String

		echos = append(echos, e)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return echos, nil
}

func (d *EchoDatabase) FindByHashes(ctx context.Context, hashes []string) ([]Echo, error) {
	if len(hashes) == 0 {
		return nil, nil
	}

	placeholders := strings.Repeat("?,", len(hashes))
	placeholders = placeholders[:len(placeholders)-1]

	var b strings.Builder

	b.WriteString("SELECT id, hash, name, extension, size, upload_size, timestamp, caption, safety FROM echos WHERE hash IN (")
	b.WriteString(placeholders)
	b.WriteString(") ORDER BY CASE hash ")

	for i := range hashes {
		b.WriteString("WHEN ? THEN ")
		b.WriteString(strconv.Itoa(i))
		b.WriteString(" ")
	}

	b.WriteString("END")

	args := make([]any, 0, len(hashes)*2)

	for _, hash := range hashes {
		args = append(args, hash)
	}

	for _, hash := range hashes {
		args = append(args, hash)
	}

	rows, err := d.QueryContext(ctx, b.String(), args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var echos []Echo

	for rows.Next() {
		var (
			e       Echo
			caption sql.NullString
			safety  sql.NullString
		)

		err := rows.Scan(&e.ID, &e.Hash, &e.Name, &e.Extension, &e.Size, &e.UploadSize, &e.Timestamp, &caption, &safety)
		if err != nil {
			return nil, err
		}

		e.Caption = caption.String
		e.Safety = safety.String

		echos = append(echos, e)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return echos, nil
}

func (d *EchoDatabase) Create(ctx context.Context, echo *Echo) error {
	err := echo.Fill(ctx)
	if err != nil {
		return err
	}

	_, err = d.Exec("INSERT INTO echos (hash, name, extension, size, upload_size, timestamp) VALUES (?, ?, ?, ?, ?, ?)", echo.Hash, echo.Name, echo.Extension, echo.Size, echo.UploadSize, echo.Timestamp)
	if err != nil {
		return err
	}

	return nil
}

func (d *EchoDatabase) Delete(hash string) error {
	_, err := d.Exec("DELETE FROM echos WHERE hash = ?", hash)
	if err != nil {
		return err
	}

	return nil
}

func (d *EchoDatabase) SetExtension(hash, extension string) error {
	_, err := d.Exec("UPDATE echos SET extension = ? WHERE hash = ?", extension, hash)
	if err != nil {
		return err
	}

	return nil
}

func (d *EchoDatabase) SetSize(hash string, size int64) error {
	_, err := d.Exec("UPDATE echos SET size = ? WHERE hash = ?", size, hash)
	if err != nil {
		return err
	}

	return nil
}

func (d *EchoDatabase) SetSafety(hash string, safety string) error {
	_, err := d.Exec("UPDATE echos SET safety = ? WHERE hash = ?", safety, hash)
	if err != nil {
		return err
	}

	return nil
}

func (d *EchoDatabase) SetTags(hash string, entry EchoTag) error {
	categories, tags, caption, text, safety := entry.Serialize()

	_, err := d.Exec("UPDATE echos SET categories = ?, tags = ?, caption = ?, text = ?, safety = ? WHERE hash = ?", categories, tags, caption, text, safety, hash)
	if err != nil {
		return err
	}

	return nil
}

func (d *EchoDatabase) Verify() (uint64, uint64, error) {
	var total int64

	err := d.QueryRow("SELECT COUNT(id) FROM echos").Scan(&total)
	if err != nil {
		return 0, 0, err
	}

	if total == 0 {
		return 0, 0, nil
	}

	var (
		wg        sync.WaitGroup
		echos     []Echo
		completed atomic.Uint64
		offset    int
		totalSize uint64

		done = make(chan bool)
	)

	log.Printf("Verifying echos 0%% (0 of %d)\n", total)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	wg.Go(func() {
		var (
			last   uint64
			totalF = float64(total)
		)

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				current := completed.Load()

				if current == last {
					continue
				}

				percentage := float64(current) / totalF * 100

				log.Printf("Verifying echos %.1f%% (%d of %d)\n", percentage, current, total)

				last = current
			}
		}
	})

	invalid := make([]any, 0)

	for {
		echos, err = d.FindAll(context.Background(), offset, VerifyChunkSize)
		if err != nil {
			break
		}

		for _, echo := range echos {
			path := echo.Storage()

			stat, err := os.Stat(path)
			if err != nil {
				if os.IsNotExist(err) {
					invalid = append(invalid, echo.Hash)
				} else {
					log.Warnf("%s: %v\n", path, err)
				}
			} else {
				size := stat.Size()

				if echo.Size != size {
					echo.Size = size

					szErr := d.SetSize(echo.Hash, size)
					if szErr != nil {
						log.Warnf("Failed to update size (%s): %v\n", echo.Hash, szErr)
					}
				}

				totalSize += uint64(size)
			}

			completed.Add(1)
		}

		if len(echos) < VerifyChunkSize {
			break
		}

		offset += VerifyChunkSize
	}

	close(done)

	wg.Wait()

	if err != nil {
		return 0, 0, err
	}

	log.Printf("Verifying echos 100%% (%d of %d)\n", total, total)

	if len(invalid) > 0 {
		if config.Server.DeleteOrphans {
			log.Printf("Deleting %d orphan echos...\n", len(invalid))

			placeholders := strings.Repeat("?,", len(invalid))
			placeholders = placeholders[:len(placeholders)-1]

			_, err := d.Exec(fmt.Sprintf("DELETE FROM echos WHERE hash IN (%s)", placeholders), invalid...)
			if err != nil {
				return 0, 0, err
			}

			total -= int64(len(invalid))

			log.Println("Completed")
		} else {
			log.Warnf("%d echos are orphaned\n", len(invalid))
		}
	}

	return totalSize, uint64(total), nil
}
