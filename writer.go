package main

import "os"

type CountWriter struct {
	*os.File
	N int
}

func NewCountWriter(f *os.File) *CountWriter {
	return &CountWriter{
		File: f,
	}
}

func (c *CountWriter) Write(p []byte) (int, error) {
	n, err := c.File.Write(p)

	c.N += n

	return n, err
}
