//go:build ai

package main

import (
	"fmt"
	"path/filepath"
	"sync"

	ort "github.com/yalue/onnxruntime_go"
)

const (
	clipDim       = 512
	clipMaxTokens = 77
	clipSOT       = 49406
	clipEOT       = 49407
)

type CLIP struct {
	tok *CLIPTokenizer

	visSess  *ort.AdvancedSession
	textSess *ort.AdvancedSession

	visIn   *ort.Tensor[float32] // [1,3,224,224]
	visOut  *ort.Tensor[float32] // [1,512]
	textIDs *ort.Tensor[int64]   // [1,77]
	textAtt *ort.Tensor[int64]   // [1,77]
	textOut *ort.Tensor[float32] // [1,512]

	mu sync.Mutex
}

func NewCLIPTokenizerFromDir(dir string) (*CLIPTokenizer, error) {
	vocabPath := filepath.Join(dir, "vocab.json")
	mergesPath := filepath.Join(dir, "merges.txt")

	// Prefer vocab.json + merges.txt if present
	if fileExists(vocabPath) && fileExists(mergesPath) {
		return NewCLIPTokenizerFromVocabMerges(vocabPath, mergesPath)
	}

	tokenizerJSONPath := filepath.Join(dir, "tokenizer.json")

	// Fallback to tokenizer.json
	if fileExists(tokenizerJSONPath) {
		return NewCLIPTokenizerFromTokenizerJSON(tokenizerJSONPath)
	}

	return nil, fmt.Errorf("clip tokenizer not found in %s (expected vocab.json+merges.txt or tokenizer.json)", dir)
}

func NewCLIP(modelsDir string) (*CLIP, error) {
	tokDir := filepath.Join(modelsDir, "clip_tokenizer")

	tok, err := NewCLIPTokenizerFromDir(tokDir)
	if err != nil {
		return nil, err
	}

	visualModel := filepath.Join(modelsDir, "clip_visual.onnx")
	textModel := filepath.Join(modelsDir, "clip_text.onnx")

	visIn, err := ort.NewTensor(ort.NewShape(1, 3, 224, 224), make([]float32, 1*3*224*224))
	if err != nil {
		return nil, fmt.Errorf("create visIn: %w", err)
	}

	visOut, err := ort.NewEmptyTensor[float32](ort.NewShape(1, clipDim))
	if err != nil {
		visIn.Destroy()

		return nil, fmt.Errorf("create visOut: %w", err)
	}

	textIDs, err := ort.NewTensor(ort.NewShape(1, clipMaxTokens), make([]int64, clipMaxTokens))
	if err != nil {
		visIn.Destroy()
		visOut.Destroy()

		return nil, fmt.Errorf("create textIDs: %w", err)
	}

	textAtt, err := ort.NewTensor(ort.NewShape(1, clipMaxTokens), make([]int64, clipMaxTokens))
	if err != nil {
		visIn.Destroy()
		visOut.Destroy()
		textIDs.Destroy()

		return nil, fmt.Errorf("create textAtt: %w", err)
	}

	textOut, err := ort.NewEmptyTensor[float32](ort.NewShape(1, clipDim))
	if err != nil {
		visIn.Destroy()
		visOut.Destroy()
		textIDs.Destroy()
		textAtt.Destroy()

		return nil, fmt.Errorf("create textOut: %w", err)
	}

	visSess, err := ort.NewAdvancedSession(
		visualModel,
		[]string{"pixel_values"},
		[]string{"image_embeds"},
		[]ort.ArbitraryTensor{visIn},
		[]ort.ArbitraryTensor{visOut},
		nil,
	)
	if err != nil {
		visIn.Destroy()
		visOut.Destroy()
		textIDs.Destroy()
		textAtt.Destroy()
		textOut.Destroy()

		return nil, fmt.Errorf("create vis session: %w", err)
	}

	textSess, err := ort.NewAdvancedSession(
		textModel,
		[]string{"input_ids", "attention_mask"},
		[]string{"text_embeds"},
		[]ort.ArbitraryTensor{textIDs, textAtt},
		[]ort.ArbitraryTensor{textOut},
		nil,
	)
	if err != nil {
		visSess.Destroy()
		visIn.Destroy()
		visOut.Destroy()
		textIDs.Destroy()
		textAtt.Destroy()
		textOut.Destroy()

		return nil, fmt.Errorf("create text session: %w", err)
	}

	return &CLIP{
		tok:      tok,
		visSess:  visSess,
		textSess: textSess,
		visIn:    visIn,
		visOut:   visOut,
		textIDs:  textIDs,
		textAtt:  textAtt,
		textOut:  textOut,
	}, nil
}

func (c *CLIP) Close() {
	if c == nil {
		return
	}

	c.visSess.Destroy()
	c.textSess.Destroy()
	c.visIn.Destroy()
	c.visOut.Destroy()
	c.textIDs.Destroy()
	c.textAtt.Destroy()
	c.textOut.Destroy()
}

func (c *CLIP) EmbedText(text string) ([]float32, error) {
	ids, att := c.tok.Encode(text)

	c.mu.Lock()
	defer c.mu.Unlock()

	copy(c.textIDs.GetData(), ids)
	copy(c.textAtt.GetData(), att)

	err := c.textSess.Run()
	if err != nil {
		return nil, err
	}

	out := c.textOut.GetData()
	emb := make([]float32, clipDim)
	copy(emb, out[:clipDim])

	return emb, nil
}

func (c *CLIP) EmbedImageFile(path string) ([]float32, error) {
	imgTensor, err := PreprocessImageFileToNCHW(path)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	copy(c.visIn.GetData(), imgTensor)

	err = c.visSess.Run()
	if err != nil {
		return nil, err
	}

	out := c.visOut.GetData()
	emb := make([]float32, clipDim)
	copy(emb, out[:clipDim])

	return emb, nil
}
