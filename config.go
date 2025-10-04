package main

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

type EchoConfigServer struct {
	URL         string `json:"url"`
	Port        uint16 `json:"port"`
	UploadToken string `json:"token"`
	MaxFileSize uint32 `json:"max_file_size"`
}

type EchoConfigSettings struct {
	Effort      uint8 `json:"effort"`
	Quality     uint8 `json:"quality"`
	ReEncodeGif bool  `json:"re_encode_gif"`
}

type EchoConfig struct {
	Server   EchoConfigServer   `json:"server"`
	Settings EchoConfigSettings `json:"settings"`
}

func loadConfig() error {
	// Defaults
	config = EchoConfig{
		Server: EchoConfigServer{
			URL:         "http://localhost:8080",
			Port:        8080,
			UploadToken: "p4$$w0rd",
			MaxFileSize: 10,
		},
		Settings: EchoConfigSettings{
			Effort:      4,
			Quality:     90,
			ReEncodeGif: true,
		},
	}

	file, err := os.OpenFile("config.yml", os.O_RDONLY, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return config.Store()
		}

		return err
	}

	defer file.Close()

	return yaml.NewDecoder(file).Decode(&config)
}

func (c *EchoConfig) MaxFileSizeBytes() int64 {
	return int64(c.Server.MaxFileSize * 1024 * 1024)
}

func (c *EchoConfig) Addr() string {
	return fmt.Sprintf(":%d", c.Server.Port)
}

func (e *EchoConfig) Store() error {
	comments := yaml.CommentMap{
		"$.server.url":           {yaml.HeadComment(" base url of your instance (default: http://localhost:8080)")},
		"$.server.port":          {yaml.HeadComment(" port to run echo-vault on (default: 8080)")},
		"$.server.token":         {yaml.HeadComment(" upload token for authentication, leave empty to disable auth (default: p4$$w0rd)")},
		"$.server.max_file_size": {yaml.HeadComment(" maximum upload file-size in MB (default: 10MB)")},

		"$.settings.effort":        {yaml.HeadComment(" quality/speed trade-off (0 = fast, 6 = slower-better; default: 4)")},
		"$.settings.quality":       {yaml.HeadComment(" webp quality (0-100, 100 = lossless; default: 90)")},
		"$.settings.re_encode_gif": {yaml.HeadComment(" re-encode gif's (removes metadata, cleans uploaded gif; default: true)")},
	}

	file, err := os.OpenFile("config.yml", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	defer file.Close()

	return yaml.NewEncoder(file, yaml.WithComment(comments)).Encode(e)
}
