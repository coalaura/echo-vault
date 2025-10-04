package main

import (
	"math/rand"
	"regexp"
)

var hashRgx = regexp.MustCompile(`^[0-9A-Z]{10}$`)

func (d *EchoDatabase) Hash() (string, error) {
	for {
		hash := generateHash()

		echo, err := d.Find(hash)
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

	for i := 0; i < 10; i++ {
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
