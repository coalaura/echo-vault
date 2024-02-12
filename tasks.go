package main

import (
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
		log.MustPanic(scanStorage())
	default:
		log.WarningF("Unknown task: %s\n", task)
	}

	os.Exit(0)
}

func scanStorage() error {
	log.Info("Scanning storage...")

	var echos []Echo

	path, err := storageAbs()
	if err != nil {
		return err
	}

	err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)

		if ext == ".webp" || ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" {
			hash := strings.TrimSuffix(filepath.Base(path), ext)

			// Same shit, different day
			if ext == ".jpeg" {
				ext = ".jpg"
			}

			echos = append(echos, Echo{
				Hash:       hash,
				Name:       info.Name(),
				UploadSize: info.Size(),
				Extension:  ext[1:],
				Timestamp:  info.ModTime().Unix(),
			})
		}

		return nil
	})
	if err != nil {
		return err
	}

	log.InfoF("Checking %d echos...\n", len(echos))

	var (
		newEchos []Echo
	)

	for index, echo := range echos {
		log.NoteF("%d of %d completed...\r", index+1, len(echos))

		exists, err := database.Exists(echo.Hash)
		if err != nil {
			return err
		}

		if echo.Extension == "jpg" || echo.Extension == "png" {
			err = convertEchoToWebP(&echo)
			if err != nil {
				return err
			}

			if exists {
				err = database.SetExtension(echo.Hash, "webp")
				if err != nil {
					return err
				}
			}
		}

		if !exists {
			newEchos = append(newEchos, echo)
		}
	}

	if len(newEchos) == 0 {
		log.Info("No new echos found.")

		return nil
	}

	log.InfoF("Creating %d new echos...\n", len(newEchos))

	for i, echo := range newEchos {
		log.NoteF("[%d/%d] Creating echo %s...\n", i+1, len(newEchos), echo.Hash)

		err = database.Create(&echo)
		if err != nil {
			return err
		}
	}

	log.Info("Done.")

	return nil
}
