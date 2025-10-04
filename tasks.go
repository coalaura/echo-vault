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

		ext := filepath.Ext(path)

		if ext == ".webp" || ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" {
			hash := strings.TrimSuffix(filepath.Base(path), ext)

			// Same shit, different day
			if ext == ".jpeg" {
				ext = ".jpg"
			}

			info, err := entry.Info()
			if err != nil {
				return err
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

	log.Printf("Checking %d echos...\n", len(echos))

	var (
		newEchos  []Echo
		converted int
	)

	for index, echo := range echos {
		log.Printf("%d of %d\r", index+1, len(echos))

		exists, err := database.Exists(echo.Hash)
		if err != nil {
			return err
		}

		if echo.Extension == "jpg" || echo.Extension == "png" || echo.Extension == "gif" {
			ok, err := convertEchoToWebP(&echo)
			if err != nil {
				return err
			}

			if ok {
				if exists {
					err = database.SetExtension(echo.Hash, "webp")
					if err != nil {
						return err
					}
				}

				converted++
			}
		}

		if !exists {
			newEchos = append(newEchos, echo)
		}
	}

	if converted > 0 {
		log.Printf("Converted %d echos to webp.\n", converted)
	}

	if len(newEchos) == 0 {
		log.Println("No new echos found.")

		return nil
	}

	log.Printf("Creating %d new echos...\n", len(newEchos))

	for i, echo := range newEchos {
		log.Printf("[%d/%d] Creating echo %s...\n", i+1, len(newEchos), echo.Hash)

		err = database.Create(&echo)
		if err != nil {
			return err
		}
	}

	log.Println("Done.")

	return nil
}
