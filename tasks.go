package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func handleTasks() {
	if len(os.Args) < 2 {
		return
	}

	task := os.Args[1]

	switch task {
	case "scan":
		log.MustFail(scanStorage())
	default:
		log.Warnf("Unknown task: %s\n", task)
	}

	os.Exit(0)
}

func scanStorage() error {
	log.Println("Scanning storage...")

	var echos []Echo

	path, err := storageAbs()
	if err != nil {
		return err
	}

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

	log.Printf("Checking %d echos...\n", len(echos))

	var create []Echo

	for index, echo := range echos {
		log.Printf("%d of %d\r", index+1, len(echos))

		exists, err := database.Exists(echo.Hash)
		if err != nil {
			return err
		}

		if exists {
			continue
		}

		create = append(create, echo)
	}

	if len(create) == 0 {
		log.Println("No new echos found.")

		return nil
	}

	log.Printf("Creating %d new echos...\n", len(create))

	for i, echo := range create {
		log.Printf("[%d/%d] Creating echo %s...\n", i+1, len(create), echo.Hash)

		err = database.Create(&echo)
		if err != nil {
			return err
		}
	}

	log.Println("Done.")

	return nil
}
