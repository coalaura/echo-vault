package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/goccy/go-yaml"
)

type EchoConfigServer struct {
	URL            string `yaml:"url"`
	Port           int    `yaml:"port"`
	UploadToken    string `yaml:"token"`
	MaxFileSize    int    `yaml:"max_file_size"`
	MaxConcurrency int    `yaml:"max_concurrency"`
}

type EchoConfigImages struct {
	Format  string `yaml:"format"`
	Effort  int    `yaml:"effort"`
	Quality int    `yaml:"quality"`
}

type EchoConfigVideos struct {
	Enabled  bool   `yaml:"enabled"`
	Format   string `yaml:"format"`
	Optimize bool   `yaml:"optimize"`
}

type EchoConfigGIFs struct {
	Enabled   bool `yaml:"enabled"`
	Optimize  bool `yaml:"optimize"`
	Framerate int  `yaml:"framerate"`
}

type EchoConfig struct {
	Server EchoConfigServer `yaml:"server"`
	Images EchoConfigImages `yaml:"images"`
	Videos EchoConfigVideos `yaml:"videos"`
	GIFs   EchoConfigGIFs   `yaml:"gifs"`
}

func NewDefaultConfig() EchoConfig {
	return EchoConfig{
		Server: EchoConfigServer{
			URL:            "http://localhost:8080/",
			Port:           8080,
			UploadToken:    "p4$$w0rd",
			MaxFileSize:    10,
			MaxConcurrency: 4,
		},
		Images: EchoConfigImages{
			Format:  "webp",
			Effort:  2,
			Quality: 90,
		},
		Videos: EchoConfigVideos{
			Enabled:  false,
			Format:   "mp4",
			Optimize: true,
		},
		GIFs: EchoConfigGIFs{
			Enabled:   true,
			Optimize:  false,
			Framerate: 15,
		},
	}
}

func LoadConfig() (*EchoConfig, error) {
	cfg := NewDefaultConfig()

	file, err := OpenFileForReading("config.yml")
	if !os.IsNotExist(err) {
		if err != nil {
			return nil, err
		}

		defer file.Close()

		err = yaml.NewDecoder(file).Decode(&cfg)
		if err != nil {
			return nil, err
		}
	} else {
		err = cfg.Store()
		if err != nil {
			return nil, err
		}
	}

	err = cfg.Validate()
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *EchoConfig) Validate() error {
	// server
	if c.Server.URL == "" {
		return fmt.Errorf("server.url is empty")
	} else if !strings.HasSuffix(c.Server.URL, "/") {
		c.Server.URL += "/"
	}

	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be 1-65535, got %d", c.Server.Port)
	}

	if c.Server.UploadToken == "" {
		return fmt.Errorf("server.token is empty")
	}

	if c.Server.MaxFileSize < 1 {
		return fmt.Errorf("server.max_file_size must be >= 1, got %d", c.Server.MaxFileSize)
	}

	if c.Server.MaxConcurrency < 1 {
		return fmt.Errorf("server.max_concurrency must be >= 1, got %d", c.Server.MaxConcurrency)
	}

	// images
	if !c.IsValidImageFormat(c.Images.Format) {
		return fmt.Errorf("images.format must be one of (webp, png, jpeg), got %q", c.Images.Format)
	}

	if c.Images.Effort < 1 || c.Images.Effort > 3 {
		return fmt.Errorf("images.effort must be 1-3, got %d", c.Images.Effort)
	}

	if c.Images.Quality < 1 || c.Images.Quality > 100 {
		return fmt.Errorf("images.quality must be 1-100, got %d", c.Images.Quality)
	}

	// videos
	if !c.IsValidVideoFormat(c.Videos.Format, false) {
		return fmt.Errorf("videos.format must be one of (mp4, webm, mov, m4v, mkv, gif), got %q", c.Videos.Format)
	}

	// gifs
	if c.GIFs.Framerate < 1 || c.GIFs.Framerate > 30 {
		return fmt.Errorf("gifs.framerate must be 1-30, got %d", c.GIFs.Framerate)
	}

	// check ffmpeg dependency
	if c.Videos.Enabled {
		_, err := exec.LookPath("ffmpeg")
		if err != nil {
			return errors.New("ffmpeg is required for videos.enabled")
		}
	}

	// check gifsicle dependency
	if c.GIFs.Enabled && c.GIFs.Optimize {
		_, err := exec.LookPath("gifsicle")
		if err != nil {
			return errors.New("gifsicle is required for gifs.optimize")
		}
	}

	return nil
}

func (c *EchoConfig) MaxFileSizeBytes() int64 {
	return int64(c.Server.MaxFileSize * 1024 * 1024)
}

func (c *EchoConfig) Addr() string {
	return fmt.Sprintf(":%d", c.Server.Port)
}

func (e *EchoConfig) Store() error {
	def := NewDefaultConfig()

	comments := yaml.CommentMap{
		"$.server.url":             {yaml.HeadComment(fmt.Sprintf(" base url of your instance (default: %v)", def.Server.URL))},
		"$.server.port":            {yaml.HeadComment(fmt.Sprintf(" port to run echo-vault on (default: %v)", def.Server.Port))},
		"$.server.token":           {yaml.HeadComment(fmt.Sprintf(" upload token for authentication, leave empty to disable auth (default: %v)", def.Server.UploadToken))},
		"$.server.max_file_size":   {yaml.HeadComment(fmt.Sprintf(" maximum upload file-size in MB (default: %vMB)", def.Server.MaxFileSize))},
		"$.server.max_concurrency": {yaml.HeadComment(fmt.Sprintf(" maximum concurrent uploads (default: %v)", def.Server.MaxConcurrency))},

		"$.images.format":  {yaml.HeadComment(fmt.Sprintf(" target format for images (webp, png or jpeg; default: %v)", def.Images.Format))},
		"$.images.effort":  {yaml.HeadComment(fmt.Sprintf(" quality/speed trade-off (1 = fast/big, 2 = medium, 3 = slow/small; default: %v)", def.Images.Effort))},
		"$.images.quality": {yaml.HeadComment(fmt.Sprintf(" webp quality (0-100, 100 = lossless; default: %v)", def.Images.Quality))},

		"$.videos.enabled":  {yaml.HeadComment(fmt.Sprintf(" allow video uploads (requires ffmpeg; default: %v)", def.Videos.Enabled))},
		"$.videos.format":   {yaml.HeadComment(fmt.Sprintf(" target format for videos (mp4, webm, mov, m4v, mkv or gif; default: %v)", def.Videos.Format))},
		"$.videos.optimize": {yaml.HeadComment(fmt.Sprintf(" optimize videos (compresses and re-encodes; default: %v)", def.Videos.Optimize))},

		"$.gifs.enabled":   {yaml.HeadComment(fmt.Sprintf(" allow gif uploads (default: %v)", def.GIFs.Enabled))},
		"$.gifs.optimize":  {yaml.HeadComment(fmt.Sprintf(" optimize gifs (compresses and re-encodes; requires gifsicle; default: %v)", def.GIFs.Optimize))},
		"$.gifs.framerate": {yaml.HeadComment(fmt.Sprintf(" gif target fps (1 - 30; default: %v)", def.GIFs.Framerate))},
	}

	file, err := OpenFileForWriting("config.yml")
	if err != nil {
		return err
	}

	defer file.Close()

	return yaml.NewEncoder(file, yaml.WithComment(comments)).Encode(e)
}

func (e *EchoConfig) IsValidImageFormat(format string) bool {
	switch format {
	case "webp", "png", "jpeg":
		return true
	}

	return false
}

func (e *EchoConfig) IsValidVideoFormat(format string, checkEnabled bool) bool {
	if format == "gif" {
		return !checkEnabled || e.GIFs.Enabled
	}

	if checkEnabled && !e.Videos.Enabled {
		return false
	}

	switch format {
	case "mp4", "webm", "mov", "m4v", "mkv":
		return true
	}

	return false
}
