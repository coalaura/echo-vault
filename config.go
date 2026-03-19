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
	Direct         bool   `yaml:"direct"`
	UploadToken    string `yaml:"token"`
	MaxFileSize    int    `yaml:"max_file_size"`
	MaxConcurrency int    `yaml:"max_concurrency"`
	DeleteOrphans  bool   `yaml:"delete_orphans"`
}

type EchoConfigBackup struct {
	Enabled     bool `yaml:"enabled"`
	Interval    int  `yaml:"interval"`
	KeepAmount  int  `yaml:"keep_amount"`
	BackupFiles bool `yaml:"backup_files"`
}

type EchoConfigImages struct {
	Format  string `yaml:"format"`
	Effort  int    `yaml:"effort"`
	Quality int    `yaml:"quality"`
}

type EchoConfigVideos struct {
	Enabled bool `yaml:"enabled"`
}

type EchoConfigGIFs struct {
	Enabled bool   `yaml:"enabled"`
	Format  string `yaml:"format"`
}

type EchoConfig struct {
	ffmpeg string

	Server EchoConfigServer `yaml:"server"`
	Backup EchoConfigBackup `yaml:"backup"`
	Images EchoConfigImages `yaml:"images"`
	Videos EchoConfigVideos `yaml:"videos"`
	GIFs   EchoConfigGIFs   `yaml:"gifs"`
}

func NewDefaultConfig() EchoConfig {
	return EchoConfig{
		Server: EchoConfigServer{
			URL:            "http://localhost:8080/",
			Port:           8080,
			Direct:         false,
			UploadToken:    "p4$$w0rd",
			MaxFileSize:    20,
			MaxConcurrency: 4,
			DeleteOrphans:  false,
		},
		Backup: EchoConfigBackup{
			Enabled:     true,
			Interval:    5 * 24,
			KeepAmount:  4,
			BackupFiles: true,
		},
		Images: EchoConfigImages{
			Format:  "webp",
			Effort:  2,
			Quality: 90,
		},
		Videos: EchoConfigVideos{
			Enabled: false,
		},
		GIFs: EchoConfigGIFs{
			Enabled: true,
			Format:  "webp",
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
	}

	err = cfg.Validate()
	if err != nil {
		return nil, err
	}

	return &cfg, cfg.Store()
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

	// backup
	if c.Backup.Enabled {
		if c.Backup.Interval <= 0 {
			return fmt.Errorf("backup.interval must be >= 1, got %d", c.Backup.Interval)
		}

		if c.Backup.KeepAmount <= 0 {
			return fmt.Errorf("backup.keep_amountt must be >= 1, got %d", c.Backup.KeepAmount)
		}
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

	// gifs
	if c.GIFs.Format != "gif" && c.GIFs.Format != "webp" {
		return fmt.Errorf("gifs.format must be one of (gif, webp), got %q", c.GIFs.Format)
	}

	// check ffmpeg dependency
	if c.Videos.Enabled || (c.GIFs.Enabled && c.GIFs.Format == "gif") {
		ffmpeg, err := exec.LookPath("ffmpeg")
		if err != nil {
			return errors.New("ffmpeg is required for video/gif in/output")
		}

		c.ffmpeg = ffmpeg
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
		"$.server.direct":          {yaml.HeadComment(fmt.Sprintf(" only append the filename to the base url, no \"/i/\" (for custom endpoints; default: %v)", def.Server.Direct))},
		"$.server.token":           {yaml.HeadComment(fmt.Sprintf(" upload token for authentication, leave empty to disable auth (default: %v)", def.Server.UploadToken))},
		"$.server.max_file_size":   {yaml.HeadComment(fmt.Sprintf(" maximum upload file-size in MB (default: %vMB)", def.Server.MaxFileSize))},
		"$.server.max_concurrency": {yaml.HeadComment(fmt.Sprintf(" maximum concurrent uploads (default: %v)", def.Server.MaxConcurrency))},
		"$.server.delete_orphans":  {yaml.HeadComment(fmt.Sprintf(" if echos without their file should be deleted (default: %v)", def.Server.DeleteOrphans))},

		"$.backup.enabled":      {yaml.HeadComment(fmt.Sprintf(" if backups should be created (default: %v)", def.Backup.Enabled))},
		"$.backup.interval":     {yaml.HeadComment(fmt.Sprintf(" how often backups should be created (in hours; default: %v)", def.Backup.Interval))},
		"$.backup.keep_amount":  {yaml.HeadComment(fmt.Sprintf(" how many backups to keep before deleting the oldest (default: %v)", def.Backup.KeepAmount))},
		"$.backup.backup_files": {yaml.HeadComment(fmt.Sprintf(" if files (images/videos) should be included in backups (without, only the database is backed up; default: %v)", def.Backup.BackupFiles))},

		"$.images.format":  {yaml.HeadComment(fmt.Sprintf(" target format for images (webp, png or jpeg; default: %v)", def.Images.Format))},
		"$.images.effort":  {yaml.HeadComment(fmt.Sprintf(" quality/speed trade-off (1 = fast/big, 2 = medium, 3 = slow/small; default: %v)", def.Images.Effort))},
		"$.images.quality": {yaml.HeadComment(fmt.Sprintf(" webp quality (0-100, 100 = lossless; default: %v)", def.Images.Quality))},

		"$.videos.enabled": {yaml.HeadComment(fmt.Sprintf(" allow video uploads (requires ffmpeg/ffprobe; default: %v)", def.Videos.Enabled))},

		"$.gifs.enabled": {yaml.HeadComment(fmt.Sprintf(" allow gif uploads (requires ffmpeg/ffprobe; default: %v)", def.GIFs.Enabled))},
		"$.gifs.format":  {yaml.HeadComment(fmt.Sprintf(" target format for gifs (gif or webp; default: %v)", def.GIFs.Format))},
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
	if format == "gif" || format == "webp" {
		// Both GIF and animated WebP require GIF processing pipeline
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
