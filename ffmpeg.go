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

	cmd.Stdout = os.Stdout
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

func remuxVideo(ctx context.Context, input, path, ext string) (int64, error) {
	args := []string{
		"-map", "0:v:0",
		"-map", "0:a:0?",
		"-c", "copy",
		"-map_metadata", "-1", // Strip metadata
	}

	// Only apply faststart to mp4/mov containers
	if ext == "mp4" || ext == "mov" {
		args = append(args, "-movflags", "+faststart")
	}

	args = append(args, "-f", ext)

	return runFFMpeg(ctx, input, path, args)
}
