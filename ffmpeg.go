package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func ffmpegScaleExpression(maxWidth int) string {
	return fmt.Sprintf("scale='if(gt(iw,ih),%d,-2)':'if(gt(iw,ih),-2,%d)':flags=lanczos", maxWidth, maxWidth)
}

func parseFPSRational(line string) (float64, bool) {
	if index := strings.Index(line, "/"); index != -1 {
		num, err1 := strconv.ParseFloat(line[:index], 64)
		den, err2 := strconv.ParseFloat(line[index+1:], 64)

		if err1 == nil && err2 == nil && den != 0 {
			return num / den, true
		}
	} else {
		fps, err := strconv.ParseFloat(line, 64)
		if err == nil {
			return fps, true
		}
	}

	return 0, false
}

func probeFPS(ctx context.Context, path string) (float64, error) {
	args := []string{
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=avg_frame_rate,r_frame_rate",
		"-of", "default=noprint_wrappers=1:nokey=1",
		path,
	}

	cmd := exec.CommandContext(ctx, config.ffprobe, args...)

	var stdout bytes.Buffer

	cmd.Stdout = &stdout

	var stderr bytes.Buffer

	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return 0, err
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return 0, errors.New("empty ffprobe output")
	}

	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)

		if line == "" || line == "0/0" {
			continue
		}

		fps, ok := parseFPSRational(line)
		if ok && fps > 0 {
			return fps, nil
		}
	}

	return 0, errors.New("failed to determine fps")
}

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
