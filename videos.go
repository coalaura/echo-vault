package main

import (
	"context"
)

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
	// Base settings: single video/audio stream, strip metadata/subs, yuv420p for device compatibility,
	// faststart relocates moov atom so browsers can begin playback before full download
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
		// Smaller files + streaming-friendly: slow preset for better compression, CRF 24 balances size/quality,
		// high profile improves efficiency, regular keyframes enable instant seeking, maxrate/bufsize prevent
		// bitrate spikes that cause buffering, tune=film preserves detail in typical video content
		return append(common,
			"-c:v", "libx264",
			"-preset", "slow",
			"-crf", "24",
			"-profile:v", "high",
			"-level", "4.1",
			"-tune", "film",
			"-g", "48",
			"-keyint_min", "24",
			"-maxrate", "5M",
			"-bufsize", "10M",
			"-c:a", "aac",
			"-b:a", "128k",
			"-ar", "48000",
		)
	}

	// Fast encode prioritizing quality: veryfast preset for speed, lower CRF retains more detail,
	// still includes keyframes for seeking but skips rate limiting for maximum quality
	return append(common,
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-crf", "20",
		"-profile:v", "high",
		"-level", "4.1",
		"-g", "48",
		"-keyint_min", "24",
		"-c:a", "aac",
		"-b:a", "160k",
		"-ar", "48000",
	)
}

func getWebMArgs() []string {
	// Base settings: single video/audio stream, strip metadata/subs, yuv420p for compatibility
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
		// VP9 quality mode (b:v=0 enables pure CRF), cpu-used 2 balances speed/quality, row-mt enables
		// multithreading, tile-columns + frame-parallel improve browser decode performance, keyframes for seeking
		return append(common,
			"-c:v", "libvpx-vp9",
			"-crf", "32",
			"-b:v", "0",
			"-row-mt", "1",
			"-cpu-used", "2",
			"-g", "48",
			"-tile-columns", "2",
			"-frame-parallel", "1",
			"-c:a", "libopus",
			"-b:a", "96k",
			"-ar", "48000",
		)
	}

	// Fast VP9 encode: cpu-used 4 significantly speeds up encoding, lower CRF preserves more quality
	return append(common,
		"-c:v", "libvpx-vp9",
		"-crf", "28",
		"-b:v", "0",
		"-row-mt", "1",
		"-cpu-used", "4",
		"-g", "48",
		"-tile-columns", "2",
		"-c:a", "libopus",
		"-b:a", "128k",
		"-ar", "48000",
	)
}

func getMOVArgs() []string {
	// Base settings: same as MP4 (both are QuickTime-family containers), faststart for streaming
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
		// Mirrors MP4 optimized settings: slow preset, balanced CRF, streaming-safe rate control
		return append(common,
			"-c:v", "libx264",
			"-preset", "slow",
			"-crf", "24",
			"-profile:v", "high",
			"-level", "4.1",
			"-tune", "film",
			"-g", "48",
			"-keyint_min", "24",
			"-maxrate", "5M",
			"-bufsize", "10M",
			"-c:a", "aac",
			"-b:a", "128k",
			"-ar", "48000",
		)
	}

	// Mirrors MP4 fast settings: quick encode, quality-focused
	return append(common,
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-crf", "20",
		"-profile:v", "high",
		"-level", "4.1",
		"-g", "48",
		"-keyint_min", "24",
		"-c:a", "aac",
		"-b:a", "160k",
		"-ar", "48000",
	)
}

func getMKVArgs() []string {
	// Base settings: MKV is flexible container, no faststart needed (different seeking mechanism)
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
		// Same H.264 settings as MP4/MOV for consistency, keyframes still important for seeking
		return append(common,
			"-c:v", "libx264",
			"-preset", "slow",
			"-crf", "24",
			"-profile:v", "high",
			"-level", "4.1",
			"-tune", "film",
			"-g", "48",
			"-keyint_min", "24",
			"-c:a", "aac",
			"-b:a", "128k",
			"-ar", "48000",
		)
	}

	// Fast encode, quality-focused
	return append(common,
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-crf", "20",
		"-profile:v", "high",
		"-level", "4.1",
		"-g", "48",
		"-keyint_min", "24",
		"-c:a", "aac",
		"-b:a", "160k",
		"-ar", "48000",
	)
}
