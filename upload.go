package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)

		log.Warnln("upload: failed to read form")
		log.Warnln(err)

		return
	}

	file, header, err := r.FormFile("upload")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)

		log.Warnln("upload: failed to read file")
		log.Warnln(err)

		return
	}

	defer file.Close()

	if header.Size > config.MaxFileSizeBytes() {
		w.WriteHeader(http.StatusRequestEntityTooLarge)

		log.Warnln("upload: file too big")

		return
	}

	var tmp [64]byte

	n, err := file.Read(tmp[:])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)

		log.Warnln("upload: failed to read file header")
		log.Warnln(err)

		return
	}

	echo := &Echo{
		Name:       header.Filename,
		UploadSize: header.Size,
	}

	contentType := http.DetectContentType(tmp[:n])

	switch contentType {
	case "image/jpeg":
		echo.Extension = "jpg"
	case "image/png":
		echo.Extension = "png"
	case "image/gif":
		echo.Extension = "gif"
	case "image/webp":
		echo.Extension = "webp"
	default:
		w.WriteHeader(http.StatusBadRequest)

		log.Warnf("upload: invalid file type %q", contentType)

		return
	}

	size, err := echo.SaveUploadedFile(header)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		log.Warnf("upload: failed to save uploaded file: %v\n", err)
		log.Warnln(err)

		return
	}

	err = database.Create(echo)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		log.Warnf("upload: failed to create echo in database: %v\n", err)
		log.Warnln(err)

		return
	}

	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"hash":      echo.Hash,
		"extension": echo.Extension,
		"url":       echo.URL(),
		"size":      byteCountSI(size),
	})
}

func byteCountSI(b int) string {
	if b < 1000 {
		return fmt.Sprintf("%d B", b)
	}

	div, exp := 1000, 0

	for n := b / 1000; n >= 1000; n /= 1000 {
		div *= 1000
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}
