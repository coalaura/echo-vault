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

	"github.com/coalaura/schgo"
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

	schema, err := schgo.NewSchema(db)
	if err != nil {
		db.Close()

		return nil, err
	}

	table := schema.Table("echos")

	table.Primary("id", "INTEGER")

	table.Column("hash", "TEXT").NotNull().Unique()
	table.Column("name", "TEXT").NotNull()
	table.Column("extension", "TEXT").NotNull()
	table.Column("animated", "INTEGER").NotNull().Default("0")
	table.Column("size", "INTEGER").NotNull().Default("0")
	table.Column("upload_size", "INTEGER").NotNull().Default("0")
	table.Column("timestamp", "INTEGER").NotNull().Default("0")

	table.Index("idx_echos_timestamp", "timestamp")

	err = schema.Apply()
	if err != nil {
		db.Close()

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
	var e Echo

	err := d.QueryRowContext(ctx, "SELECT id, hash, name, extension, animated, size, upload_size, timestamp FROM echos WHERE hash = ? LIMIT 1", hash).Scan(&e.ID, &e.Hash, &e.Name, &e.Extension, &e.Animated, &e.Size, &e.UploadSize, &e.Timestamp)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &e, nil
}

func (d *EchoDatabase) FindAll(ctx context.Context, offset, limit int) ([]Echo, error) {
	rows, err := d.QueryContext(ctx, "SELECT id, hash, name, extension, animated, size, upload_size, timestamp FROM echos ORDER BY timestamp DESC LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var echos []Echo

	for rows.Next() {
		var e Echo

		err := rows.Scan(&e.ID, &e.Hash, &e.Name, &e.Extension, &e.Animated, &e.Size, &e.UploadSize, &e.Timestamp)
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

func (d *EchoDatabase) FindByHashes(ctx context.Context, hashes []string) ([]Echo, error) {
	if len(hashes) == 0 {
		return nil, nil
	}

	placeholders := strings.Repeat("?,", len(hashes))
	placeholders = placeholders[:len(placeholders)-1]

	var b strings.Builder

	b.WriteString("SELECT id, hash, name, extension, animated, size, upload_size, timestamp FROM echos WHERE hash IN (")
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
		var e Echo

		err := rows.Scan(&e.ID, &e.Hash, &e.Name, &e.Extension, &e.Animated, &e.Size, &e.UploadSize, &e.Timestamp)
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

func (d *EchoDatabase) Create(ctx context.Context, echo *Echo) error {
	err := echo.Fill(ctx)
	if err != nil {
		return err
	}

	_, err = d.Exec("INSERT INTO echos (hash, name, extension, animated, size, upload_size, timestamp) VALUES (?, ?, ?, ?, ?, ?, ?)", echo.Hash, echo.Name, echo.Extension, echo.Animated, echo.Size, echo.UploadSize, echo.Timestamp)
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

	err = d.Optimize(20)
	if err != nil {
		log.Warnf("Verify optimization failed: %v\n", err)
	}

	return totalSize, uint64(total), nil
}

func (d *EchoDatabase) Optimize(thresholdMB int64) error {
	var (
		pageCount     int64
		freelistCount int64
		pageSize      int64
	)

	err := d.QueryRow("PRAGMA page_count").Scan(&pageCount)
	if err != nil {
		return err
	}

	err = d.QueryRow("PRAGMA freelist_count").Scan(&freelistCount)
	if err != nil {
		return err
	}

	err = d.QueryRow("PRAGMA page_size").Scan(&pageSize)
	if err != nil {
		return err
	}

	if freelistCount == 0 {
		return nil
	}

	wastedBytes := freelistCount * pageSize
	wastedMB := wastedBytes / 1024 / 1024

	if wastedMB < thresholdMB {
		return nil
	}

	totalBytes := pageCount * pageSize
	fragPercent := (float64(wastedBytes) / float64(totalBytes)) * 100

	log.Printf("Database fragmentation: %.2f%% (%d MB wasted). Optimization started...\n", fragPercent, wastedMB)

	_, err = d.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	if err != nil {
		return err
	}

	_, err = d.Exec("VACUUM")
	if err != nil {
		return err
	}

	log.Println("Database optimization complete.")

	return nil
}
