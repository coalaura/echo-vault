package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func (e *Echo) Storage() string {
	return fmt.Sprintf("./storage/%s.%s", e.Hash, e.Extension)
}

func (e *Echo) URL() string {
	if !strings.HasSuffix(config.BaseURL, "/") {
		config.BaseURL += "/"
	}

	return fmt.Sprintf("%s%s.%s", config.BaseURL, e.Hash, e.Extension)
}

func (e *Echo) Compress() {
	if e.Extension != "png" {
		return
	}

	_, err := exec.LookPath("pngquant")
	if err != nil {
		return
	}

	cmd := exec.Command("pngquant", "--force", "--ext", ".png", "256", e.Storage())

	_ = cmd.Start()
}

func (e *Echo) ExistsInStorage() bool {
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
