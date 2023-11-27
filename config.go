package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type EchoConfig struct {
	BaseURL       string `json:"base_url"`
	Port          int64  `json:"port"`
	UploadToken   string `json:"upload_token"`
	MaxFileSizeMB int64  `json:"max_file_size_mb"`
}

func loadConfig() error {
	// Defaults
	config = EchoConfig{
		BaseURL:       "http://localhost:8080",
		Port:          8080,
		UploadToken:   "p4$$w0rd",
		MaxFileSizeMB: 10,
	}

	if _, err := os.Stat("./config.json"); os.IsNotExist(err) {
		b, _ := json.MarshalIndent(config, "", "\t")

		return os.WriteFile("./config.json", b, 0644)
	}

	b, err := os.ReadFile("./config.json")
	if err != nil {
		return err
	}

	return json.Unmarshal(b, &config)
}

func (c *EchoConfig) MaxFileSize() int64 {
	return c.MaxFileSizeMB * 1024 * 1024
}

func (c *EchoConfig) Addr() string {
	return fmt.Sprintf(":%d", c.Port)
}
