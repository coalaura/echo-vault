package main

import "os"

type CountWriter struct {
	*os.File
	N int64
}

func OpenCountWriter(path string) (*CountWriter, error) {
	file, err := OpenFileForWriting(path)
	if err != nil {
		return nil, err
	}

	return &CountWriter{
		File: file,
	}, nil
}

func (c *CountWriter) Write(p []byte) (int, error) {
	n, err := c.File.Write(p)

	c.N += int64(n)

	return n, err
}

func (c *CountWriter) Close() error {
	return c.File.Close()
}
