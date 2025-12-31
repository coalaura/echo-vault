package main

import (
	"database/sql"
	"fmt"
	"os"
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
	db, err := sql.Open("sqlite", DatabasePath)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)

	b := NewTableBuilder(db, "echos")

	err = b.Create()
	if err != nil {
		return nil, err
	}

	err = b.AddColumns([]SQLiteColumn{
		{"hash", "TEXT NOT NULL UNIQUE"},
		{"name", "TEXT NOT NULL"},
		{"extension", "TEXT NOT NULL"},
		{"size", "INTEGER NOT NULL DEFAULT 0"},
		{"upload_size", "INTEGER NOT NULL DEFAULT 0"},
		{"timestamp", "INTEGER NOT NULL DEFAULT 0"},
	})
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

func (d *EchoDatabase) Find(hash string) (*Echo, error) {
	var e Echo

	err := d.QueryRow("SELECT id, hash, name, extension, size, upload_size, timestamp FROM echos WHERE hash = ? LIMIT 1", hash).Scan(&e.ID, &e.Hash, &e.Name, &e.Extension, &e.Size, &e.UploadSize, &e.Timestamp)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	return &e, nil
}

func (d *EchoDatabase) FindAll(offset, limit int) ([]Echo, error) {
	rows, err := d.Query("SELECT id, hash, name, extension, size, upload_size, timestamp FROM echos ORDER BY timestamp DESC LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var echos []Echo

	for rows.Next() {
		var e Echo

		err := rows.Scan(&e.ID, &e.Hash, &e.Name, &e.Extension, &e.Size, &e.UploadSize, &e.Timestamp)
		if err != nil {
			return nil, err
		}

		echos = append(echos, e)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return echos, nil
}

func (d *EchoDatabase) Create(echo *Echo) error {
	err := echo.Fill()
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

	log.Printf("Verifying 0%% (0 of %d)\n", total)

	ticker := time.NewTicker(time.Second)

	defer ticker.Stop()

	wg.Go(func() {
		totalF := float64(total)

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				current := completed.Load()
				percentage := float64(current) / totalF * 100

				log.Printf("Verifying %.1f%% (%d of %d)\n", percentage, current, total)
			}
		}
	})

	invalid := make([]any, 0)

	for {
		echos, err = d.FindAll(offset, VerifyChunkSize)
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

	log.Printf("Verifying 100%% (%d of %d)\n", total, total)

	if len(invalid) > 0 {
		if config.Server.DeleteOrphans {
			log.Printf("Deleting %d orphan echos...\n", len(invalid))

			placeholders := strings.Repeat("?,", len(invalid))
			placeholders = placeholders[:len(placeholders)-1]

			_, err := d.Exec(fmt.Sprintf("DELETE FROM echos WHERE hash IN (%s)", placeholders), invalid...)
			if err != nil {
				return 0, 0, err
			}

			log.Println("Completed")
		} else {
			log.Warnf("%d echos are orphaned\n", len(invalid))
		}
	}

	return totalSize, uint64(total), nil
}
