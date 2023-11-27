package main

import "math/rand"

func findFreeHash() (string, error) {
	var count int

	for {
		hash := generateHash(10)

		err := database.QueryRow("SELECT COUNT(*) FROM echos WHERE hash = ?", hash).Scan(&count)
		if err != nil {
			return "", err
		}

		if count == 0 {
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
