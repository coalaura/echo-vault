package main

import (
	"context"
	"sync"

	"github.com/philippgille/chromem-go"
)

type VectorStore struct {
	mx         sync.RWMutex
	db         *chromem.DB
	collection *chromem.Collection
}

type VectorResult struct {
	Hash       string
	Similarity float32
}

func LoadVectorStore() (*VectorStore, error) {
	if config.AI.OpenRouterToken == "" {
		return nil, nil
	}

	db, err := chromem.NewPersistentDB("tags", false)
	if err != nil {
		return nil, err
	}

	embedder := chromem.NewEmbeddingFuncOpenAICompat("https://openrouter.ai/api/v1", config.AI.OpenRouterToken, config.AI.EmbeddingModel, nil)

	collection, err := db.GetOrCreateCollection("tags", nil, embedder)
	if err != nil {
		return nil, err
	}

	return &VectorStore{
		db:         db,
		collection: collection,
	}, nil
}

func (s *VectorStore) Query(ctx context.Context, query string, max int) ([]VectorResult, error) {
	amount := min(max, s.collection.Count())

	if amount == 0 {
		return nil, nil
	}

	s.mx.RLock()
	defer s.mx.RUnlock()

	results, err := s.collection.Query(ctx, query, amount, nil, nil)
	if err != nil {
		return nil, err
	}

	threshold := float32(config.AI.MinSimilarity) / 100.0

	hashes := make([]VectorResult, 0, len(results))

	for _, result := range results {
		if config.AI.MinSimilarity > 0 && result.Similarity < threshold {
			continue
		}

		hashes = append(hashes, VectorResult{
			Hash:       result.ID,
			Similarity: result.Similarity,
		})
	}

	return hashes, nil
}

func (s *VectorStore) Store(hash string, entry EchoTag) error {
	s.mx.Lock()
	defer s.mx.Unlock()

	document := chromem.Document{
		ID:      hash,
		Content: entry.EmbeddingString(),
	}

	return s.collection.AddDocument(context.Background(), document)
}
