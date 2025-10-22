package main

import (
	"context"
	"fmt"
	"os"
	"time"
)

type Echo struct {
	ID         int64  `json:"id"`
	Hash       string `json:"hash"`
	Name       string `json:"name"`
	Extension  string `json:"extension"`
	UploadSize int64  `json:"upload_size"`
	Timestamp  int64  `json:"timestamp"`
}

func (e *Echo) Fill() error {
	if e.Hash == "" {
		hash, err := database.Hash()
		if err != nil {
			return err
		}

		e.Hash = hash
	}

	if e.Timestamp == 0 {
		e.Timestamp = time.Now().Unix()
	}

	return nil
}

func (e *Echo) Storage() string {
	return fmt.Sprintf("./storage/%s.%s", e.Hash, e.Extension)
}

func (e *Echo) URL() string {
	return fmt.Sprintf("%s%s.%s", config.Server.URL, e.Hash, e.Extension)
}

func (e *Echo) Exists() bool {
	_, err := os.Stat(e.Storage())

	return err == nil
}

func (e *Echo) Unlink() error {
	file := e.Storage()

	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil
	}

	return os.Remove(file)
}

func (e *Echo) SaveUploadedFile(ctx context.Context, path string) (int64, error) {
	err := e.Fill()
	if err != nil {
		return 0, err
	}

	switch e.Extension {
	case "jpg", "jpeg", "png", "webp":
		file, err := OpenFileForReading(path)
		if err != nil {
			return 0, err
		}

		defer file.Close()

		e.Extension = config.Images.Format

		switch e.Extension {
		case "webp":
			return saveImageAsWebP(file, e.Storage())
		case "png":
			return saveImageAsPNG(file, e.Storage())
		case "jpeg":
			return saveImageAsJPEG(file, e.Storage())
		}
	case "gif":
		e.Extension = "gif"

		// return saveGIFAsGIF(file, e.Storage())
	case "mp4", "webm", "mov", "m4v", "mkv":
		e.Extension = config.Videos.Format

		switch e.Extension {
		case "mp4":
			return saveVideoAsMP4(ctx, path, e.Storage())
		case "webm":
			return saveVideoAsWebM(ctx, path, e.Storage())
		case "mov":
			return saveVideoAsMOV(ctx, path, e.Storage())
		case "m4v":
			return saveVideoAsM4V(ctx, path, e.Storage())
		case "mkv":
			return saveVideoAsMKV(ctx, path, e.Storage())
		case "gif":
			// return saveVideoAsGIF(file, e.Storage())
		}
	}

	return 0, fmt.Errorf("unsupported extension %q", e.Extension)
}
