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

type EchoConfigUI struct {
	NoSafetyBlur bool     `yaml:"no_safety_blur"`
	IgnoreSafety []string `yaml:"ignore_safety"`
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
	Enabled  bool   `yaml:"enabled"`
	Format   string `yaml:"format"`
	Optimize bool   `yaml:"optimize"`
}

type EchoConfigGIFs struct {
	Enabled      bool `yaml:"enabled"`
	Optimize     bool `yaml:"optimize"`
	Effort       int  `yaml:"effort"`
	Quality      int  `yaml:"quality"`
	MaxColors    int  `yaml:"max_colors"`
	MaxFramerate int  `yaml:"max_framerate"`
	MaxWidth     int  `yaml:"max_width"`
}

type EchoConfigAI struct {
	OpenRouterToken string `yaml:"openrouter_token"`
	TaggingModel    string `yaml:"tagging_model"`
	EmbeddingModel  string `yaml:"embedding_model"`
	MinSimilarity   int    `yaml:"min_similarity"`
	ReTagEmpty      bool   `yaml:"re_tag_empty"`
}

type EchoConfig struct {
	ffmpeg   string
	ffprobe  string
	gifsicle string

	Server EchoConfigServer `yaml:"server"`
	UI     EchoConfigUI     `yaml:"ui"`
	Backup EchoConfigBackup `yaml:"backup"`
	Images EchoConfigImages `yaml:"images"`
	Videos EchoConfigVideos `yaml:"videos"`
	GIFs   EchoConfigGIFs   `yaml:"gifs"`
	AI     EchoConfigAI     `yaml:"ai"`
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
		UI: EchoConfigUI{
			NoSafetyBlur: false,
			IgnoreSafety: []string{},
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
			Enabled:  false,
			Format:   "mp4",
			Optimize: true,
		},
		GIFs: EchoConfigGIFs{
			Enabled:      false,
			Optimize:     true,
			Effort:       2,
			Quality:      90,
			MaxColors:    256,
			MaxFramerate: 15,
			MaxWidth:     480,
		},
		AI: EchoConfigAI{
			OpenRouterToken: "",
			TaggingModel:    "google/gemma-3-27b-it",
			EmbeddingModel:  "openai/text-embedding-3-small",
			MinSimilarity:   25,
			ReTagEmpty:      false,
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

	// ui
	if len(c.UI.IgnoreSafety) > 0 {
		for i, tag := range c.UI.IgnoreSafety {
			if !IsValidSafety(tag) {
				return fmt.Errorf("ui.ignore_safety[%d] is not a valid safety tag: %q", i, tag)
			}
		}
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

	// videos
	if !c.IsValidVideoFormat(c.Videos.Format, false) {
		return fmt.Errorf("videos.format must be one of (mp4, webm, mov, m4v, mkv, gif), got %q", c.Videos.Format)
	}

	// gifs
	if c.GIFs.Effort < 1 || c.GIFs.Effort > 3 {
		return fmt.Errorf("gifs.effort must be 1-3, got %d", c.GIFs.Effort)
	}

	if c.GIFs.Quality < 1 || c.GIFs.Quality > 100 {
		return fmt.Errorf("gifs.quality must be 1-100, got %d", c.GIFs.Quality)
	}

	if c.GIFs.MaxColors < 2 || c.GIFs.MaxColors > 256 {
		return fmt.Errorf("gifs.max_colors must be 2-256, got %d", c.GIFs.MaxColors)
	}

	if c.GIFs.MaxFramerate < 1 || c.GIFs.MaxFramerate > 30 {
		return fmt.Errorf("gifs.max_framerate must be 1-30, got %d", c.GIFs.MaxFramerate)
	}

	if c.GIFs.MaxWidth < 1 || c.GIFs.MaxWidth > 1024 {
		return fmt.Errorf("gifs.max_width must be 1-1024, got %d", c.GIFs.MaxWidth)
	}

	// ai
	if c.AI.OpenRouterToken != "" {
		if c.AI.TaggingModel == "" {
			return errors.New("ai.tagging_model must be set")
		}

		if c.AI.EmbeddingModel == "" {
			return errors.New("ai.embedding_model must be set")
		}

		if c.AI.MinSimilarity < 0 || c.AI.MinSimilarity > 100 {
			return fmt.Errorf("ai.min_similarity must be 0-100, got %d", c.AI.MinSimilarity)
		}
	}

	// check ffmpeg dependency
	if c.Videos.Enabled || c.GIFs.Enabled {
		ffmpeg, err := exec.LookPath("ffmpeg")
		if err != nil {
			return errors.New("ffmpeg is required for video/gif input")
		}

		c.ffmpeg = ffmpeg

		if c.Videos.Format == "gif" || c.GIFs.Enabled {
			ffprobe, err := exec.LookPath("ffprobe")
			if err != nil {
				return errors.New("ffprobe is required for gif input/output")
			}

			c.ffprobe = ffprobe
		}
	}

	// check gifsicle dependency
	if (c.GIFs.Enabled || (c.Videos.Enabled && c.Videos.Format == "gif")) && c.GIFs.Optimize {
		gifsicle, err := exec.LookPath("gifsicle")
		if err != nil {
			return errors.New("gifsicle is required for gifs.optimize")
		}

		c.gifsicle = gifsicle
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

		"$.ui.no_safety_blur": {yaml.HeadComment(fmt.Sprintf(" fully ignore safety tags in the UI (default: %v)", def.UI.NoSafetyBlur))},
		"$.ui.ignore_safety":  {yaml.HeadComment(fmt.Sprintf(" list of safety tags to ignore in the UI (default: %v)", def.UI.IgnoreSafety))},

		"$.backup.enabled":      {yaml.HeadComment(fmt.Sprintf(" if backups should be created (default: %v)", def.Backup.Enabled))},
		"$.backup.interval":     {yaml.HeadComment(fmt.Sprintf(" how often backups should be created (in hours; default: %v)", def.Backup.Interval))},
		"$.backup.keep_amount":  {yaml.HeadComment(fmt.Sprintf(" how many backups to keep before deleting the oldest (default: %v)", def.Backup.KeepAmount))},
		"$.backup.backup_files": {yaml.HeadComment(fmt.Sprintf(" if files (images/videos) should be included in backups (without, only the database is backed up; default: %v)", def.Backup.BackupFiles))},

		"$.images.format":  {yaml.HeadComment(fmt.Sprintf(" target format for images (webp, png or jpeg; default: %v)", def.Images.Format))},
		"$.images.effort":  {yaml.HeadComment(fmt.Sprintf(" quality/speed trade-off (1 = fast/big, 2 = medium, 3 = slow/small; default: %v)", def.Images.Effort))},
		"$.images.quality": {yaml.HeadComment(fmt.Sprintf(" webp quality (0-100, 100 = lossless; default: %v)", def.Images.Quality))},

		"$.videos.enabled":  {yaml.HeadComment(fmt.Sprintf(" allow video uploads (requires ffmpeg/ffprobe; default: %v)", def.Videos.Enabled))},
		"$.videos.format":   {yaml.HeadComment(fmt.Sprintf(" target format for videos (mp4, webm, mov, m4v, mkv or gif; default: %v)", def.Videos.Format))},
		"$.videos.optimize": {yaml.HeadComment(fmt.Sprintf(" optimize videos (compresses and re-encodes; default: %v)", def.Videos.Optimize))},

		"$.gifs.enabled":       {yaml.HeadComment(fmt.Sprintf(" allow gif uploads (requires ffmpeg/ffprobe; default: %v)", def.GIFs.Enabled))},
		"$.gifs.optimize":      {yaml.HeadComment(fmt.Sprintf(" optimize gifs (compresses and re-encodes; including video.format = gif; requires gifsicle; default: %v)", def.GIFs.Optimize))},
		"$.gifs.effort":        {yaml.HeadComment(fmt.Sprintf(" gifsicle optimization effort (1 = fast/big, 2 = medium, 3 = slow/small; default: %v)", def.GIFs.Effort))},
		"$.gifs.quality":       {yaml.HeadComment(fmt.Sprintf(" visual quality (1 - 100; 100=lossless; lower values enable gifsicle --lossy and increase compression; default: %v)", def.GIFs.Quality))},
		"$.gifs.max_colors":    {yaml.HeadComment(fmt.Sprintf(" maximum colors in GIF palette (2-256; smaller = smaller files; default: %v)", def.GIFs.MaxColors))},
		"$.gifs.max_framerate": {yaml.HeadComment(fmt.Sprintf(" maximum fps (1 - 30; default: %v)", def.GIFs.MaxFramerate))},
		"$.gifs.max_width":     {yaml.HeadComment(fmt.Sprintf(" maximum width/height (1 - 1024; default: %v)", def.GIFs.MaxWidth))},

		"$.ai.openrouter_token": {yaml.HeadComment(fmt.Sprintf(" openrouter token for image tagging (if empty, disables image tagging; default: %v)", def.AI.OpenRouterToken))},
		"$.ai.tagging_model":    {yaml.HeadComment(fmt.Sprintf(" model used for image tagging (requires vision and structured output support; default: %v)", def.AI.TaggingModel))},
		"$.ai.embedding_model":  {yaml.HeadComment(fmt.Sprintf(" model used for embedding (requires embedding support; default: %v)", def.AI.EmbeddingModel))},
		"$.ai.min_similarity":   {yaml.HeadComment(fmt.Sprintf(" minimum similarity percentage (0-100) for results to be included (default: %v)", def.AI.MinSimilarity))},
		"$.ai.re_tag_empty":     {yaml.HeadComment(fmt.Sprintf(" if echos without tags should be re-tagged (default: %v)", def.AI.ReTagEmpty))},
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
