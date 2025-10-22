package main

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func saveGIFAsGIF(ctx context.Context, input, path string) (int64, error) {
	fps, err := probeFPS(ctx, input)
	if err != nil {
		log.Warnf("failed to probe fps: %v\n", err)
	}

	// palettegen/paletteuse at target fps for quality while resizing frame rate
	args := []string{
		"-map", "0:v:0",
		"-an", "-sn",
		"-filter_complex", buildGifFilter(fps > 0 && fps > float64(config.GIFs.MaxFramerate)),
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

func saveVideoAsGIF(ctx context.Context, input, path string) (int64, error) {
	fps, err := probeFPS(ctx, input)
	if err != nil {
		log.Warnf("failed to probe fps: %v\n", err)
	}

	args := []string{
		"-map", "0:v:0",
		"-an", "-sn",
		"-filter_complex", buildGifFilter(fps > 0 && fps > float64(config.GIFs.MaxFramerate)),
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
	args := gifsicleArgs(path)

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

func buildGifFilter(resampleFPS bool) string {
	var chain strings.Builder

	if resampleFPS {
		chain.WriteString(fmt.Sprintf("fps=%d", config.GIFs.MaxFramerate))
	}

	if chain.Len() > 0 {
		chain.WriteByte(',')
	}

	chain.WriteString(ffmpegScaleExpression(config.GIFs.MaxWidth))

	return fmt.Sprintf(
		"[0:v]%s,split[s0][s1];[s0]palettegen=stats_mode=full:max_colors=%d[p];[s1][p]paletteuse=dither=bayer:bayer_scale=5:diff_mode=rectangle",
		chain.String(),
		config.GIFs.MaxColors,
	)
}

func gifsicleArgs(path string) []string {
	args := []string{
		fmt.Sprintf("-O%d", config.GIFs.Effort),
		"--no-comments", "--no-names", "--no-extensions",
		"--colors", strconv.Itoa(config.GIFs.MaxColors),
	}

	if config.GIFs.Quality < 100 {
		lossy := int(math.Round(float64(100-config.GIFs.Quality) * 1.4))

		if lossy > 0 {
			args = append(args, fmt.Sprintf("--lossy=%d", lossy))
		}
	}

	return append(
		args,
		"-o", path,
		path,
	)
}
