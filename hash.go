package main

import (
	"context"
	"math/rand"
	"regexp"
)

var hashRgx = regexp.MustCompile(`^[0-9A-Za-z]{10}$`)

func (d *EchoDatabase) Hash(ctx context.Context) (string, error) {
	for {
		hash := generateHash()

		echo, err := d.Find(ctx, hash)
		if err != nil {
			return "", err
		}

		if echo == nil {
			return hash, nil
		}
	}
}

func generateHash() string {
	var hash string

	for range 10 {
		char := rand.Intn(36) + 48

		if char > 57 {
			char += 7
		}

		hash += string(rune(char))
	}

	return hash
}

func validateHash(hash string) bool {
	if len(hash) != 10 {
		return false
	}

	return hashRgx.MatchString(hash)
}
