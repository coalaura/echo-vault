package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func handleTasks() bool {
	if len(os.Args) < 2 {
		return false
	}

	log.SetDate("")

	task := os.Args[1]

	switch task {
	case "scan":
		log.MustFail(taskScanStorage())
	case "clear-tags":
		log.MustFail(taskClearTags())
	default:
		fmt.Printf("Unknown task: %s\n", task)
		fmt.Println()
		fmt.Println("Available tasks:")
		fmt.Println("  scan        Scan storage directory for new files and add them to the database")
		fmt.Println("  clear-tags  Remove all generated tags, descriptions, and vector embeddings")
	}

	return true
}

func taskScanStorage() error {
	path, err := storageAbs()
	if err != nil {
		return err
	}

	log.Printf("Scanning %s...\n", path)

	var echos []Echo

	err = filepath.WalkDir(path, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}

		ext := strings.ToLower(filepath.Ext(path))
		if len(ext) < 2 {
			return nil
		}

		ext = ext[1:]
		if ext == "jpg" {
			ext = "jpeg"
		}

		if config.IsValidImageFormat(ext) || config.IsValidVideoFormat(ext, false) {
			hash := strings.TrimSuffix(filepath.Base(path), "."+ext)

			info, err := entry.Info()
			if err != nil {
				return err
			}

			echos = append(echos, Echo{
				Hash:       hash,
				Name:       info.Name(),
				UploadSize: info.Size(),
				Extension:  ext,
				Timestamp:  info.ModTime().Unix(),
			})
		}

		return nil
	})

	if err != nil {
		return err
	}

	log.Printf("Found %d files, checking database...\n", len(echos))

	var create []Echo

	for _, echo := range echos {
		exists, err := database.Exists(echo.Hash)
		if err != nil {
			return err
		}

		if !exists {
			create = append(create, echo)
		}
	}

	if len(create) == 0 {
		log.Println("All files already in database, nothing to do.")

		return nil
	}

	log.Printf("Adding %d new entries to database...\n", len(create))

	for i, echo := range create {
		log.Printf("  [%d/%d] %s\n", i+1, len(create), echo.Hash)

		err = database.Create(context.Background(), &echo)
		if err != nil {
			return err
		}
	}

	log.Printf("Done! Added %d entries.\n", len(create))

	return nil
}

func taskClearTags() error {
	confirmed, err := log.ConfirmWithEcho("This will remove all tags, descriptions, and vector embeddings. Continue?", false, " ")
	if err != nil {
		return err
	}

	if !confirmed {
		log.Println("Cancelled.")

		return nil
	}

	log.Print("Clearing tags from database... ")

	_, err = database.Exec("UPDATE echos SET description = NULL, phrases = NULL, safety = NULL")
	if err != nil {
		log.Println("failed")

		return err
	}

	log.Println("done")

	log.Print("Removing vector database... ")

	err = os.RemoveAll(TagsDirectory)
	if err != nil {
		log.Println("failed")

		return err
	}

	log.Println("done")

	log.Println("All tags cleared successfully.")

	return nil
}
