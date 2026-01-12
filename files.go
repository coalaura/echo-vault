package main

import (
	"os"
)

func OpenFileForReading(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY, 0)
}

func OpenFileForWriting(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
}

func OpenTempFileForWriting() (*CountWriter, string, error) {
	file, err := os.CreateTemp("", "echo_upl_*")
	if err != nil {
		return nil, "", err
	}

	return &CountWriter{
		File: file,
	}, file.Name(), nil
}
