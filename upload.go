package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"sync/atomic"
)

var limiter atomic.Int32

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	concurrent := limiter.Add(1)
	defer limiter.Add(-1)

	if concurrent > int32(config.Server.MaxConcurrency) {
		abort(w, http.StatusTooManyRequests)

		log.Warnln("upload: too many concurrent uploads")

		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, config.MaxFileSizeBytes())

	mr, err := r.MultipartReader()
	if err != nil {
		abort(w, http.StatusBadRequest)

		log.Warnln("upload: not multipart")
		log.Warnln(err)

		return
	}

	var part *multipart.Part

	for {
		p, err := mr.NextPart()
		if err != nil {
			if err == io.EOF {
				break
			}

			abort(w, http.StatusBadRequest)

			log.Warnln("upload: failed to read part")
			log.Warnln(err)

			return
		}

		if p.FormName() == "upload" {
			part = p

			break
		}

		p.Close()
	}

	if part == nil {
		abort(w, http.StatusBadRequest)

		log.Warnln("upload: missing 'upload' part")

		return
	}

	defer part.Close()

	timer := NewTimer().Start("read")

	var sniff bytes.Buffer

	limited := io.LimitedReader{
		R: part,
		N: MaxSniffBytes,
	}

	_, err = io.Copy(&sniff, &limited)
	if err != nil && err != io.EOF {
		abort(w, http.StatusBadRequest)

		log.Warnln("upload: failed to read file header")
		log.Warnln(err)

		return
	}

	sniffed := sniffType(sniff.Bytes())

	if sniffed == "" || (!config.IsValidImageFormat(sniffed) && !config.IsValidVideoFormat(sniffed, true)) {
		abort(w, http.StatusBadRequest)

		log.Warnln("upload: invalid/unrecognized filetype")

		return
	}

	echo := &Echo{
		Name:      part.FileName(),
		Extension: sniffed,
	}

	file, path, err := OpenTempFileForWriting()
	if err != nil {
		abort(w, http.StatusInternalServerError)

		log.Warnln("upload: failed to open temporary file")
		log.Warnln(err)

		return
	}

	defer file.Close()
	defer os.Remove(path)

	_, err = file.Write(sniff.Bytes())
	if err != nil {
		abort(w, http.StatusInternalServerError)

		log.Warnln("upload: failed to write file header")
		log.Warnln(err)

		return
	}

	_, err = io.Copy(file, part)
	if err != nil {
		abort(w, http.StatusInternalServerError)

		log.Warnln("upload: failed to write file body")
		log.Warnln(err)

		return
	}

	file.Close()

	echo.UploadSize = file.N

	timer.Stop("read").Start("write")

	size, err := echo.SaveUploadedFile(r.Context(), path)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		log.Warnln("upload: failed to save uploaded file")
		log.Warnln(err)

		os.Remove(echo.Storage())

		return
	}

	timer.Stop("write").Start("store")

	err = database.Create(echo)
	if err != nil {
		abort(w, http.StatusInternalServerError)

		log.Warnln("upload: failed to create echo in database")
		log.Warnln(err)

		os.Remove(echo.Storage())

		return
	}

	timer.Stop("store")

	okay(w, "application/json")

	json.NewEncoder(w).Encode(map[string]any{
		"hash":      echo.Hash,
		"sniffed":   sniffed,
		"extension": echo.Extension,
		"url":       echo.URL(),
		"size":      byteCountSI(size),
		"timing":    timer,
	})
}

func byteCountSI(b int64) string {
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
