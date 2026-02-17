//go:build !ai

package main

import "context"

type VectorStore struct{}

func LoadVectorStore() (*VectorStore, error) {
	return nil, nil
}

func (s *VectorStore) Query(ctx context.Context, query string, max int) ([]VectorResult, error) {
	return nil, nil
}

func (s *VectorStore) IndexImage(ctx context.Context, hash, imagePath string) error {
	return nil
}

func (s *VectorStore) Has(hash string) bool {
	return false
}

func (s *VectorStore) Delete(hash string) error {
	return nil
}

func (s *VectorStore) Close() {
}
