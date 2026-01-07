package main

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	BackupDirectory  = "backups"
	BackupTimeFormat = "2006_01_02-15_04"
)

type Backup struct {
	Time time.Time
	Path string
}

type BackupList struct {
	Backups []*Backup
}

func (b *Backup) Exists() bool {
	_, err := os.Stat(b.Path)

	return !os.IsNotExist(err)
}

func (b *BackupList) Next() (time.Duration, time.Time) {
	now := time.Now().UTC()

	var newest time.Time

	for _, backup := range b.Backups {
		if newest.IsZero() || newest.Before(backup.Time) {
			newest = backup.Time
		}
	}

	if newest.IsZero() {
		return 0, now
	}

	next := newest.Add(time.Duration(config.Backup.Interval) * time.Hour)

	if next.Before(now) {
		return 0, now
	}

	return next.Sub(now), next
}

func (b *BackupList) Evict() error {
	if len(b.Backups) == 0 {
		return nil
	}

	clean := make([]*Backup, 0, len(b.Backups))

	for _, backup := range b.Backups {
		if !backup.Exists() {
			continue
		}

		clean = append(clean, backup)
	}

	sort.Slice(clean, func(i, j int) bool {
		return clean[i].Time.Before(clean[j].Time)
	})

	for len(clean) > config.Backup.KeepAmount {
		evict := clean[0]

		log.Printf("Evicting %s...\n", evict.Path)

		err := os.Remove(evict.Path)
		if err != nil {
			return err
		}

		clean = clean[1:]
	}

	return nil
}

func (b *BackupList) Create(now time.Time) error {
	backup, err := CreateBackup(now)
	if err != nil {
		return err
	}

	b.Backups = append(b.Backups, backup)

	return nil
}

func StartBackupLoop() error {
	if !config.Backup.Enabled {
		return nil
	}

	backups, err := ReadBackups()
	if err != nil {
		return err
	}

	go func() {
		for {
			err := backups.Evict()
			if err != nil {
				log.Warnf("Failed to evict backups: %v\n", err)
			}

			if count.Load() > 0 {
				wait, next := backups.Next()

				if wait > 0 {
					log.Printf("Next backup in %s\n", wait.Round(time.Second))

					time.Sleep(wait)
				}

				err = backups.Create(next)
				if err != nil {
					log.Warnf("Failed to create backup: %v\n", err)

					time.Sleep(5 * time.Second)
				}
			} else {
				wait := time.Duration(config.Backup.Interval) * time.Hour

				log.Println("Nothing to back up")
				log.Printf("Next backup in %s\n", wait.Round(time.Second))

				time.Sleep(wait)
			}
		}
	}()

	return nil
}

func ReadBackups() (*BackupList, error) {
	log.Println("Reading backups...")

	if _, err := os.Stat(BackupDirectory); os.IsNotExist(err) {
		err = os.MkdirAll(BackupDirectory, 0755)
		if err != nil {
			return nil, err
		}
	}

	backups := &BackupList{}

	err := filepath.WalkDir(BackupDirectory, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		name := filepath.Base(path)

		rgx := regexp.MustCompile(`(?m)^\d{4}_\d{2}_\d{2}-\d{2}_\d{2}\.tar\.gz$`)

		if !rgx.MatchString(name) {
			return nil
		}

		name = strings.TrimSuffix(name, ".tar.gz")

		created, err := time.ParseInLocation(BackupTimeFormat, name, time.UTC)
		if err != nil {
			log.Warnf("Failed to parse backup name: %v\n", err)

			return nil
		}

		backups.Backups = append(backups.Backups, &Backup{
			Time: created,
			Path: path,
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	return backups, nil
}

func CreateBackup(now time.Time) (*Backup, error) {
	log.Println("Creating new backup...")

	path := filepath.Join(BackupDirectory, now.Format(BackupTimeFormat)+".tar.gz")

	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	gzWriter := gzip.NewWriter(file)

	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)

	defer tarWriter.Close()

	err = WriteBackup(tarWriter)
	if err != nil {
		defer os.Remove(path)

		return nil, err
	}

	log.Println("Completed")

	return &Backup{
		Time: now,
		Path: path,
	}, nil
}

func WriteBackup(wr *tar.Writer) error {
	buf := make([]byte, 1024*1024)

	log.Println("Backing up database...")

	err := AddFileToBackup(wr, DatabasePath, buf)
	if err != nil {
		return err
	}

	if !config.Backup.BackupFiles {
		log.Println("Skipping file backup")

		return nil
	}

	log.Println("Reading storage...")

	dir, err := os.OpenFile(StorageDirectory, os.O_RDONLY, 0)
	if err != nil {
		return err
	}

	defer dir.Close()

	err = WriteFileToBackup(wr, dir, StorageDirectory, buf)
	if err != nil {
		return err
	}

	files, err := dir.ReadDir(0)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}

		return err
	}

	var (
		wg        sync.WaitGroup
		completed atomic.Uint64
		total     = len(files)
		done      = make(chan bool)
	)

	log.Printf("Backing up 0%% (0 of %d)\n", total)

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

				log.Printf("Backing up %.1f%% (%d of %d)\n", percentage, current, total)
			}
		}
	})

	defer func() {
		close(done)

		wg.Wait()
	}()

	for _, file := range files {
		path := filepath.Join(StorageDirectory, file.Name())

		path = filepath.ToSlash(path)

		err = AddFileToBackup(wr, path, buf)
		if err != nil {
			return err
		}

		completed.Add(1)
	}

	return nil
}

func AddFileToBackup(wr *tar.Writer, path string, buf []byte) error {
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return err
	}

	defer file.Close()

	return WriteFileToBackup(wr, file, path, buf)
}

func WriteFileToBackup(wr *tar.Writer, file *os.File, path string, buf []byte) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}

	header := &tar.Header{
		Name:    path,
		Size:    info.Size(),
		Mode:    int64(info.Mode()),
		ModTime: info.ModTime(),
	}

	if info.IsDir() {
		header.Typeflag = tar.TypeDir
	} else {
		header.Typeflag = tar.TypeReg
	}

	err = wr.WriteHeader(header)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return nil
	}

	_, err = io.CopyBuffer(wr, file, buf)
	return err
}
