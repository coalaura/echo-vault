package main

import (
	"flag"
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
	var convertOnly bool

	flag.BoolVar(&convertOnly, "convert-only", false, "Don't add new echos, only convert existing ones.")
	flag.Parse()

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

		if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" {
			hash := strings.TrimSuffix(filepath.Base(path), ext)

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
		convertedToWebP = 0

		newEchos []Echo
	)

	for index, echo := range echos {
		log.NoteF("%d of %d completed...\r", index+1, len(echos))

		if echo.Extension == "jpg" || echo.Extension == "png" {
			err = convertEchoToWebP(&echo)
			if err != nil {
				return err
			}

			convertedToWebP++
		}

		// Skip check if we are only converting
		if !convertOnly {
			exists, err := database.Exists(echo.Hash)
			if err != nil {
				return err
			}

			if !exists {
				newEchos = append(newEchos, echo)
			}
		}
	}

	if convertedToWebP > 0 {
		log.InfoF("Updating %d webp echos...\n", convertedToWebP)

		// Easier than tracking hashes and same result
		_, err := database.Exec("UPDATE echos SET extension = ? WHERE extension IN (?, ?)", "webp", "jpg", "png")
		if err != nil {
			return err
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
