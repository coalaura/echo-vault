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
	"strings"
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

	return next.Sub(now), next
}

func (b *BackupList) Evict() error {
	if len(b.Backups) == 0 {
		return nil
	}

	oldest := time.Now().UTC().Add(-time.Duration(config.Backup.KeepTime) * time.Hour)

	evicted := make([]*Backup, 0, len(b.Backups))

	for _, backup := range b.Backups {
		if backup.Time.After(oldest) {
			evicted = append(evicted, backup)

			continue
		}

		log.Printf("Evicting old backup: %s\n", backup.Path)

		err := os.Remove(backup.Path)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	b.Backups = evicted

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
	log.Println("Backing up database...")

	err := AddFileToBackup(wr, DatabasePath)
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

	err = WriteFileToBackup(wr, dir, StorageDirectory)
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

	log.Printf("Backing up %d files...\n", len(files))

	for _, file := range files {
		path := filepath.Join(StorageDirectory, file.Name())

		path = filepath.ToSlash(path)

		err = AddFileToBackup(wr, path)
		if err != nil {
			return err
		}
	}

	return nil
}

func AddFileToBackup(wr *tar.Writer, path string) error {
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return err
	}

	defer file.Close()

	return WriteFileToBackup(wr, file, path)
}

func WriteFileToBackup(wr *tar.Writer, file *os.File, path string) error {
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

	if !info.IsDir() {
		_, err = io.Copy(wr, file)
		if err != nil {
			return err
		}
	}

	return nil
}
