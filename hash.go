package main

import "math/rand"

func (d *EchoDatabase) Hash() (string, error) {
	for {
		hash := generateHash(10)

		echo, err := d.Find(hash)
		if err != nil {
			return "", err
		}

		if echo == nil {
			return hash, nil
		}
	}
}

func generateHash(len int) string {
	var hash string

	for i := 0; i < len; i++ {
		char := rand.Intn(36) + 48

		if char > 57 {
			char += 7
		}

		hash += string(rune(char))
	}

	return hash
}
