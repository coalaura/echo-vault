package main

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
)

func OpenFileForReading(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY, 0)
}

func OpenFileForWriting(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
}

func GetTempFilePath() (string, error) {
	b := make([]byte, 16)

	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return filepath.Join(os.TempDir(), "echo_"+hex.EncodeToString(b)), nil
}

func OpenTempFileForWriting() (*CountWriter, string, error) {
	path, err := GetTempFilePath()
	if err != nil {
		return nil, "", err
	}

	file, err := OpenCountWriter(path)
	if err != nil {
		return nil, "", err
	}

	return file, path, nil
}
