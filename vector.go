package main

import "errors"

const ClipDirectory = "clip"

type VectorResult struct {
	Hash       string
	Similarity float32
}

var ErrAIDisabled = errors.New("ai build is disabled")
