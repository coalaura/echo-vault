//go:build ai

package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/philippgille/chromem-go"
	ort "github.com/yalue/onnxruntime_go"
)

type VectorStore struct {
	db         *chromem.DB
	collection *chromem.Collection
	clip       *CLIP
}

func LoadVectorStore() (*VectorStore, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	onnxRoot := filepath.Join(cwd, "onnx")

	if _, err := os.Stat(onnxRoot); err != nil {
		log.Warnf("AI build: missing %s, semantic search disabled\n", onnxRoot)

		return nil, nil
	}

	libDir, ortLib, modelsDir, err := resolveONNX(onnxRoot)
	if err != nil {
		return nil, err
	}

	prependDynlibSearchPath(libDir)

	ort.SetSharedLibraryPath(ortLib)

	err = ort.InitializeEnvironment()
	if err != nil {
		return nil, err
	}

	clip, err := NewCLIP(modelsDir)
	if err != nil {
		return nil, err
	}

	db, err := chromem.NewPersistentDB(ClipDirectory, false)
	if err != nil {
		return nil, err
	}

	embedder := chromem.EmbeddingFunc(func(ctx context.Context, text string) ([]float32, error) {
		return clip.EmbedText(text)
	})

	coll, err := db.GetOrCreateCollection("clip", nil, embedder)
	if err != nil {
		return nil, err
	}

	return &VectorStore{
		db:         db,
		collection: coll,
		clip:       clip,
	}, nil
}

func (s *VectorStore) Close() {
	if s == nil {
		return
	}

	if s.clip != nil {
		s.clip.Close()
	}

	ort.DestroyEnvironment()
}

func (s *VectorStore) IndexImage(ctx context.Context, hash, imagePath string) error {
	if s == nil {
		return nil
	}

	emb, err := s.clip.EmbedImageFile(imagePath)
	if err != nil {
		return err
	}

	return s.collection.AddDocument(ctx, chromem.Document{
		ID:        hash,
		Embedding: emb,
		Content:   imagePath,
	})
}

func (s *VectorStore) Query(ctx context.Context, query string, max int) ([]VectorResult, error) {
	if s == nil {
		return nil, nil
	}

	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	amount := min(max, s.collection.Count())
	if amount == 0 {
		return nil, nil
	}

	res, err := s.collection.Query(ctx, query, amount, nil, nil)
	if err != nil {
		return nil, err
	}

	out := make([]VectorResult, 0, len(res))

	for _, r := range res {
		out = append(out, VectorResult{
			Hash:       r.ID,
			Similarity: r.Similarity,
		})
	}

	return out, nil
}

func (s *VectorStore) Has(hash string) bool {
	if s == nil {
		return false
	}

	_, err := s.collection.GetByID(context.Background(), hash)
	return err == nil
}

func (s *VectorStore) Delete(hash string) error {
	if s == nil {
		return nil
	}

	return s.collection.Delete(context.Background(), nil, nil, hash)
}

func resolveONNX(onnxRoot string) (libDir, ortLib, modelsDir string, err error) {
	modelsDir = filepath.Join(onnxRoot, "models")

	switch runtime.GOOS {
	case "windows":
		libDir = filepath.Join(onnxRoot, "lib", "windows-amd64")
		ortLib = filepath.Join(libDir, "onnxruntime.dll")
	case "linux":
		libDir = filepath.Join(onnxRoot, "lib", "linux-amd64")
		ortLib = filepath.Join(libDir, "libonnxruntime.so.1.24.1")
	default:
		return "", "", "", err
	}

	return libDir, ortLib, modelsDir, nil
}

func prependDynlibSearchPath(dir string) {
	switch runtime.GOOS {
	case "windows":
		prependEnv("PATH", dir, ";")
	case "linux":
		prependEnv("LD_LIBRARY_PATH", dir, ":")
	}
}

func prependEnv(name, dir, sep string) {
	cur := os.Getenv(name)
	if cur == "" {
		os.Setenv(name, dir)

		return
	}

	if strings.Contains(cur, dir) {
		return
	}

	os.Setenv(name, dir+sep+cur)
}
