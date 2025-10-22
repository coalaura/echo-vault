package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
)

func saveGIFAsGIF(ctx context.Context, input, path string) (int64, error) {
	fps, err := probeFPS(ctx, input)
	if err != nil {
		log.Warnf("failed to probe fps: %v\n", err)
	}

	var size int64

	if fps > 0 && fps > float64(config.GIFs.Framerate) {
		// palettegen/paletteuse at target fps for quality while resizing frame rate
		args := []string{
			"-an", "-sn",
			"-filter_complex",
			fmt.Sprintf(
				"[0:v]fps=%d,split[s0][s1];[s0]palettegen=stats_mode=full:max_colors=256[p];[s1][p]paletteuse=dither=bayer:bayer_scale=5:diff_mode=rectangle",
				config.GIFs.Framerate,
			),
			"-f", "gif",
		}

		size, err = runFFMpeg(ctx, input, path, args)
		if err != nil {
			return 0, err
		}
	} else {
		err = os.Rename(input, path)
		if err != nil {
			return 0, err
		}

		stat, err := os.Stat(path)
		if err != nil {
			return 0, err
		}

		size = stat.Size()
	}

	if config.GIFs.Optimize {
		size, err = optimizeGIF(ctx, path)
		if err != nil {
			return 0, err
		}
	}

	return size, nil
}

func saveVideoAsGIF(ctx context.Context, input, path string) (int64, error) {
	fps, err := probeFPS(ctx, input)
	if err != nil {
		log.Warnf("failed to probe fps: %v\n", err)
	}

	var filter string

	if fps > 0 && fps > float64(config.GIFs.Framerate) {
		filter = fmt.Sprintf(
			"[0:v]fps=%d,split[s0][s1];[s0]palettegen=stats_mode=full:max_colors=256[p];[s1][p]paletteuse=dither=bayer:bayer_scale=5:diff_mode=rectangle",
			config.GIFs.Framerate,
		)
	} else {
		filter = "[0:v]split[s0][s1];[s0]palettegen=stats_mode=full:max_colors=256[p];[s1][p]paletteuse=dither=bayer:bayer_scale=5:diff_mode=rectangle"
	}

	args := []string{
		"-an", "-sn",
		"-filter_complex", filter,
		"-f", "gif",
	}

	size, err := runFFMpeg(ctx, input, path, args)
	if err != nil {
		return 0, err
	}

	if config.GIFs.Optimize {
		size, err = optimizeGIF(ctx, path)
		if err != nil {
			return 0, err
		}
	}

	return size, nil
}

func optimizeGIF(ctx context.Context, path string) (int64, error) {
	args := []string{
		fmt.Sprintf("-O%d", config.GIFs.Effort),
		"--no-comments", "--no-names", "--no-extensions",
		"-o", path,
		path,
	}

	cmd := exec.CommandContext(ctx, config.gifsicle, args...)

	var stderr bytes.Buffer

	cmd.Stderr = &stderr

	err := cmd.Start()
	if err != nil {
		return 0, fmt.Errorf("start gifsicle: %w", err)
	}

	err = cmd.Wait()
	if err != nil {
		return 0, fmt.Errorf("gifsicle: %w: %s", err, stderr.String())
	}

	stat, err := os.Stat(path)
	if err != nil {
		return 0, err
	}

	return stat.Size(), nil
}
