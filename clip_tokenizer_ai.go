//go:build ai

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"
)

type CLIPTokenizer struct {
	encoder     map[string]int
	bpeRanks    map[[2]string]int
	cache       map[string]string
	pat         *regexp.Regexp
	byteEncoder map[byte]rune
}

type TokenizerStruct struct {
	Model TokenizerModel `json:"model"`
}

type TokenizerModel struct {
	Type   string         `json:"type"`
	Vocab  map[string]int `json:"vocab"`
	Merges any            `json:"merges"`
}

func NewCLIPTokenizerFromVocabMerges(vocabPath, mergesPath string) (*CLIPTokenizer, error) {
	vb, err := os.ReadFile(vocabPath)
	if err != nil {
		return nil, fmt.Errorf("read vocab.json: %w", err)
	}

	var enc map[string]int

	err = json.Unmarshal(vb, &enc)
	if err != nil {
		return nil, fmt.Errorf("parse vocab.json: %w", err)
	}

	ranks, err := loadMergesFile(mergesPath)
	if err != nil {
		return nil, err
	}

	return newCLIPTokenizer(enc, ranks), nil
}

func NewCLIPTokenizerFromTokenizerJSON(path string) (*CLIPTokenizer, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var tok TokenizerStruct

	err = json.Unmarshal(b, &tok)
	if err != nil {
		return nil, err
	}

	if strings.ToLower(tok.Model.Type) != "bpe" {
		return nil, fmt.Errorf("tokenizer.json model.type=%q (expected BPE)", tok.Model.Type)
	}

	if len(tok.Model.Vocab) == 0 {
		return nil, fmt.Errorf("tokenizer.json has empty vocab")
	}

	merges, err := normalizeMerges(tok.Model.Merges)
	if err != nil {
		return nil, err
	}

	ranks := make(map[[2]string]int, len(merges))

	for i, m := range merges {
		parts := strings.SplitN(strings.TrimSpace(m), " ", 2)
		if len(parts) != 2 {
			continue
		}

		ranks[[2]string{parts[0], parts[1]}] = i
	}

	return newCLIPTokenizer(tok.Model.Vocab, ranks), nil
}

func newCLIPTokenizer(enc map[string]int, ranks map[[2]string]int) *CLIPTokenizer {
	pat := regexp.MustCompile(`(?i)<\|startoftext\|>|<\|endoftext\|>|'s|'t|'re|'ve|'m|'ll|'d|[\p{L}]+|[\p{N}]+|[^\s\p{L}\p{N}]+`)

	return &CLIPTokenizer{
		encoder:     enc,
		bpeRanks:    ranks,
		cache:       make(map[string]string),
		pat:         pat,
		byteEncoder: bytesToUnicode(),
	}
}

func (t *CLIPTokenizer) Encode(text string) ([]int64, []int64) {
	text = html.UnescapeString(text)
	text = strings.ToLower(text)
	text = strings.Join(strings.Fields(text), " ")

	matches := t.pat.FindAllString(text, -1)

	tokens := make([]int, 0, 64)

	for _, m := range matches {
		be := t.encodeBytes([]byte(m))

		bpe := t.bpe(be)

		for _, part := range strings.Split(bpe, " ") {
			if id, ok := t.encoder[part]; ok {
				tokens = append(tokens, id)
			}
		}
	}

	ids := make([]int64, clipMaxTokens)
	att := make([]int64, clipMaxTokens)

	ids[0] = clipSOT
	att[0] = 1

	maxContent := clipMaxTokens - 2

	n := len(tokens)
	if n > maxContent {
		n = maxContent
	}

	for i := 0; i < n; i++ {
		ids[i+1] = int64(tokens[i])
		att[i+1] = 1
	}

	ids[n+1] = clipEOT
	att[n+1] = 1

	return ids, att
}

func (t *CLIPTokenizer) encodeBytes(b []byte) string {
	var sb strings.Builder

	sb.Grow(len(b))

	for _, by := range b {
		sb.WriteRune(t.byteEncoder[by])
	}

	return sb.String()
}

func (t *CLIPTokenizer) bpe(token string) string {
	if v, ok := t.cache[token]; ok {
		return v
	}

	word := make([]string, 0, utf8.RuneCountInString(token))

	for _, r := range token {
		word = append(word, string(r))
	}

	if len(word) > 0 {
		word[len(word)-1] += "</w>"
	}

	for {
		if len(word) < 2 {
			break
		}

		pairs := getPairs(word)

		bestRank := int(^uint(0) >> 1)

		var (
			best  [2]string
			found bool
		)

		for _, p := range pairs {
			if rk, ok := t.bpeRanks[p]; ok && rk < bestRank {
				bestRank = rk
				best = p

				found = true
			}
		}

		if !found {
			break
		}

		newWord := make([]string, 0, len(word))

		var i int

		for i < len(word) {
			j := indexOf(word, best[0], i)
			if j == -1 {
				newWord = append(newWord, word[i:]...)

				break
			}

			newWord = append(newWord, word[i:j]...)

			if j < len(word)-1 && word[j+1] == best[1] {
				newWord = append(newWord, best[0]+best[1])

				i = j + 2
			} else {
				newWord = append(newWord, word[j])

				i = j + 1
			}
		}

		word = newWord
	}

	out := strings.Join(word, " ")

	t.cache[token] = out

	return out
}

func getPairs(word []string) [][2]string {
	pairs := make([][2]string, 0, len(word)-1)

	for i := 0; i < len(word)-1; i++ {
		pairs = append(pairs, [2]string{word[i], word[i+1]})
	}

	return pairs
}

func indexOf(arr []string, s string, start int) int {
	for i := start; i < len(arr); i++ {
		if arr[i] == s {
			return i
		}
	}

	return -1
}

func bytesToUnicode() map[byte]rune {
	bs := make([]int, 0, 256)
	cs := make([]int, 0, 256)

	for b := int('!'); b <= int('~'); b++ {
		bs = append(bs, b)
		cs = append(cs, b)
	}

	for b := 0xA1; b <= 0xAC; b++ {
		bs = append(bs, b)
		cs = append(cs, b)
	}

	for b := 0xAE; b <= 0xFF; b++ {
		bs = append(bs, b)
		cs = append(cs, b)
	}

	var n int

	for b := 0; b < 256; b++ {
		if !containsInt(bs, b) {
			bs = append(bs, b)
			cs = append(cs, 256+n)

			n++
		}
	}

	m := make(map[byte]rune, 256)

	for i, b := range bs {
		m[byte(b)] = rune(cs[i])
	}

	return m
}

func containsInt(a []int, v int) bool {
	for _, x := range a {
		if x == v {
			return true
		}
	}

	return false
}

func fileExists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}

func loadMergesFile(path string) (map[[2]string]int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	ranks := make(map[[2]string]int)

	sc := bufio.NewScanner(file)

	var r int

	for sc.Scan() {
		ln := strings.TrimSpace(sc.Text())
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}

		parts := strings.SplitN(ln, " ", 2)
		if len(parts) != 2 {
			continue
		}

		ranks[[2]string{parts[0], parts[1]}] = r

		r++
	}

	err = sc.Err()
	if err != nil {
		return nil, err
	}

	return ranks, nil
}

func normalizeMerges(v any) ([]string, error) {
	switch vv := v.(type) {
	case []any:
		out := make([]string, 0, len(vv))

		for _, item := range vv {
			switch it := item.(type) {
			case string:
				out = append(out, it)
			case []any:
				if len(it) == 2 {
					a, aok := it[0].(string)
					b, bok := it[1].(string)

					if aok && bok {
						out = append(out, a+" "+b)
					}
				}
			default:
				// ignore
			}
		}

		return out, nil
	default:
		return nil, fmt.Errorf("unexpected merges type in tokenizer.json: %T", v)
	}
}
