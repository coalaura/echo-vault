package main

import (
	"archive/tar"
	"compress/gzip"
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

var (
	backupNamePattern = regexp.MustCompile(`^\d{4}_\d{2}_\d{2}-\d{2}_\d{2}\.tar\.gz$`)

	backupBufferPool = sync.Pool{
		New: func() any {
			buf := make([]byte, 1024*1024)

			return &buf
		},
	}
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

	return err == nil
}

func (b *BackupList) Next() (time.Duration, time.Time) {
	now := time.Now().UTC()

	var newest time.Time

	for _, backup := range b.Backups {
		if backup.Time.After(newest) {
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

	err := os.MkdirAll(BackupDirectory, 0755)
	if err != nil && !os.IsExist(err) {
		return nil, err
	}

	entries, err := os.ReadDir(BackupDirectory)
	if err != nil {
		return nil, err
	}

	backups := &BackupList{
		Backups: make([]*Backup, 0, len(entries)),
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		if !backupNamePattern.MatchString(name) {
			continue
		}

		created, err := time.ParseInLocation(BackupTimeFormat, strings.TrimSuffix(name, ".tar.gz"), time.UTC)
		if err != nil {
			log.Warnf("Failed to parse backup name: %v\n", err)

			continue
		}

		backups.Backups = append(backups.Backups, &Backup{
			Time: created,
			Path: filepath.Join(BackupDirectory, name),
		})
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
	tarWriter := tar.NewWriter(gzWriter)

	err = WriteBackup(tarWriter)

	tarErr := tarWriter.Close()
	gzErr := gzWriter.Close()

	if err != nil {
		os.Remove(path)

		return nil, err
	}

	if tarErr != nil {
		os.Remove(path)

		return nil, tarErr
	}

	if gzErr != nil {
		os.Remove(path)

		return nil, gzErr
	}

	log.Println("Completed")

	return &Backup{
		Time: now,
		Path: path,
	}, nil
}

func WriteBackup(wr *tar.Writer) error {
	log.Println("Backing up database...")

	err := addFileToBackup(wr, DatabasePath)
	if err != nil {
		return err
	}

	if vector != nil {
		err = writeDirectoryToBackup(wr, ClipDirectory)
		if err != nil {
			return err
		}
	}

	if !config.Backup.BackupFiles {
		log.Println("Skipping file backup")

		return nil
	}

	return writeDirectoryToBackup(wr, StorageDirectory)
}

func writeDirectoryToBackup(wr *tar.Writer, directory string) error {
	name := filepath.Base(directory)

	log.Printf("Reading %s...\n", name)

	var fileCount int

	err := filepath.WalkDir(directory, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		fileCount++

		return nil
	})

	if err != nil {
		return err
	}

	if fileCount == 0 {
		log.Printf("Skipping %s (empty)\n", name)

		return nil
	}

	var (
		completed atomic.Uint64
		done      = make(chan struct{})
		wg        sync.WaitGroup
	)

	log.Printf("Backing up %s 0%% (0 of %d)\n", name, fileCount)

	wg.Go(func() {
		ticker := time.NewTicker(time.Second)

		defer ticker.Stop()

		totalF := float64(fileCount)

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				current := completed.Load()
				percentage := float64(current) / totalF * 100

				log.Printf("Backing up %s %.1f%% (%d of %d)\n", name, percentage, current, fileCount)
			}
		}
	})

	err = filepath.WalkDir(directory, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		err = addEntryToBackup(wr, path, d)
		if err != nil {
			return err
		}

		if !d.IsDir() {
			completed.Add(1)
		}

		return nil
	})

	close(done)

	wg.Wait()

	if err != nil {
		return err
	}

	log.Printf("Backing up %s 100%% (%d of %d)\n", name, fileCount, fileCount)

	return nil
}

func addFileToBackup(wr *tar.Writer, path string) error {
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return err
	}

	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	return writeFileToBackup(wr, file, path, info)
}

func addEntryToBackup(wr *tar.Writer, path string, d fs.DirEntry) error {
	info, err := d.Info()
	if err != nil {
		return err
	}

	if d.IsDir() {
		return writeDirToBackup(wr, path, info)
	}

	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return err
	}

	defer file.Close()

	return writeFileToBackup(wr, file, path, info)
}

func writeDirToBackup(wr *tar.Writer, path string, info fs.FileInfo) error {
	header := tar.Header{
		Typeflag: tar.TypeDir,
		Name:     filepath.ToSlash(path) + "/",
		Mode:     int64(info.Mode()),
		ModTime:  info.ModTime(),
	}

	return wr.WriteHeader(&header)
}

func writeFileToBackup(wr *tar.Writer, file *os.File, path string, info fs.FileInfo) error {
	header := tar.Header{
		Typeflag: tar.TypeReg,
		Name:     filepath.ToSlash(path),
		Size:     info.Size(),
		Mode:     int64(info.Mode()),
		ModTime:  info.ModTime(),
	}

	err := wr.WriteHeader(&header)
	if err != nil {
		return err
	}

	bufPtr := backupBufferPool.Get().(*[]byte)

	defer backupBufferPool.Put(bufPtr)

	_, err = io.CopyBuffer(wr, file, *bufPtr)
	return err
}
