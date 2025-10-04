package main

import (
	"fmt"
	"os"
	"strings"
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
	if !strings.HasSuffix(config.Server.URL, "/") {
		config.Server.URL += "/"
	}

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
