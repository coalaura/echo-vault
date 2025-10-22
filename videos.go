package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
)

func runFFMpeg(ctx context.Context, in, out string, args []string) (int64, error) {
	args = append([]string{"-hide_banner", "-loglevel", "error", "-y", "-i", in}, args...)
	args = append(args, out)

	cmd := exec.CommandContext(ctx, config.ffmpeg, args...)

	var stderr bytes.Buffer

	cmd.Stderr = &stderr

	err := cmd.Start()
	if err != nil {
		return 0, fmt.Errorf("start ffmpeg: %w", err)
	}

	err = cmd.Wait()
	if err != nil {
		return 0, fmt.Errorf("ffmpeg: %w: %s", err, stderr.String())
	}

	stat, err := os.Stat(out)
	if err != nil {
		return 0, err
	}

	return stat.Size(), nil
}

func saveVideoAsMP4(ctx context.Context, input, path string) (int64, error) {
	return runFFMpeg(ctx, input, path, getMP4Args())
}

func saveVideoAsWebM(ctx context.Context, input, path string) (int64, error) {
	return runFFMpeg(ctx, input, path, getWebMArgs())
}

func saveVideoAsMOV(ctx context.Context, input, path string) (int64, error) {
	return runFFMpeg(ctx, input, path, getMOVArgs())
}

func saveVideoAsM4V(ctx context.Context, input, path string) (int64, error) {
	// Compatible container
	return saveVideoAsMP4(ctx, input, path)
}

func saveVideoAsMKV(ctx context.Context, input, path string) (int64, error) {
	return runFFMpeg(ctx, input, path, getMKVArgs())
}

func getMP4Args() []string {
	// map first video + optional first audio; drop subs/data; remove metadata/chapters; ensure 4:2:0; enable progressive download; set container
	common := []string{
		"-map", "0:v:0",
		"-map", "0:a:0?",
		"-sn", "-dn",
		"-map_metadata", "-1",
		"-map_chapters", "-1",
		"-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
		"-f", "mp4",
	}

	if config.Videos.Optimize {
		// optimize=true: slower preset + typical CRF for smaller files; AAC at 128k
		return append(common,
			"-c:v", "libx264",
			"-preset", "slow",
			"-crf", "23",
			"-c:a", "aac",
			"-b:a", "128k",
		)
	}

	// optimize=false: faster preset + lower CRF to preserve quality; slightly higher audio bitrate
	return append(common,
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-crf", "20",
		"-c:a", "aac",
		"-b:a", "160k",
	)
}

func getWebMArgs() []string {
	// map first video + optional first audio; drop subs/data; remove metadata/chapters; ensure 4:2:0; set container
	common := []string{
		"-map", "0:v:0",
		"-map", "0:a:0?",
		"-sn", "-dn",
		"-map_metadata", "-1",
		"-map_chapters", "-1",
		"-pix_fmt", "yuv420p",
		"-f", "webm",
	}

	if config.Videos.Optimize {
		// optimize=true: VP9 CRF mode, best quality per bit (slower); Opus 96k
		return append(common,
			"-c:v", "libvpx-vp9",
			"-crf", "33",
			"-b:v", "0",
			"-row-mt", "1",
			"-cpu-used", "0",
			"-c:a", "libopus",
			"-b:a", "96k",
		)
	}

	// optimize=false: faster VP9 encode (cpu-used 4) + slightly higher quality target; Opus 128k
	return append(common,
		"-c:v", "libvpx-vp9",
		"-crf", "28", "-b:v", "0",
		"-row-mt", "1",
		"-cpu-used", "4",
		"-c:a", "libopus",
		"-b:a", "128k",
	)
}

func getMOVArgs() []string {
	// map first video + optional first audio; drop subs/data; remove metadata/chapters; ensure 4:2:0; enable progressive download; set container
	common := []string{
		"-map", "0:v:0",
		"-map", "0:a:0?",
		"-sn", "-dn",
		"-map_metadata", "-1",
		"-map_chapters", "-1",
		"-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
		"-f", "mov",
	}

	if config.Videos.Optimize {
		// optimize=true: slower preset + typical CRF for smaller files; AAC at 128k
		return append(common,
			"-c:v", "libx264",
			"-preset", "slow",
			"-crf", "23",
			"-c:a", "aac",
			"-b:a", "128k",
		)
	}

	// optimize=false: faster preset + lower CRF to preserve quality; slightly higher audio bitrate
	return append(common,
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-crf", "20",
		"-c:a", "aac",
		"-b:a", "160k",
	)
}

func getMKVArgs() []string {
	// map first video + optional first audio; drop subs/data; remove metadata/chapters; ensure 4:2:0; set container
	common := []string{
		"-map", "0:v:0",
		"-map", "0:a:0?",
		"-sn", "-dn",
		"-map_metadata", "-1",
		"-map_chapters", "-1",
		"-pix_fmt", "yuv420p",
		"-f", "matroska",
	}

	if config.Videos.Optimize {
		// optimize=true: slower preset + typical CRF for smaller files; AAC at 128k
		return append(common,
			"-c:v", "libx264",
			"-preset", "slow",
			"-crf", "23",
			"-c:a", "aac",
			"-b:a", "128k",
		)
	}

	// optimize=false: faster preset + lower CRF to preserve quality; slightly higher audio bitrate
	return append(common,
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-crf", "20",
		"-c:a", "aac",
		"-b:a", "160k",
	)
}
