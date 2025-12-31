package main

import (
	"os"
	"path/filepath"
)

const StorageDirectory = "storage"

func EnsureStorage() error {
	if _, err := os.Stat(StorageDirectory); !os.IsNotExist(err) {
		return err
	}

	return os.MkdirAll("./storage", 0755)
}

func storageAbs() (string, error) {
	info, err := os.Lstat(StorageDirectory)
	if err != nil {
		return "", err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return os.Readlink(StorageDirectory)
	}

	return filepath.Abs(StorageDirectory)
}
