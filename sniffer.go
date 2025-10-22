package main

import (
	"bytes"
	"encoding/binary"
)

const MaxSniffBytes = 16 * 1024

func sniffType(buf []byte) string {
	if isWEBP(buf) {
		return "webp"
	}

	if isPNG(buf) {
		return "png"
	}

	if isJPEG(buf) {
		return "jpeg"
	}

	if isGIF(buf) {
		return "gif"
	}

	if brand, ok := isoBMFFBrand(buf); ok {
		switch brand {
		case "qt  ":
			return "mov"
		case "M4V ", "M4VH", "M4VP":
			return "m4v"
		case "heic", "heif", "mif1", "msf1":
			return "" // reject HEIF family
		}

		// probably mp4
		return "mp4"
	}

	if isEBML(buf) {
		switch ebmlDocType(buf) {
		case "webm":
			return "webm"
		case "matroska":
			return "mkv"
		}

		// probably mkv
		return "mkv"
	}

	return ""
}

func isPNG(b []byte) bool {
	return len(b) >= 8 && bytes.Equal(b[:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A})
}

func isJPEG(b []byte) bool {
	return len(b) >= 3 && b[0] == 0xFF && b[1] == 0xD8 && b[2] == 0xFF
}

func isGIF(b []byte) bool {
	return len(b) >= 6 && (bytes.Equal(b[:6], []byte("GIF87a")) || bytes.Equal(b[:6], []byte("GIF89a")))
}

func isWEBP(b []byte) bool {
	return len(b) >= 12 && bytes.Equal(b[:4], []byte("RIFF")) && bytes.Equal(b[8:12], []byte("WEBP"))
}

// isoBMFFBrand detects 'ftyp' box and returns major_brand (4 chars) if present.
// It scans the first ~16 KiB to allow for leading boxes/prefixes.
func isoBMFFBrand(b []byte) (string, bool) {
	n := min(len(b), MaxSniffBytes)

	var i int

	for i+8 <= n {
		size := binary.BigEndian.Uint32(b[i : i+4])

		typ := string(b[i+4 : i+8])

		var boxSize int

		switch size {
		case 0:
			// box extends to EOF; we only scan within n
			boxSize = n - i
		case 1:
			// largesize (64-bit) at i+8..i+16
			if i+16 > n {
				return "", false
			}

			largesize := binary.BigEndian.Uint64(b[i+8 : i+16])

			if largesize < 16 {
				return "", false
			}

			boxSize = int(largesize)
		default:
			if size < 8 {
				return "", false
			}

			boxSize = int(size)
		}

		if typ == "ftyp" {
			// ftyp payload starts immediately after header
			// For largesize, major_brand is at i+16; for normal, at i+8
			off := 8

			if size == 1 {
				off = 16
			}

			if i+off+4 <= len(b) {
				return string(b[i+off : i+off+4]), true
			}

			return "", false
		}

		if i+boxSize > n || boxSize <= 0 {
			break
		}

		i += boxSize
	}

	return "", false
}

// EBML (Matroska/WebM) starts with 1A 45 DF A3
func isEBML(b []byte) bool {
	return len(b) >= 4 && b[0] == 0x1A && b[1] == 0x45 && b[2] == 0xDF && b[3] == 0xA3
}

// We avoid full EBML parsing; searching for ASCII DocType within the header window is enough.
func ebmlDocType(b []byte) string {
	win := b

	if len(win) > 4096 {
		win = win[:4096]
	}

	if bytes.Contains(win, []byte("webm")) {
		return "webm"
	}

	if bytes.Contains(win, []byte("matroska")) {
		return "matroska"
	}

	return ""
}
